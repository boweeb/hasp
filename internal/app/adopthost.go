package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boweeb/hasp/internal/adapter/sshconfig"
)

// AdoptHostRequest is `adopt host`'s input (tdd.md §9's `adopt` grid cell: "Wrap an existing
// hand-written stanza in hasp's markers", D7, D14). Pattern matches a domain.Host.Patterns entry
// exactly, mirroring ShowHost's own exact-match semantics (host.go) rather than find's fuzzy clue
// matching — `adopt host` targets one specific, already-identified stanza, the same precision
// `show host <pattern>` already assumes.
type AdoptHostRequest struct {
	KeyDir  string
	Pattern string
}

// AdoptHostUseCase wraps an existing hand-written Host stanza in ~/.ssh/config into hasp's marked
// region (D7, D14, tdd.md §9's `adopt` grid cell).
//
// Scope boundary — deliberate, not an oversight: adopt/release host, at this phase, operate only
// on stanzas in the default host group (~/.ssh/config itself, D7's harder co-owned case). A host
// stanza living in a custom, wholly-owned `~/.ssh/<group>.sshconfig` file (created by `new host
// --group=foo`) is never a candidate this Plan considers — that file is never read here at all,
// because the entire file is already hasp's own creation the moment it exists (D7's "the entire
// file is the region" for a group file hasp authors outright), and "adopt"/"release" for one
// stanza inside it isn't a coherent operation the roadmap or tdd.md specify one for. This is a
// real, deliberate scope narrowing (recorded here so a future reader knows it, not something to
// "fix"), and it is satisfied *structurally*: Plan below only ever opens configPath
// (KeyDir/config), so a stanza that only exists inside an Included custom group file is simply
// never found — the ordinary "no such host stanza" refusal already covers it, with no separate
// domain.Host.HostGroup check needed in this direction.
type AdoptHostUseCase struct{}

// Plan builds adopt host's plan. newhost.go's own Plan is this phase's reference implementation
// for how ~/.ssh/config is parsed, checked for T18 marker defects, and rendered back — reused
// directly below (findFirstTopLevelHostBlock, insertBeforeFirstHostBlock, newManagedRegion) rather
// than reimplemented.
//
// Unlike `new host`, nothing here authors new free-text content: the target HostBlock is an
// already-parsed, already-on-disk node relocated verbatim (its own Directives keep their original
// RawValue/Trivia/Terminator bytes exactly as parsed). req.Pattern itself is used only as a
// lookup key compared against that already-on-disk content — it is never spliced into a newly
// constructed Directive/RawValue/Include path — so none of newhost.go's validateDirectiveValue
// injection-guard calls apply here (see this phase's own exhaustive grep-based audit, cited in the
// implementation report).
func (uc AdoptHostUseCase) Plan(req AdoptHostRequest) (Plan, error) {
	if req.KeyDir == "" {
		return Plan{}, fmt.Errorf("%w: a key directory is required", ErrUsage)
	}
	if req.Pattern == "" {
		return Plan{}, fmt.Errorf("%w: a host pattern is required", ErrUsage)
	}

	configPath := filepath.Join(req.KeyDir, "config")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Plan{}, fmt.Errorf("%w: %s does not exist; there is no unmanaged host stanza to adopt", ErrUsage, configPath)
		}
		return Plan{}, fmt.Errorf("read %s: %w", configPath, err)
	}
	configFile := sshconfig.Parse(configBytes)

	// T18: a malformed marker means hasp cannot know where its own territory ends, so writing must
	// fail closed — identical reasoning and message shape to newhost.go's own check. This is a
	// data-integrity refusal, not a bad-flag refusal, so it does not wrap ErrUsage.
	if len(configFile.MarkerDefects) > 0 {
		return Plan{}, fmt.Errorf("%s has a malformed hasp marker (%d defect(s)); refusing to write until it is fixed", configPath, len(configFile.MarkerDefects))
	}

	idx := findHostBlockIndex(configFile.Nodes, req.Pattern)
	if idx == -1 {
		// Not found among top-level nodes — either it doesn't exist anywhere reachable from this
		// file, or it's already inside the region (already managed). Checking the region body too
		// costs nothing (configFile is already fully parsed in memory) and gives a clearer error.
		if region := findMarkedRegion(configFile.Nodes); region != nil && findHostBlockIndex(region.Body, req.Pattern) != -1 {
			return Plan{}, fmt.Errorf("%w: host pattern %q is already managed (inside hasp's marked region); nothing to adopt", ErrUsage, req.Pattern)
		}
		return Plan{}, fmt.Errorf("%w: no unmanaged host stanza matches pattern %q in %s", ErrUsage, req.Pattern, configPath)
	}
	hostBlock := configFile.Nodes[idx].(*sshconfig.HostBlock)

	// Splice the target out of wherever it sits, preserving order and bytes of everything else —
	// a plain slice removal, no mutation of surviving nodes.
	modifiedNodes := spliceOutNode(configFile.Nodes, idx)

	if regionIdx := findMarkedRegionIndex(modifiedNodes); regionIdx != -1 {
		region := modifiedNodes[regionIdx].(*sshconfig.MarkedRegion)
		// Appending a HostBlock at the end of an existing region body is always safe regardless of
		// what precedes it — unlike an Include (newhost.go's planIncludeChange doc comment), a
		// HostBlock's own header line closes any preceding block's scope on its own, so there is no
		// analogous hazard to insertBeforeFirstHostBlock guard against here.
		modifiedNodes[regionIdx] = &sshconfig.MarkedRegion{
			Begin: region.Begin,
			End:   region.End,
			Body:  append(append([]sshconfig.Node{}, region.Body...), hostBlock),
		}
	} else {
		// No region exists yet — this is the first-ever managed write to this file. Reuse newhost.go's
		// own findFirstTopLevelHostBlock/insertBeforeFirstHostBlock logic, applied to the node list
		// *after* the target block has already been removed: if the adopted stanza was the only bare
		// Host block, there's no anchor and the region lands at the end (insertBeforeFirstHostBlock's
		// own "or at the end" fallback); otherwise it lands before the first remaining one, exactly
		// like `new host`'s own bug-2 fix.
		modifiedNodes = insertBeforeFirstHostBlock(modifiedNodes, newManagedRegion([]sshconfig.Node{hostBlock}))
	}

	// Whole-file Before/After (rather than newhost.go's per-region span) sidesteps needing
	// region-only splicing here: spliceRegion's exact-match-in-current-bytes logic (change_
	// writeregion.go) handles a Before that's the entire file just as correctly as a smaller span,
	// since it's just bytes.Index — and it is guaranteed unique here because len(Before) ==
	// len(current) whenever they match at all.
	after := sshconfig.RenderNodes(modifiedNodes)

	witness, err := NewWitness(configPath)
	if err != nil {
		return Plan{}, err
	}

	return Plan{
		Summary:   fmt.Sprintf("adopt host %q", req.Pattern),
		Changes:   []Change{WriteRegion{File: configPath, Marker: defaultGroupRegion, Before: configBytes, After: after}},
		Witnesses: []Witness{witness},
	}, nil
}

// findHostBlockIndex returns the index in nodes of the first *sshconfig.HostBlock whose Header
// keyword is "Host" (case-insensitive) and whose Patterns contains pattern exactly — the same
// exact-match semantics ShowHost uses (host.go) and the same Host-vs-Match filter
// collectHostBlocks uses (bindings.go), reused here so a Match block's own criteria never
// accidentally satisfies a host-pattern lookup. Returns -1 if no match. Shared by
// AdoptHostUseCase (top-level search, and the region-body "already managed" check) and
// ReleaseHostUseCase (region-body search).
//
// If more than one distinct top-level Host stanza's Patterns happens to contain the same pattern
// token (legal, if unusual, in hand-written ssh_config), the first one in nodes (i.e. file) order
// wins, deterministically — this mirrors ssh_config's own first-obtained-value-wins precedence
// (T11) rather than being an arbitrary tie-break, so it is left as-is.
func findHostBlockIndex(nodes []sshconfig.Node, pattern string) int {
	for i, n := range nodes {
		hb, ok := n.(*sshconfig.HostBlock)
		if !ok || !strings.EqualFold(hb.Header.Keyword, "Host") {
			continue
		}
		for _, p := range hb.Patterns {
			if p == pattern {
				return i
			}
		}
	}
	return -1
}

// findMarkedRegion returns the sole *sshconfig.MarkedRegion in nodes, or nil if none — parse.go's
// own invariant is at most one MarkedRegion per file (scanMarkers only ever carves out one).
// Shared by AdoptHostUseCase and ReleaseHostUseCase.
func findMarkedRegion(nodes []sshconfig.Node) *sshconfig.MarkedRegion {
	for _, n := range nodes {
		if r, ok := n.(*sshconfig.MarkedRegion); ok {
			return r
		}
	}
	return nil
}

// findMarkedRegionIndex is findMarkedRegion's index-returning twin, needed wherever the caller
// must replace or splice around the region node itself rather than just read its Body. Shared by
// AdoptHostUseCase and ReleaseHostUseCase.
func findMarkedRegionIndex(nodes []sshconfig.Node) int {
	for i, n := range nodes {
		if _, ok := n.(*sshconfig.MarkedRegion); ok {
			return i
		}
	}
	return -1
}

// spliceOutNode returns a copy of nodes with the element at idx removed, preserving order and
// leaving every other node's own bytes (LeadingTrivia/Terminator, etc.) untouched — a plain slice
// splice with no mutation of surviving elements. Shared by AdoptHostUseCase and
// ReleaseHostUseCase.
func spliceOutNode(nodes []sshconfig.Node, idx int) []sshconfig.Node {
	out := make([]sshconfig.Node, 0, len(nodes)-1)
	out = append(out, nodes[:idx]...)
	out = append(out, nodes[idx+1:]...)
	return out
}
