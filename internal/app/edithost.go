package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boweeb/hasp/internal/adapter/scan"
	"github.com/boweeb/hasp/internal/adapter/sshconfig"
)

// EditHostRequest is `edit host`'s input (tdd.md §9's `edit` grid cell for `host`: "Change
// directives; move between groups (`--group`); rebind (`--key`)"). Unlike EditKeyRequest's four
// mutually exclusive sub-operations (editkey.go's resolveOp), edit host's three capabilities all
// culminate in the same underlying mechanic — locate the current stanza, build its replacement
// content, place it — and may be combined freely in one invocation (e.g. changing HostName and
// moving groups at once). There is deliberately no rename capability (unlike edit key's --name):
// the Host pattern itself is never changed by this verb (tdd.md §9's grid cell says so explicitly).
type EditHostRequest struct {
	KeyDir  string
	Pattern string // exact match against the target stanza's Host line patterns — same semantics
	// as adopt/release host's Pattern (findHostBlockIndex's own doc comment)

	// Directive changes. nil = leave untouched. Non-nil + non-empty = set/replace the directive's
	// value. Non-nil + empty string ("") = remove the directive if present (a no-op if absent).
	HostName *string
	User     *string
	Port     *string

	// Rebind (--key / --unbind). At most one of these may be set per invocation — both set is a
	// usage error. Key is a resolved location path (the CLI resolves a name-or-clue to this,
	// exactly as `new host`'s --key does via singleRealLocation) that becomes the new IdentityFile
	// value; Unbind removes the IdentityFile directive if present.
	Key    string
	Unbind bool

	// NewGroup, if non-nil, moves the stanza to a different host group: "" means the default group
	// (~/.ssh/config itself), a non-empty string means ~/.ssh/<NewGroup>.sshconfig (created if it
	// doesn't exist yet, exactly as `new host`'s --group does, D9). Nil means "leave in its current
	// group." A pointer, not a bare string, specifically so "move to the default group" (an empty
	// string value) is distinguishable from "no move requested" (nil) — the same explicit-presence
	// convention EditKeyRequest's own doc comment already establishes for exactly this reason.
	NewGroup *string
}

// EditHostUseCase implements `edit host` (tdd.md §9, D7, D9, T11, T18, T20, T30).
type EditHostUseCase struct{}

// locatedHost is where Plan found req.Pattern's managed stanza — either inside ~/.ssh/config's own
// MarkedRegion, or inside a custom, wholly-owned host-group file (D7: the entire file is the
// region there, no separate MarkedRegion to check). Exactly one of the two shapes is populated,
// selected by isRegion.
type locatedHost struct {
	hostBlock *sshconfig.HostBlock // the original, already-parsed node — its Header is reused
	// verbatim in the rebuilt replacement (no rename capability)
	isRegion bool
	bodyIdx  int // index of hostBlock within region.Body (isRegion) or groupParsed.Nodes (!isRegion)

	// Populated only when !isRegion: the custom group file's own resolved path, its bytes as read
	// by this use case (not scan.LoadConfigTree's own internal read — a fresh read this use case
	// controls, so its own Before/Witness (T30) trace back to a read it actually performed), and
	// that File's own parse.
	groupFile   string
	groupBytes  []byte
	groupParsed *sshconfig.File
}

// directiveFieldEdit pairs a directive keyword with the change requested for it: nil means leave
// every existing directive with that keyword untouched, "" means remove it, non-empty means
// set/replace it. Shared shape for HostName/User/Port/IdentityFile (identityWant, computed from
// Key/Unbind) so applyDirectiveFieldEdits below needs only one loop, not four near-identical ones.
type directiveFieldEdit struct {
	keyword string
	want    *string
}

// Plan builds `edit host`'s plan. The numbered steps below mirror this phase's own design notes
// (mirroring newhost.go/adopthost.go/releasehost.go's own convention of a heavily-commented,
// step-by-step Plan for a use case with real branching).
func (uc EditHostUseCase) Plan(req EditHostRequest) (Plan, error) {
	// Step 1: usage guards.
	if req.KeyDir == "" {
		return Plan{}, fmt.Errorf("%w: a key directory is required", ErrUsage)
	}
	if req.Pattern == "" {
		return Plan{}, fmt.Errorf("%w: a host pattern is required", ErrUsage)
	}
	if req.HostName == nil && req.User == nil && req.Port == nil && req.Key == "" && !req.Unbind && req.NewGroup == nil {
		return Plan{}, fmt.Errorf("%w: edit host requires at least one change", ErrUsage)
	}
	if req.Key != "" && req.Unbind {
		return Plan{}, fmt.Errorf("%w: --key and --unbind are mutually exclusive", ErrUsage)
	}

	// Every non-nil, non-empty directive value below is about to be spliced into a freshly built
	// Directive's RawValue (buildEditDirective) — validate before any Directive/HostBlock is
	// touched, the identical trust-boundary discipline newhost.go's Plan applies to its own
	// HostName/User/Port/IdentityFile fields (see validateDirectiveValue's own doc comment for the
	// injection this guards against).
	for _, f := range []struct {
		name  string
		value *string
	}{
		{"hostname", req.HostName},
		{"user", req.User},
		{"port", req.Port},
	} {
		if f.value != nil && *f.value != "" {
			if err := validateDirectiveValue(f.name, *f.value); err != nil {
				return Plan{}, err
			}
		}
	}
	if req.Key != "" {
		if err := validateDirectiveValue("identity file", req.Key); err != nil {
			return Plan{}, err
		}
	}

	var destFile string
	destRequested := req.NewGroup != nil
	if destRequested {
		var err error
		destFile, err = resolveHostGroupFile(req.KeyDir, *req.NewGroup)
		if err != nil {
			return Plan{}, err
		}
		// The identical extra check newhost.go's own Plan performs on groupFile: destFile is what
		// actually lands in an Include directive's RawValue if this move creates a brand-new
		// custom group (planCrossGroupMove below), so it needs the same directive-embedding
		// validation any KeyDir-derived path gets there — resolveHostGroupFile already validated
		// the group *name*, this validates the full resolved *path*.
		if err := validateDirectiveValue("key directory", destFile); err != nil {
			return Plan{}, err
		}
	}

	// Step 2/3: locate the current stanza. Always read ~/.ssh/config first: either it holds the
	// stanza directly (default group), or it's the root of the Include graph a custom group is
	// reachable from.
	configPath := filepath.Join(req.KeyDir, "config")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Plan{}, fmt.Errorf("%w: %s does not exist; there is no managed host stanza to edit", ErrUsage, configPath)
		}
		return Plan{}, fmt.Errorf("read %s: %w", configPath, err)
	}
	configFile := sshconfig.Parse(configBytes)

	// T18: a malformed marker means hasp cannot know where its own territory ends, so writing must
	// fail closed — identical reasoning and message shape to adopthost.go/releasehost.go's own
	// checks.
	if len(configFile.MarkerDefects) > 0 {
		return Plan{}, fmt.Errorf("%s has a malformed hasp marker (%d defect(s)); refusing to write until it is fixed", configPath, len(configFile.MarkerDefects))
	}
	region := findMarkedRegion(configFile.Nodes)

	loc, err := locateManagedHost(configPath, configFile, region, req.Pattern)
	if err != nil {
		return Plan{}, err
	}
	if loc == nil {
		// Give a genuinely useful error: distinguish "exists, but unmanaged" (a bare top-level
		// stanza in ~/.ssh/config) from "not found anywhere reachable" — mirroring the care
		// adopthost.go's own Plan takes distinguishing "not found" from "already managed".
		if findHostBlockIndex(configFile.Nodes, req.Pattern) != -1 {
			return Plan{}, fmt.Errorf("%w: host pattern %q exists but is not managed; use adopt host first", ErrUsage, req.Pattern)
		}
		return Plan{}, fmt.Errorf("%w: no managed host stanza matches pattern %q — use adopt host first if it exists unmanaged", ErrUsage, req.Pattern)
	}

	// Step 5: build the replacement HostBlock — original Header (unchanged, no rename), directives
	// surgically edited in place rather than regenerated wholesale (unlike new host's
	// buildNewHostBlock, this preserves every untouched directive's original raw bytes/position).
	var identityWant *string
	switch {
	case req.Key != "":
		v := req.Key
		identityWant = &v
	case req.Unbind:
		v := ""
		identityWant = &v
	}
	fields := []directiveFieldEdit{
		{"HostName", req.HostName},
		{"User", req.User},
		{"Port", req.Port},
		{"IdentityFile", identityWant},
	}
	rebuilt := &sshconfig.HostBlock{
		Header:     loc.hostBlock.Header,
		Patterns:   loc.hostBlock.Patterns,
		Directives: applyDirectiveFieldEdits(loc.hostBlock.Directives, fields),
	}

	// Step 4/6: determine the destination and apply. No move requested at all is trivially
	// in-place. A move requested to the file the stanza is already in is also in-place — a
	// deliberate no-op for the move (not an error, not pointless extra work), per this phase's own
	// design notes.
	if !destRequested {
		return planInPlace(req.Pattern, configPath, configBytes, configFile, region, loc, rebuilt)
	}

	currentFile := configPath
	if !loc.isRegion {
		currentFile = loc.groupFile
	}
	if filepath.Clean(destFile) == filepath.Clean(currentFile) {
		return planInPlace(req.Pattern, configPath, configBytes, configFile, region, loc, rebuilt)
	}

	return planCrossGroupMove(req.Pattern, configPath, configBytes, configFile, region, loc, rebuilt, *req.NewGroup == "", destFile)
}

// locateManagedHost finds pattern's managed stanza: first inside ~/.ssh/config's own marked region
// (region may be nil if none exists yet), then, if not found there, by walking every custom
// host-group file reachable via Include — mirroring newhost.go's own duplicate-pattern guard walk
// (existingHostPatterns, via scan.LoadConfigTree) but stopping at, and reporting, the first match
// rather than merely recording existence: edit host needs to know exactly where the stanza lives in
// order to rebuild and place it. Returns (nil, nil) if the pattern is not found managed anywhere —
// Plan itself decides, from that, whether to report "not found" or "found but unmanaged".
//
// A pattern matching stanzas in more than one place — the region and a custom group, or two
// distinct custom groups — resolves deterministically to whichever is checked first: the default
// region, then custom groups in scan.LoadConfigTree's own file-Include order (T11's own
// first-obtained-value-wins precedence order), mirroring findHostBlockIndex's identical tie-break
// within a single node list.
func locateManagedHost(configPath string, configFile *sshconfig.File, region *sshconfig.MarkedRegion, pattern string) (*locatedHost, error) {
	if region != nil {
		if idx := findHostBlockIndex(region.Body, pattern); idx != -1 {
			hb, ok := region.Body[idx].(*sshconfig.HostBlock)
			if !ok {
				return nil, fmt.Errorf("internal error: region.Body[%d] is not a *HostBlock", idx)
			}
			return &locatedHost{hostBlock: hb, isRegion: true, bodyIdx: idx}, nil
		}
	}

	configFiles, err := scan.LoadConfigTree(configPath)
	if err != nil {
		return nil, err
	}
	resolvedConfigPath, evalErr := filepath.EvalSymlinks(configPath)
	if evalErr != nil {
		resolvedConfigPath = configPath
	}
	for _, cf := range configFiles {
		if cf.Path == resolvedConfigPath || cf.Path == configPath {
			continue // the default group itself — already checked above (region, or unmanaged top level)
		}
		if findHostBlockIndex(cf.Body.Nodes, pattern) == -1 {
			continue
		}
		// Re-read and re-parse this group file directly (rather than reusing cf.Body, which
		// scan.LoadConfigTree parsed from its own internal read) so this use case controls its own
		// Before bytes and Witness (T30) precisely — the same discipline newhost.go's "group
		// already exists" branch follows (its own separate os.ReadFile + NewWitness call, distinct
		// from any earlier read of the same path).
		groupBytes, readErr := os.ReadFile(cf.Path)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", cf.Path, readErr)
		}
		groupParsed := sshconfig.Parse(groupBytes)
		idx := findHostBlockIndex(groupParsed.Nodes, pattern)
		if idx == -1 {
			continue // a benign race between LoadConfigTree's read and this one; keep looking
		}
		hb, ok := groupParsed.Nodes[idx].(*sshconfig.HostBlock)
		if !ok {
			return nil, fmt.Errorf("internal error: groupParsed.Nodes[%d] is not a *HostBlock", idx)
		}
		return &locatedHost{
			hostBlock:   hb,
			isRegion:    false,
			bodyIdx:     idx,
			groupFile:   cf.Path,
			groupBytes:  groupBytes,
			groupParsed: groupParsed,
		}, nil
	}

	return nil, nil
}

// applyDirectiveFieldEdits walks directives once and, for each node, checks whether it's a
// *Directive whose keyword matches one of fields' requested edits: untouched (nil want) leaves the
// node exactly as parsed (original RawValue/Trivia/Terminator bytes, original position); removal
// ("" want) drops the node; set/replace (non-empty want) substitutes a freshly built Directive at
// that same position. Only the *first* directive matching a given keyword is edited — a duplicate
// directive with the same keyword (legal, if unusual, in hand-authored ssh_config) is left
// untouched on any subsequent occurrence, mirroring findHostBlockIndex's own "first wins, rest
// preserved" discipline rather than silently deleting or overwriting something this verb was never
// asked to deduplicate. Any node that isn't a *Directive (a Line/comment, or a directive whose
// keyword isn't in fields at all) is always left completely untouched, in its original position.
// A requested field with no existing directive to replace is appended at the end, in fields' own
// order (HostName, User, Port, IdentityFile) — never reordering anything that was already there.
func applyDirectiveFieldEdits(directives []sshconfig.Node, fields []directiveFieldEdit) []sshconfig.Node {
	handled := map[string]bool{}
	out := make([]sshconfig.Node, 0, len(directives))
	for _, n := range directives {
		d, ok := n.(*sshconfig.Directive)
		if !ok {
			out = append(out, n)
			continue
		}
		var match *directiveFieldEdit
		for i := range fields {
			f := &fields[i]
			if f.want != nil && strings.EqualFold(d.Keyword, f.keyword) {
				match = f
				break
			}
		}
		if match == nil || handled[strings.ToLower(match.keyword)] {
			out = append(out, n)
			continue
		}
		handled[strings.ToLower(match.keyword)] = true
		if *match.want == "" {
			continue // removed
		}
		out = append(out, buildEditDirective(match.keyword, *match.want))
	}
	for _, f := range fields {
		if f.want != nil && *f.want != "" && !handled[strings.ToLower(f.keyword)] {
			out = append(out, buildEditDirective(f.keyword, *f.want))
		}
	}
	return out
}

// buildEditDirective constructs a freshly authored directive line, matching buildNewHostBlock's own
// construction style (newhost.go) exactly: 4-space indent, a leading space before the value, a
// hardcoded "\n" terminator. There is no prior human formatting to preserve for a line hasp itself
// authors inside territory it owns (D7 elaboration 1) — this is a surgical replacement of one
// directive's value, not an attempt to preserve the original directive's own raw formatting.
func buildEditDirective(keyword, value string) *sshconfig.Directive {
	return &sshconfig.Directive{
		LeadingTrivia: []byte("    "),
		Keyword:       keyword,
		RawValue:      []byte(" " + value),
		Terminator:    []byte("\n"),
	}
}

// planInPlace builds the Plan for an edit that never crosses a host-group boundary: the rebuilt
// HostBlock replaces the original at its own original index — never remove-and-append, which would
// silently reorder it relative to sibling stanzas and could change which stanza wins on an
// ambiguous/overlapping pattern (T11 first-match-wins; Phase 2's own regression test exists for
// exactly this class of concern, mirrored here). One WriteRegion Change on the one file the stanza
// already lives in.
func planInPlace(pattern, configPath string, configBytes []byte, configFile *sshconfig.File, region *sshconfig.MarkedRegion, loc *locatedHost, rebuilt *sshconfig.HostBlock) (Plan, error) {
	summary := fmt.Sprintf("edit host %q", pattern)

	if loc.isRegion {
		newBody := append([]sshconfig.Node{}, region.Body...)
		newBody[loc.bodyIdx] = rebuilt
		newRegion := &sshconfig.MarkedRegion{Begin: region.Begin, End: region.End, Body: newBody}
		modifiedNodes := append([]sshconfig.Node{}, configFile.Nodes...)
		modifiedNodes[findMarkedRegionIndex(modifiedNodes)] = newRegion
		after := sshconfig.RenderNodes(modifiedNodes)

		witness, err := NewWitness(configPath)
		if err != nil {
			return Plan{}, err
		}
		return Plan{
			Summary:   summary,
			Changes:   []Change{WriteRegion{File: configPath, Marker: defaultGroupRegion, Before: configBytes, After: after}},
			Witnesses: []Witness{witness},
		}, nil
	}

	newNodes := append([]sshconfig.Node{}, loc.groupParsed.Nodes...)
	newNodes[loc.bodyIdx] = rebuilt
	after := sshconfig.RenderNodes(newNodes)

	witness, err := NewWitness(loc.groupFile)
	if err != nil {
		return Plan{}, err
	}
	return Plan{
		Summary:   summary,
		Changes:   []Change{WriteRegion{File: loc.groupFile, Marker: groupFileRegion, Before: loc.groupBytes, After: after}},
		Witnesses: []Witness{witness},
	}, nil
}

// planGroupFileRemoval builds the WriteRegion Change that removes loc's stanza from its own custom
// group file — loc must be the !isRegion shape. Shared by every cross-group-move branch whose
// source is a custom group file.
func planGroupFileRemoval(loc *locatedHost) (Change, Witness, error) {
	newNodes := spliceOutNode(loc.groupParsed.Nodes, loc.bodyIdx)
	after := sshconfig.RenderNodes(newNodes)
	witness, err := NewWitness(loc.groupFile)
	if err != nil {
		return nil, Witness{}, err
	}
	return WriteRegion{File: loc.groupFile, Marker: groupFileRegion, Before: loc.groupBytes, After: after}, witness, nil
}

// planRegionRemoval builds the WriteRegion Change that removes the stanza at bodyIdx from
// ~/.ssh/config's own marked region. Shared by every cross-group-move branch whose source is the
// default group and whose destination is a different, already-existing file (the brand-new-custom-
// group destination case merges this removal with the Include addition instead — see
// planCrossGroupMove's own doc comment for why).
func planRegionRemoval(configPath string, configBytes []byte, configFile *sshconfig.File, region *sshconfig.MarkedRegion, bodyIdx int) (Change, Witness, error) {
	newRegion := &sshconfig.MarkedRegion{Begin: region.Begin, End: region.End, Body: spliceOutNode(region.Body, bodyIdx)}
	modifiedNodes := append([]sshconfig.Node{}, configFile.Nodes...)
	modifiedNodes[findMarkedRegionIndex(modifiedNodes)] = newRegion
	after := sshconfig.RenderNodes(modifiedNodes)

	witness, err := NewWitness(configPath)
	if err != nil {
		return nil, Witness{}, err
	}
	return WriteRegion{File: configPath, Marker: defaultGroupRegion, Before: configBytes, After: after}, witness, nil
}

// planCrossGroupMove builds the Plan for an edit that relocates the stanza to a different host
// group — up to three files, per this phase's own design notes' exhaustive case enumeration.
//
// T20 "least recoverable last": in every branch, whatever creates or updates the *destination*
// comes before whatever removes the stanza from its *source* — so that any prefix of Changes that
// actually applies leaves the stanza present somewhere, never in neither place. destIsDefault and
// destFile together identify the destination unambiguously (destIsDefault implies destFile ==
// configPath, though this function never needs to compare them — Plan's own caller already decided
// which sub-case applies).
func planCrossGroupMove(pattern, configPath string, configBytes []byte, configFile *sshconfig.File, region *sshconfig.MarkedRegion, loc *locatedHost, rebuilt *sshconfig.HostBlock, destIsDefault bool, destFile string) (Plan, error) {
	summary := fmt.Sprintf("edit host %q", pattern)

	if destIsDefault {
		// Destination = the default group. loc must be the custom-group-file shape here: if the
		// source were configPath's own region, resolving to the default group would have made this
		// a same-file, in-place edit already (Plan's own no-op-move check above never reaches this
		// function in that case).
		//
		// Reuse newhost.go's own planDefaultGroupChange unmodified — it already handles both
		// "region exists -> append at the end (always safe, a HostBlock's own header line closes
		// any preceding block's scope)" and "region == nil -> create it fresh, anchored before any
		// remaining top-level bare Host block" (bug 2's fix) — exactly this destination's own two
		// sub-cases, and rebuilt (not a freshly *authored* block, but a relocated one) is what's
		// appended either way.
		destChange, destWitness, err := planDefaultGroupChange(configPath, region, configFile.Nodes, rebuilt)
		if err != nil {
			return Plan{}, err
		}
		srcChange, srcWitness, err := planGroupFileRemoval(loc)
		if err != nil {
			return Plan{}, err
		}

		changes := []Change{destChange, srcChange} // destination before source removal (T20)
		witnesses := []Witness{}
		if destWitness != nil {
			witnesses = append(witnesses, *destWitness)
		}
		witnesses = append(witnesses, srcWitness)
		return Plan{Summary: summary, Changes: changes, Witnesses: witnesses}, nil
	}

	// Destination = a custom host group.
	destExists, err := fileExistsForEdit(destFile)
	if err != nil {
		return Plan{}, err
	}

	if destExists {
		destBytes, readErr := os.ReadFile(destFile)
		if readErr != nil {
			return Plan{}, fmt.Errorf("read %s: %w", destFile, readErr)
		}
		destParsed := sshconfig.Parse(destBytes)
		after := sshconfig.RenderNodes(append(append([]sshconfig.Node{}, destParsed.Nodes...), rebuilt))
		destWitness, err := NewWitness(destFile)
		if err != nil {
			return Plan{}, err
		}
		destChange := WriteRegion{File: destFile, Marker: groupFileRegion, Before: destBytes, After: after}

		var srcChange Change
		var srcWitness Witness
		if loc.isRegion {
			srcChange, srcWitness, err = planRegionRemoval(configPath, configBytes, configFile, region, loc.bodyIdx)
		} else {
			srcChange, srcWitness, err = planGroupFileRemoval(loc)
		}
		if err != nil {
			return Plan{}, err
		}

		return Plan{
			Summary:   summary,
			Changes:   []Change{destChange, srcChange}, // destination before source removal (T20)
			Witnesses: []Witness{destWitness, srcWitness},
		}, nil
	}

	// Destination is a brand-new custom group file: create it exactly as `new host` does (header +
	// the rebuilt stanza), and point ~/.ssh/config's own region at it with one Include line (T11) —
	// reusing newhost.go's exact placement rule (an Include always lands before any HostBlock
	// already in the region body, never after; insertBeforeFirstHostBlock's own doc comment).
	//
	// T20 ordering: the (harmless, unreferenced-until-the-next-step) new group file is created
	// first; whatever touches ~/.ssh/config and the source removal — the steps that actually change
	// something a human already relies on — come after, in the order that leaves the stanza present
	// somewhere at every prefix.
	destChange := WriteRegion{
		File:   destFile,
		Marker: groupFileRegion,
		Before: nil,
		After:  append([]byte(groupFileHeader), sshconfig.RenderNodes([]sshconfig.Node{rebuilt})...),
	}
	changes := []Change{destChange}
	var witnesses []Witness

	if loc.isRegion {
		// MERGE, deliberate: configPath is both the Include-addition target and the removal
		// source. Two separate WriteRegion Changes against the same file would need the second
		// one's Before to reflect the first one's After, which WriteRegion.Apply does not model —
		// each Change re-reads the file fresh and matches its own Before against *current* bytes
		// (spliceRegion, T30's witness re-verification is Plan-wide, not sequential-within-Plan) —
		// so both edits are folded into one atomic whole-file rewrite instead: the stanza removed
		// from the region body, and the new Include line inserted (always before any HostBlock
		// already in the body), in a single WriteRegion Change.
		newBody := spliceOutNode(append([]sshconfig.Node{}, region.Body...), loc.bodyIdx)
		include := &sshconfig.Directive{Keyword: "Include", RawValue: []byte(" " + destFile), Terminator: []byte("\n")}
		newBody = insertBeforeFirstHostBlock(newBody, include)
		newRegion := &sshconfig.MarkedRegion{Begin: region.Begin, End: region.End, Body: newBody}
		modifiedNodes := append([]sshconfig.Node{}, configFile.Nodes...)
		modifiedNodes[findMarkedRegionIndex(modifiedNodes)] = newRegion
		after := sshconfig.RenderNodes(modifiedNodes)

		witness, witnessErr := NewWitness(configPath)
		if witnessErr != nil {
			return Plan{}, witnessErr
		}
		changes = append(changes, WriteRegion{File: configPath, Marker: defaultGroupRegion, Before: configBytes, After: after})
		witnesses = append(witnesses, witness)
	} else {
		includeChange, includeWitness, includeErr := planIncludeChange(configPath, region, configFile.Nodes, destFile)
		if includeErr != nil {
			return Plan{}, includeErr
		}
		changes = append(changes, includeChange)
		if includeWitness != nil {
			witnesses = append(witnesses, *includeWitness)
		}

		srcChange, srcWitness, srcErr := planGroupFileRemoval(loc)
		if srcErr != nil {
			return Plan{}, srcErr
		}
		changes = append(changes, srcChange)
		witnesses = append(witnesses, srcWitness)
	}

	return Plan{Summary: summary, Changes: changes, Witnesses: witnesses}, nil
}

// fileExistsForEdit reports whether path exists — a read, safe under P5 — used to decide whether a
// cross-group move's destination custom group file already exists or needs to be created fresh.
// Named distinctly from adoptkey.go's own hasSidecar (same shape, `os.Lstat` vs. `os.Stat` and a
// bool-only return) because this call site needs to distinguish "doesn't exist" from "some other
// stat error" rather than collapsing both to false.
func fileExistsForEdit(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}
