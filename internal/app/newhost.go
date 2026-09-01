package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boweeb/hasp/internal/adapter/scan"
	"github.com/boweeb/hasp/internal/adapter/sshconfig"
)

// groupFileHeader is the verbatim two-line header `new host` writes at the top of a freshly
// created custom host-group file (D7's "explicit host groups... carries a single header comment,
// not a delimiter pair — the entire file is the region"). Unlike profileMarkerHeader, this is not
// a marker hasp checks for presence of — it's a note for the human who might open the file, since
// hasp itself always knows a group file is wholly its own the moment it created it.
const groupFileHeader = `# hasp:owned — this file is fully managed by hasp. Hand edits here are lost on the next write.
# See ~/.ssh/config for how it is included.
`

// groupFileRegion is the cosmetic RegionID used for a custom host group's own WriteRegion Change
// (the whole file is the region, D7) — distinct from the default group's "managed" marker ID
// purely so Preview's summary text reads sensibly either way.
const groupFileRegion RegionID = "file"

// defaultGroupRegion is the RegionID for ~/.ssh/config's own hasp:managed marked region.
const defaultGroupRegion RegionID = "managed"

// NewHostRequest is `new host`'s input (tdd.md §9's `new` grid cell, D9's host-group naming).
// Group == "" means the default host group — ~/.ssh/config itself, D9's "sole exception" to the
// <name>.sshconfig naming rule. HostName/User/Port/IdentityFile are each optional; an empty string
// means "omit this directive," not "write an empty value" (a host with no explicit binding is
// already a legitimate M1 shape).
type NewHostRequest struct {
	KeyDir       string
	Patterns     []string
	Group        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
}

// NewHostUseCase creates a new "Host ..." stanza, either inside ~/.ssh/config's own managed region
// (default group) or inside a custom host group file hasp owns wholly (--group, D7, D9, T11).
type NewHostUseCase struct{}

// Plan builds `new host`'s plan. See tdd.md §6/§9 and design.md §5.5 for the host-group semantics
// this implements; the numbered steps below mirror this phase's own design notes closely, since
// there is real branching here (default vs. custom group, group file present vs. absent) that a
// prose summary would only obscure.
func (uc NewHostUseCase) Plan(req NewHostRequest) (Plan, error) {
	if req.KeyDir == "" {
		return Plan{}, fmt.Errorf("%w: a key directory is required", ErrUsage)
	}
	var patterns []string
	for _, p := range req.Patterns {
		if p != "" {
			patterns = append(patterns, p)
		}
	}
	if len(patterns) == 0 {
		return Plan{}, fmt.Errorf("%w: at least one host pattern is required", ErrUsage)
	}

	// Bug 3 (this review round): patterns is spliced unescaped into the "Host ..." header's
	// RawValue (buildNewHostBlock, strings.Join(patterns, " ")) — the identical bug-1 trust
	// boundary HostName/User/Port/IdentityFile are validated against below, just missed on the
	// first pass. Validate every element individually, not the joined string: a Host line
	// legitimately carries multiple space-separated patterns (e.g. "work-* personal-laptop"), so
	// joining first would validate the wrong thing (a string that's supposed to contain spaces)
	// and would lose which element was bad in the error message.
	//
	// A bare space or tab *inside* one pattern element is deliberately left unrejected here: unlike
	// '\n'/'\r'/'\x00'/'#', whitespace is not a structurally dangerous byte in this context — it
	// can't forge a line boundary, a comment, or hasp's own end marker. It is a correctness footgun
	// (it silently fragments what the caller meant as one pattern into two ssh_config patterns on
	// the next Parse), not a P1 injection concern, so it stays out of scope for
	// validateDirectiveValue, which exists specifically to guard the directive-embedding trust
	// boundary.
	for _, p := range patterns {
		if err := validateDirectiveValue(fmt.Sprintf("host pattern %q", p), p); err != nil {
			return Plan{}, err
		}
	}

	groupFile, err := resolveHostGroupFile(req.KeyDir, req.Group)
	if err != nil {
		return Plan{}, err
	}
	// groupFile (KeyDir joined with the group's filename) is the identical bug-1 trust boundary
	// once req.Group != "": planIncludeChange below splices it verbatim into the "Include <path>"
	// directive's RawValue that lands in ~/.ssh/config's own managed region. KeyDir is not an
	// operator-typo-only surface — it defaults from os.UserHomeDir() (internal/cli/root.go), so
	// environment tampering reaches it too. Validate here, right after groupFile is computed and
	// before either branch below can use it, rather than validating req.KeyDir directly: groupFile
	// is the actual string that gets embedded, and this also covers the (currently unreachable but
	// not worth relying on staying that way) case where a future change starts using groupFile in
	// the Group == "" branch too.
	if err := validateDirectiveValue("key directory", groupFile); err != nil {
		return Plan{}, err
	}

	// Bug 1 (review finding): HostName/User/Port/IdentityFile are spliced unescaped into a
	// Directive's RawValue (buildNewHostBlock) and rendered verbatim with a hardcoded "\n"
	// Terminator (node.go's Directive.render). Validate every one of them here, at Plan()'s own
	// trust boundary, before any Directive or HostBlock is constructed from them — resolveHostGroupFile
	// above already does the equivalent for Group, since Group is validated as part of resolving its
	// own path.
	for _, field := range []struct{ name, value string }{
		{"hostname", req.HostName},
		{"user", req.User},
		{"port", req.Port},
		{"identity file", req.IdentityFile},
	} {
		if err := validateDirectiveValue(field.name, field.value); err != nil {
			return Plan{}, err
		}
	}

	// Always read ~/.ssh/config: either the new HostBlock (default group) or the new Include line
	// (custom group, first time only) lands in its marked region. A missing file is not an error
	// (mirrors scan.LoadConfigTree's own fail-open precedent) — `new host` must work against a
	// machine with no ~/.ssh/config yet.
	configPath := filepath.Join(req.KeyDir, "config")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return Plan{}, fmt.Errorf("read %s: %w", configPath, err)
		}
		configBytes = nil
	}
	configFile := sshconfig.Parse(configBytes)

	// T18: a malformed marker means hasp cannot know where its own territory ends, so writing must
	// fail closed. This is a data-integrity refusal, not a bad-flag refusal, so it does not wrap
	// ErrUsage.
	if len(configFile.MarkerDefects) > 0 {
		return Plan{}, fmt.Errorf("%s has a malformed hasp marker (%d defect(s)); refusing to write until it is fixed", configPath, len(configFile.MarkerDefects))
	}

	// P1: never silently create a stanza that's immediately shadowed or ambiguous. Walk every
	// config file reachable from ~/.ssh/config (mirrors bindings.go's ResolveHosts's own walk) and
	// refuse if any requested pattern is already spoken for anywhere.
	configFiles, err := scan.LoadConfigTree(configPath)
	if err != nil {
		return Plan{}, err
	}
	existing := existingHostPatterns(configFiles)
	for _, p := range patterns {
		if existing[p] {
			return Plan{}, fmt.Errorf("%w: host pattern %q already exists in the reachable configuration", ErrUsage, p)
		}
	}

	// At most one MarkedRegion per file (scanMarkers's own invariant) — a linear scan for it.
	var region *sshconfig.MarkedRegion
	for _, n := range configFile.Nodes {
		if r, ok := n.(*sshconfig.MarkedRegion); ok {
			region = r
			break
		}
	}

	hostBlock := buildNewHostBlock(patterns, req.HostName, req.User, req.Port, req.IdentityFile)

	var changes []Change
	var witnesses []Witness

	if req.Group == "" {
		change, witness, err := planDefaultGroupChange(configPath, region, configFile.Nodes, hostBlock)
		if err != nil {
			return Plan{}, err
		}
		changes = append(changes, change)
		if witness != nil {
			witnesses = append(witnesses, *witness)
		}
	} else {
		groupExists := false
		if _, statErr := os.Stat(groupFile); statErr == nil {
			groupExists = true
		} else if !os.IsNotExist(statErr) {
			return Plan{}, fmt.Errorf("check %s: %w", groupFile, statErr)
		}

		if !groupExists {
			// T20 "least recoverable last": create the (currently unreferenced, harmless) group
			// file first, then point ~/.ssh/config at it. The reverse order would leave
			// ~/.ssh/config's Include pointing at a file that doesn't exist yet if the second write
			// never ran.
			groupChange := WriteRegion{
				File:   groupFile,
				Marker: groupFileRegion,
				Before: nil,
				After:  append([]byte(groupFileHeader), sshconfig.RenderNodes([]sshconfig.Node{hostBlock})...),
			}
			changes = append(changes, groupChange)

			includeChange, witness, err := planIncludeChange(configPath, region, configFile.Nodes, groupFile)
			if err != nil {
				return Plan{}, err
			}
			changes = append(changes, includeChange)
			if witness != nil {
				witnesses = append(witnesses, *witness)
			}
		} else {
			// The Include line was already added by a prior `new host` call — only the group file
			// itself needs the new stanza appended.
			groupBytes, err := os.ReadFile(groupFile)
			if err != nil {
				return Plan{}, fmt.Errorf("read %s: %w", groupFile, err)
			}
			witness, err := NewWitness(groupFile)
			if err != nil {
				return Plan{}, err
			}
			witnesses = append(witnesses, witness)
			after := append(append([]byte{}, groupBytes...), sshconfig.RenderNodes([]sshconfig.Node{hostBlock})...)
			changes = append(changes, WriteRegion{File: groupFile, Marker: groupFileRegion, Before: groupBytes, After: after})
		}
	}

	return Plan{
		Summary:   fmt.Sprintf("create host %q", strings.Join(patterns, " ")),
		Changes:   changes,
		Witnesses: witnesses,
	}, nil
}

// planDefaultGroupChange builds the single WriteRegion Change for the default-group case: the new
// HostBlock joins ~/.ssh/config's own managed region, alongside whatever was already there.
// Appending at the very end of the body is always safe, unlike planIncludeChange's own careful
// insertion point: a HostBlock's own "Host ..." header line is itself what closes any preceding
// HostBlock's scope, so a new HostBlock never needs to be partitioned ahead of existing ones the
// way a new Include directive does. configNodes is only consulted (by planRegionChange) when
// region == nil — see planRegionChange's own doc comment for why (bug 2).
func planDefaultGroupChange(configPath string, region *sshconfig.MarkedRegion, configNodes []sshconfig.Node, hostBlock *sshconfig.HostBlock) (Change, *Witness, error) {
	return planRegionChange(configPath, region, configNodes, hostBlock, func(body []sshconfig.Node) []sshconfig.Node {
		return append(body, hostBlock)
	})
}

// planIncludeChange builds the WriteRegion Change that adds `Include <groupFile>` to
// ~/.ssh/config's managed region, for the first `new host --group=...` call against a given group.
// configNodes is only consulted (by planRegionChange) when region == nil — see planRegionChange's
// own doc comment for why (bug 2).
func planIncludeChange(configPath string, region *sshconfig.MarkedRegion, configNodes []sshconfig.Node, groupFile string) (Change, *Witness, error) {
	include := &sshconfig.Directive{
		Keyword:    "Include",
		RawValue:   []byte(" " + groupFile),
		Terminator: []byte("\n"),
	}
	return planRegionChange(configPath, region, configNodes, include, func(body []sshconfig.Node) []sshconfig.Node {
		// Must land before any HostBlock already in the body, never after: ssh_config has no way
		// back to top-level scope once a "Host ..." line has appeared with no other Host/Match line
		// to follow it (sshconfig's own scanMarkers doc comment) — an Include placed after an
		// existing HostBlock would be grouped into that HostBlock's own Directives on the next
		// Parse, silently scoping it to only that Host pattern matching, and scan.LoadConfigTree's
		// includeTargets deliberately never looks inside a HostBlock for an Include to follow
		// (that's not a shape hasp itself is ever supposed to write). This is what actually makes it
		// safe for the default group's own region to carry both Include lines and HostBlocks (M3):
		// the two kinds are kept partitioned, Includes always first.
		return insertBeforeFirstHostBlock(body, include)
	})
}

// planRegionChange builds the WriteRegion Change that adds newNode into ~/.ssh/config's own
// managed region, creating that region for the first time if necessary. insertBody controls where
// within an *existing* region's body newNode lands (append at the end for a HostBlock, or
// insertBeforeFirstHostBlock for an Include directive — see planIncludeChange's own doc comment);
// it is never consulted when region == nil, since a brand-new region's body is trivially just
// newNode on its own.
//
// Bug 2 (review finding): when region == nil, the naive fix — Before: nil, After:
// renderNewRegion([]sshconfig.Node{newNode}), relying on spliceRegion's empty-Before branch to
// append After at the end of the file's current bytes — is only safe if the file has no
// pre-existing *top-level* Host/Match block (D7/T18). If it does (the ordinary shape of a real
// ~/.ssh/config, and the single most likely first-use scenario for `new host` on an actual
// machine), the newly appended region lands textually *after* that block. scanMarkers's inHostBlock
// latch is deliberately sticky once armed by a pre-region Host/Match line — ssh_config has no way
// back to top-level scope within one parse pass — so hasp's own freshly-written begin/end markers
// get misclassified DefectNestedInHostBlock on the very next Parse, and hasp has silently locked
// itself out of the region it just created (T18: every subsequent write then fails closed).
//
// The fix (approach (a) from the review, chosen over refusing outright: it costs one more
// RenderNodes/NewWitness call, matches the shape of the problem exactly, and does not narrow what
// `new host` can do on a first real machine, which a refusal would): when configNodes' first
// top-level node is a *HostBlock (there is no region yet in this branch, so every node in
// configNodes is top-level by construction — Parse only ever carves a MarkedRegion out of the top
// level), anchor Before on that HostBlock's own exact rendered bytes and place the brand-new region
// immediately ahead of it in After. This mirrors insertBeforeFirstHostBlock's own reasoning (which
// already solves the analogous problem *inside* an existing region's body) applied at the
// whole-file level instead: the new region lands before the earliest point scanMarkers's latch
// could ever arm. spliceRegion's own exact-match, refuse-if-not-unique semantics are reused
// unmodified — Before is guaranteed to appear at least once (T2: Render(Parse(b)) == b outside a
// region, so the anchor's rendered bytes are exactly what's already in the file), and a Witness is
// captured here since this reads real on-disk content to build the Plan (T30), same as any other
// case where Before carries a non-empty anchor.
func planRegionChange(configPath string, region *sshconfig.MarkedRegion, configNodes []sshconfig.Node, newNode sshconfig.Node, insertBody func(body []sshconfig.Node) []sshconfig.Node) (Change, *Witness, error) {
	if region == nil {
		if anchor := findFirstTopLevelHostBlock(configNodes); anchor != nil {
			before := sshconfig.RenderNodes([]sshconfig.Node{anchor})
			after := append(renderNewRegion([]sshconfig.Node{newNode}), before...)
			w, err := NewWitness(configPath)
			if err != nil {
				return nil, nil, err
			}
			return WriteRegion{File: configPath, Marker: defaultGroupRegion, Before: before, After: after}, &w, nil
		}
		return WriteRegion{File: configPath, Marker: defaultGroupRegion, Before: nil, After: renderNewRegion([]sshconfig.Node{newNode})}, nil, nil
	}

	before := sshconfig.RenderNodes([]sshconfig.Node{region})
	body := insertBody(append([]sshconfig.Node{}, region.Body...))
	after := renderNewRegion(body)

	var witness *Witness
	if len(before) > 0 {
		w, err := NewWitness(configPath)
		if err != nil {
			return nil, nil, err
		}
		witness = &w
	}
	return WriteRegion{File: configPath, Marker: defaultGroupRegion, Before: before, After: after}, witness, nil
}

// insertBeforeFirstHostBlock inserts node into body immediately before body's first *HostBlock
// (or at the end, if body has none) — see planIncludeChange's own doc comment for why an Include
// directive must never land after a HostBlock within the same region body.
func insertBeforeFirstHostBlock(body []sshconfig.Node, node sshconfig.Node) []sshconfig.Node {
	idx := len(body)
	for i, n := range body {
		if _, ok := n.(*sshconfig.HostBlock); ok {
			idx = i
			break
		}
	}
	out := make([]sshconfig.Node, 0, len(body)+1)
	out = append(out, body[:idx]...)
	out = append(out, node)
	out = append(out, body[idx:]...)
	return out
}

// findFirstTopLevelHostBlock returns the first *HostBlock in nodes, or nil if there is none.
// Called only when region == nil (planRegionChange, bug 2's fix): Parse only ever carves a
// MarkedRegion out of the top level of a file (parse.go), never nests one inside another node, so
// when there is no MarkedRegion yet, every node reachable this way — including any Host/Match
// block already in a human-authored ~/.ssh/config — is by construction top-level, exactly the
// shape scanMarkers's inHostBlock latch (region.go) is armed by.
func findFirstTopLevelHostBlock(nodes []sshconfig.Node) *sshconfig.HostBlock {
	for _, n := range nodes {
		if hb, ok := n.(*sshconfig.HostBlock); ok {
			return hb
		}
	}
	return nil
}

// renderNewRegion wraps body in a freshly built MarkedRegion (whose Begin/End are byte-identical
// to what scanMarkers recognizes on the next Parse) and renders it.
func renderNewRegion(body []sshconfig.Node) []byte {
	region := &sshconfig.MarkedRegion{
		Begin: []byte(sshconfig.ManagedRegionBegin + "\n"),
		End:   []byte(sshconfig.ManagedRegionEnd + "\n"),
		Body:  body,
	}
	return sshconfig.RenderNodes([]sshconfig.Node{region})
}

// resolveHostGroupFile turns a short --group name into its file path (design.md §5.5, D9): "" is
// the default group, ~/.ssh/config itself; otherwise <keyDir>/<group>.sshconfig. "config" is
// rejected as a group name — that literal basename is reserved for the default group's own file,
// and allowing it as a --group value would let a custom group collide with (or be confused for)
// the default group hasp already treats specially.
func resolveHostGroupFile(keyDir, group string) (string, error) {
	if group == "" {
		return filepath.Join(keyDir, "config"), nil
	}
	if err := validatePathSegment("host group name", group); err != nil {
		return "", err
	}
	// Group is not only a filesystem path segment (validatePathSegment's job just above) but also,
	// once resolved, becomes literal text spliced into planIncludeChange's "Include <path>"
	// directive's RawValue — the identical bug-1 trust boundary HostName/User/Port/IdentityFile go
	// through below, so it needs both protections (validatePathSegment's path-segment shape checks,
	// and validateDirectiveValue's directive-embedding checks), not just one.
	if err := validateDirectiveValue("host group name", group); err != nil {
		return "", err
	}
	if group == "config" {
		return "", fmt.Errorf("%w: host group name %q is reserved for the default group's own file", ErrUsage, group)
	}
	return filepath.Join(keyDir, group+".sshconfig"), nil
}

// existingHostPatterns collects every Host pattern already present anywhere in configFiles
// (mirrors bindings.go's own collectHostBlocks walk, reused directly since newhost.go already
// lives in package app) — P1's guard against silently creating a shadowed or ambiguous stanza.
func existingHostPatterns(configFiles []scan.ConfigFile) map[string]bool {
	existing := map[string]bool{}
	for _, cf := range configFiles {
		for _, found := range collectHostBlocks(cf.Body.Nodes, false) {
			for _, p := range found.block.Patterns {
				existing[p] = true
			}
		}
	}
	return existing
}

// validateDirectiveValue rejects a value that would corrupt the region it lands in once rendered
// as a Directive's raw bytes (buildNewHostBlock's HostName/User/Port/IdentityFile, and
// resolveHostGroupFile's Group by way of planIncludeChange's own Include directive). A Directive's
// render (sshconfig/node.go) concatenates RawValue verbatim ahead of a hardcoded "\n" Terminator,
// so an embedded '\n' or '\r' becomes one or more literal new physical lines in the rendered
// file — including, if crafted, hasp's own end-marker sentinel, letting attacker-controlled text
// escape hasp's tracked region entirely (this review's bug 1, reproduced via
// --hostname $'evil\n# <<< hasp:managed <<<\n...'). A NUL byte is rejected for the same class of
// reason: undefined and dangerous once written into a text config file, and already this
// codebase's convention (validatePathSegment's own NUL check, newkey.go).
//
// An unquoted '#' is also rejected (this review's finding 4): it round-trips fine on the immediate
// Render, but sshconfig/parse.go's findUnquotedHash/splitTrailingComment reinterprets it as a
// trailing comment on the *next* Parse, silently truncating the value. Rejecting is chosen over
// double-quoting per ssh_config(5)'s quoting rule because sshconfig has no existing "quote a raw
// value for output" helper to reuse (only tokenizeArgs, which parses the reverse direction) —
// writing and maintaining one just for this would not be "nearly free" — and because these are
// short, identifier-like values (hostnames, usernames, ports, paths) where a literal '#' is
// vanishingly unlikely to be an intentional part of the value.
func validateDirectiveValue(field, value string) error {
	if strings.ContainsAny(value, "\n\r\x00") {
		return fmt.Errorf("%w: %s must not contain a newline, carriage return, or null byte", ErrUsage, field)
	}
	if strings.ContainsRune(value, '#') {
		return fmt.Errorf("%w: %s must not contain %q, which would be reinterpreted as a comment on the next parse", ErrUsage, field, "#")
	}
	return nil
}

// buildNewHostBlock constructs the "Host ..." stanza `new host` writes: the header line plus one
// 4-space-indented directive per non-empty optional field, in the fixed order HostName/User/
// Port/IdentityFile. Every line is synthesized fresh (no LeadingTrivia/Trivia to preserve, unlike
// a parsed Directive) since this is hasp authoring a line itself inside territory it owns (D7
// elaboration 1).
func buildNewHostBlock(patterns []string, hostName, user, port, identityFile string) *sshconfig.HostBlock {
	header := &sshconfig.Directive{
		Keyword:    "Host",
		RawValue:   []byte(" " + strings.Join(patterns, " ")),
		Terminator: []byte("\n"),
	}

	var directives []sshconfig.Node
	appendIfSet := func(keyword, value string) {
		if value == "" {
			return
		}
		directives = append(directives, &sshconfig.Directive{
			LeadingTrivia: []byte("    "),
			Keyword:       keyword,
			RawValue:      []byte(" " + value),
			Terminator:    []byte("\n"),
		})
	}
	appendIfSet("HostName", hostName)
	appendIfSet("User", user)
	appendIfSet("Port", port)
	appendIfSet("IdentityFile", identityFile)

	return &sshconfig.HostBlock{Header: header, Patterns: patterns, Directives: directives}
}
