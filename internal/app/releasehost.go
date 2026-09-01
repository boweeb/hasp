package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boweeb/hasp/internal/adapter/sshconfig"
)

// ReleaseHostRequest is `release host`'s input — the exact inverse of AdoptHostRequest (D14,
// D18).
type ReleaseHostRequest struct {
	KeyDir  string
	Pattern string
}

// ReleaseHostUseCase moves a managed Host stanza out of ~/.ssh/config's marked region and
// re-inserts it as plain text immediately after that region — content-preserving, never deletes
// (D18, tdd.md §9's `release` grid cell). The inverse of AdoptHostUseCase; without it, adopt would
// be a one-way door (D14's own rationale for why release exists at all).
//
// Same scope boundary as AdoptHostUseCase (see its own doc comment for the full argument): only
// ~/.ssh/config's own marked region is ever read or written here, so a stanza that only exists
// inside a custom, wholly-owned `--group` file is never a candidate this Plan considers —
// structurally, because that file is never opened, not by a separate domain.Host.HostGroup check.
type ReleaseHostUseCase struct{}

// Plan builds release host's plan. Mirrors AdoptHostUseCase.Plan's own parse/T18 preamble; see
// D18 for why this relocates the stanza rather than deleting it, and why WriteRegion's own
// Before/After diff (change_writeregion.go's Preview) is sufficient to satisfy D15/D18's "show
// what's being removed first" requirement without a separate Remove-with-PriorContent mechanism
// (that mechanism is for the deletion case; this is a relocation).
func (uc ReleaseHostUseCase) Plan(req ReleaseHostRequest) (Plan, error) {
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
			return Plan{}, fmt.Errorf("%w: %s does not exist; there is no managed host stanza to release", ErrUsage, configPath)
		}
		return Plan{}, fmt.Errorf("read %s: %w", configPath, err)
	}
	configFile := sshconfig.Parse(configBytes)

	// T18: identical fail-closed reasoning and message shape to AdoptHostUseCase.Plan's own check.
	if len(configFile.MarkerDefects) > 0 {
		return Plan{}, fmt.Errorf("%s has a malformed hasp marker (%d defect(s)); refusing to write until it is fixed", configPath, len(configFile.MarkerDefects))
	}

	regionIdx := findMarkedRegionIndex(configFile.Nodes)
	if regionIdx == -1 {
		// No region at all means nothing in this file is managed — there is nothing to release.
		return Plan{}, fmt.Errorf("%w: %s has no managed region; host pattern %q is not currently managed", ErrUsage, configPath, req.Pattern)
	}
	region := configFile.Nodes[regionIdx].(*sshconfig.MarkedRegion)

	bodyIdx := findHostBlockIndex(region.Body, req.Pattern)
	if bodyIdx == -1 {
		return Plan{}, fmt.Errorf("%w: host pattern %q is not currently managed (not found in hasp's marked region)", ErrUsage, req.Pattern)
	}
	hostBlock := region.Body[bodyIdx].(*sshconfig.HostBlock)

	// Splice the stanza out of the region's body — preserving order and bytes of every other node
	// in it exactly. An empty (or nil) resulting Body is a legitimate, still-present region: D18
	// concerns the stanza's content, not the region's own existence.
	newRegion := &sshconfig.MarkedRegion{
		Begin: region.Begin,
		End:   region.End,
		Body:  spliceOutNode(region.Body, bodyIdx),
	}

	// Re-insert the released stanza as a top-level node immediately after the region (D18). It is
	// unmanaged the instant it lands, so P2 protects it again from this point on.
	modifiedNodes := make([]sshconfig.Node, 0, len(configFile.Nodes)+1)
	modifiedNodes = append(modifiedNodes, configFile.Nodes[:regionIdx]...)
	modifiedNodes = append(modifiedNodes, newRegion, hostBlock)
	modifiedNodes = append(modifiedNodes, configFile.Nodes[regionIdx+1:]...)

	// Whole-file Before/After, exactly as AdoptHostUseCase.Plan's own doc comment explains.
	after := sshconfig.RenderNodes(modifiedNodes)

	witness, err := NewWitness(configPath)
	if err != nil {
		return Plan{}, err
	}

	return Plan{
		Summary:   fmt.Sprintf("release host %q", req.Pattern),
		Changes:   []Change{WriteRegion{File: configPath, Marker: defaultGroupRegion, Before: configBytes, After: after}},
		Witnesses: []Witness{witness},
	}, nil
}
