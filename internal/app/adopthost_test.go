package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/sshconfig"
)

// TestAdoptThenReleaseHost_RoundTrip_ByteIdentical is roadmap.md M3 exit criterion 2's mechanical
// proof, mirrored on adopt_release_roundtrip_test.go's own shape for keys: adopt a hand-written
// stanza, then release it, and ~/.ssh/config returns to byte-identical to where it started. This
// is the single most important test for this phase.
//
// The fixture pre-seeds an *empty* managed region, immediately followed by the bare Host stanza
// that will be adopted — the host-CST analog of adopt_release_roundtrip_test.go's own
// mkManagedProfileDir scaffolding (that test's round trip proves the *key file* returns
// identically, not that the profile directory's own .hasp marker disappears too). D18 requires
// release to leave a stanza's now-empty region in place rather than delete it (see
// ReleaseHostUseCase's own doc comment), so a byte-identical round trip is only achievable when
// the "before" state already reflects that same persistent, empty-region scaffolding — exactly
// what release restores the region to. Starting from *no* region at all is covered separately by
// TestAdoptHostUseCase_Plan_NoPriorRegion_CreatesRegionAtEOF, which does not expect the region to
// vanish again on release (it doesn't, by design).
func TestAdoptThenReleaseHost_RoundTrip_ByteIdentical(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := "# a human wrote this\n\n" +
		sshconfig.ManagedRegionBegin + "\n" +
		sshconfig.ManagedRegionEnd + "\n" +
		"Host foobarco-prod\n" +
		"    HostName foobarco.example.com\n" +
		"    User deploy\n"
	writeFile(t, configPath, original)

	applier := newHostApplier(dir)

	adoptPlan, err := (AdoptHostUseCase{}).Plan(AdoptHostRequest{KeyDir: dir, Pattern: "foobarco-prod"})
	if err != nil {
		t.Fatalf("adopt Plan: %v", err)
	}
	if _, err := applier.Apply(adoptPlan); err != nil {
		t.Fatalf("adopt Apply: %v", err)
	}

	adopted, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	if bytes.Equal(adopted, []byte(original)) {
		t.Fatal("adopt did not change the file at all — the stanza was not actually wrapped")
	}
	adoptedParsed := sshconfig.Parse(adopted)
	if len(adoptedParsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects after adopt = %+v, want none", adoptedParsed.MarkerDefects)
	}
	region := soleMarkedRegion(t, adoptedParsed)
	if findHostBlock(t, region.Body, "foobarco-prod") == nil {
		t.Fatal("adopted stanza not found inside the marked region")
	}

	releasePlan, err := (ReleaseHostUseCase{}).Plan(ReleaseHostRequest{KeyDir: dir, Pattern: "foobarco-prod"})
	if err != nil {
		t.Fatalf("release Plan: %v", err)
	}
	if _, err := applier.Apply(releasePlan); err != nil {
		t.Fatalf("release Apply: %v", err)
	}

	released, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	if !bytes.Equal(released, []byte(original)) {
		t.Fatalf("round-trip guard: config not byte-identical after adopt+release\nwant:\n%q\ngot:\n%q", original, released)
	}
}

// TestAdoptHostUseCase_Plan_NoPriorRegion_CreatesRegionAtEOF covers required test 2: no region
// exists yet, one pre-existing unmanaged Host stanza (the target) and no other bare Host blocks —
// the region is created at EOF, the stanza now lives inside it, and the whole file round-trips.
func TestAdoptHostUseCase_Plan_NoPriorRegion_CreatesRegionAtEOF(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := "# hand-written\nHost solo\n    HostName solo.example.com\n"
	writeFile(t, configPath, original)

	uc := AdoptHostUseCase{}
	plan, err := uc.Plan(AdoptHostRequest{KeyDir: dir, Pattern: "solo"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	if len(plan.Witnesses) != 1 {
		t.Fatalf("len(Witnesses) = %d, want 1 (configPath was read)", len(plan.Witnesses))
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	parsed := sshconfig.Parse(written)
	if len(parsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %+v, want none", parsed.MarkerDefects)
	}
	region := soleMarkedRegion(t, parsed)
	hb := soleHostBlock(t, region.Body)
	if len(hb.Patterns) != 1 || hb.Patterns[0] != "solo" {
		t.Errorf("Patterns = %v, want [solo]", hb.Patterns)
	}
	if !bytes.HasPrefix(written, []byte("# hand-written\n")) {
		t.Errorf("hand-written prefix not preserved: %q", written[:min(len(written), 30)])
	}
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
}

// TestAdoptHostUseCase_Plan_NoPriorRegion_PlacedBeforeRemainingBareHostBlock covers required test
// 3: the target plus another unrelated bare Host block that must remain unmanaged — after
// adopting the target, the new region is placed before the remaining bare block (mirroring
// newhost.go's own bug-2 regression test shape), and the remaining block stays unmanaged/untouched.
func TestAdoptHostUseCase_Plan_NoPriorRegion_PlacedBeforeRemainingBareHostBlock(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := "Host target\n    HostName target.example.com\n" +
		"Host bastion\n    HostName bastion.example.com\n"
	writeFile(t, configPath, original)

	uc := AdoptHostUseCase{}
	plan, err := uc.Plan(AdoptHostRequest{KeyDir: dir, Pattern: "target"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	parsed := sshconfig.Parse(written)
	if len(parsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %+v, want none (bug-2-style regression)", parsed.MarkerDefects)
	}

	// The region must appear before the remaining "Host bastion" block in Nodes order.
	var regionIdx, bastionIdx = -1, -1
	for i, n := range parsed.Nodes {
		switch v := n.(type) {
		case *sshconfig.MarkedRegion:
			regionIdx = i
		case *sshconfig.HostBlock:
			if len(v.Patterns) == 1 && v.Patterns[0] == "bastion" {
				bastionIdx = i
			}
		}
	}
	if regionIdx == -1 {
		t.Fatal("no MarkedRegion found")
	}
	if bastionIdx == -1 {
		t.Fatal("bastion HostBlock not found at top level — it must remain unmanaged")
	}
	if regionIdx > bastionIdx {
		t.Errorf("region (index %d) must come before the remaining bastion block (index %d)", regionIdx, bastionIdx)
	}

	region := soleMarkedRegion(t, parsed)
	if findHostBlock(t, region.Body, "target") == nil {
		t.Error("target stanza not found inside the region")
	}
	if !bytes.Contains(written, []byte("Host bastion\n    HostName bastion.example.com\n")) {
		t.Errorf("bastion block not preserved verbatim: %q", written)
	}
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
}

// TestAdoptHostUseCase_Plan_ExistingRegion_AppendsSecondStanza covers required test 4: a region
// already exists with one stanza in it; adopting a second, currently-unmanaged stanza lands both
// in the region, and hand-written content elsewhere in the file (comments, unrelated directives)
// is preserved byte-for-byte outside the affected spans.
//
// "second" is deliberately followed by a third, unrelated bare Host block ("third") rather than a
// bare trailing comment: the CST attributes trailing comment/blank lines with no following
// Host/Match header to whichever Host block precedes them (hostblock.go's groupHostBlocks — there
// is no way back to top-level scope once a Host line has appeared, mirroring real ssh_config
// semantics exactly). Without an anchor after it, "second"'s own trailing comment would correctly
// move into the region *with* "second" on adopt — not a bug, but not what this test means to
// exercise ("hand-written content elsewhere... preserved"), so "third" gives the trailing comment
// a scope of its own that adopting "second" never touches.
func TestAdoptHostUseCase_Plan_ExistingRegion_AppendsSecondStanza(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := "# top comment\n\n" +
		sshconfig.ManagedRegionBegin + "\n" +
		"Host existing\n    User carol\n" +
		sshconfig.ManagedRegionEnd + "\n" +
		"\n# a note about the second host\nHost second\n    HostName second.example.com\n" +
		"\nHost third\n    User frank\n" +
		"\n# trailing comment attached to third\n"
	writeFile(t, configPath, original)

	uc := AdoptHostUseCase{}
	plan, err := uc.Plan(AdoptHostRequest{KeyDir: dir, Pattern: "second"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Witnesses) != 1 {
		t.Fatalf("len(Witnesses) = %d, want 1", len(plan.Witnesses))
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	if !bytes.HasPrefix(written, []byte("# top comment\n\n")) {
		t.Errorf("leading hand-written content not preserved: %q", written[:min(len(written), 30)])
	}
	if !bytes.Contains(written, []byte("# a note about the second host\n")) {
		t.Error("comment that preceded the now-adopted stanza was not preserved")
	}
	if !bytes.Contains(written, []byte("Host third\n    User frank\n")) {
		t.Error("unrelated bare Host block (third) not preserved")
	}
	if !bytes.HasSuffix(written, []byte("\n# trailing comment attached to third\n")) {
		t.Errorf("trailing hand-written content not preserved: %q", written[max(0, len(written)-50):])
	}

	parsed := sshconfig.Parse(written)
	if len(parsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %+v, want none", parsed.MarkerDefects)
	}
	region := soleMarkedRegion(t, parsed)
	if findHostBlock(t, region.Body, "existing") == nil {
		t.Error("pre-existing region member (existing) missing after adopt")
	}
	if findHostBlock(t, region.Body, "second") == nil {
		t.Error("newly adopted stanza (second) missing from the region")
	}
	if findHostBlock(t, region.Body, "third") != nil {
		t.Error("unrelated bare Host block (third) must not have been swept into the region")
	}
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
}

// TestAdoptHostUseCase_Plan_PatternNotFound covers required test 5.
func TestAdoptHostUseCase_Plan_PatternNotFound(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, "Host somethingelse\n    HostName x.example.com\n")

	_, err := (AdoptHostUseCase{}).Plan(AdoptHostRequest{KeyDir: dir, Pattern: "nosuchhost"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// TestAdoptHostUseCase_Plan_AlreadyManaged covers required test 6: the target pattern already
// sits inside the marked region — ErrUsage, with a message distinguishing "already managed" from
// "not found."
func TestAdoptHostUseCase_Plan_AlreadyManaged(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nHost already\n    User carol\n"+sshconfig.ManagedRegionEnd+"\n")

	_, err := (AdoptHostUseCase{}).Plan(AdoptHostRequest{KeyDir: dir, Pattern: "already"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
	if !bytes.Contains([]byte(err.Error()), []byte("already managed")) {
		t.Errorf("error = %q, want it to distinguish already-managed from not-found", err.Error())
	}
}

// TestAdoptHostUseCase_Plan_ScopeBoundary_CustomGroupOnly covers required test 7: a host whose
// only appearance is inside a custom, wholly-owned --group file (never as a bare top-level stanza
// in ~/.ssh/config) is correctly rejected — ErrUsage — rather than silently doing nothing or
// crashing. This is the scope boundary documented on AdoptHostUseCase's own doc comment, satisfied
// structurally: Plan never opens the custom group file at all, so the pattern is simply never
// found, which is exactly the ordinary "not found" refusal already covers.
func TestAdoptHostUseCase_Plan_ScopeBoundary_CustomGroupOnly(t *testing.T) {
	dir := t.TempDir()
	groupFile := filepath.Join(dir, "work.sshconfig")
	writeFile(t, groupFile, "# hasp:owned\nHost custom-only\n    HostName custom.example.com\n")
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nInclude "+groupFile+"\n"+sshconfig.ManagedRegionEnd+"\n")

	_, err := (AdoptHostUseCase{}).Plan(AdoptHostRequest{KeyDir: dir, Pattern: "custom-only"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage (stanza only exists in a custom group file, out of adopt host's scope)", err)
	}
}

// TestAdoptHostUseCase_Plan_MissingConfigFile covers the "nothing to adopt" precondition when
// ~/.ssh/config doesn't exist at all.
func TestAdoptHostUseCase_Plan_MissingConfigFile(t *testing.T) {
	dir := t.TempDir()
	_, err := (AdoptHostUseCase{}).Plan(AdoptHostRequest{KeyDir: dir, Pattern: "anything"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// TestAdoptHostUseCase_Plan_FailsClosedOnMarkerDefect covers T18 (required test 11's adopt-side
// coverage): a malformed marker must refuse the whole Plan.
func TestAdoptHostUseCase_Plan_FailsClosedOnMarkerDefect(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, sshconfig.ManagedRegionEnd+"\n")

	_, err := (AdoptHostUseCase{}).Plan(AdoptHostRequest{KeyDir: dir, Pattern: "anything"})
	if err == nil {
		t.Fatal("Plan succeeded against a malformed marker, want a refusal")
	}
}

// TestAdoptHostUseCase_Plan_RejectsEmptyRequest covers basic usage-error guards.
func TestAdoptHostUseCase_Plan_RejectsEmptyRequest(t *testing.T) {
	cases := []AdoptHostRequest{
		{Pattern: "foo"}, // no KeyDir
		{KeyDir: "/tmp"}, // no Pattern
	}
	for i, req := range cases {
		if _, err := (AdoptHostUseCase{}).Plan(req); !errors.Is(err, ErrUsage) {
			t.Errorf("case %d: err = %v, want ErrUsage", i, err)
		}
	}
}

// TestAdoptHostUseCase_Plan_DuplicatePattern_FirstStanzaWins covers findHostBlockIndex's
// documented tie-break: two distinct top-level "Host shared" stanzas both list the pattern being
// adopted (legal, if unusual, in hand-written ssh_config — nothing requires Patterns to be
// globally unique). adopt host must deterministically target the first one in file order
// (T11's first-obtained-value-wins precedence), leaving the second untouched, unmanaged, and
// byte-identical.
func TestAdoptHostUseCase_Plan_DuplicatePattern_FirstStanzaWins(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	secondStanza := "Host shared\n    User second\n"
	original := "Host shared\n    User first\n" + secondStanza
	writeFile(t, configPath, original)

	uc := AdoptHostUseCase{}
	plan, err := uc.Plan(AdoptHostRequest{KeyDir: dir, Pattern: "shared"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	parsed := sshconfig.Parse(written)
	if len(parsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %+v, want none", parsed.MarkerDefects)
	}

	// The first stanza (User: first) must be the one now inside the region.
	region := soleMarkedRegion(t, parsed)
	adopted := soleHostBlock(t, region.Body)
	if got, ok := directiveValue(adopted, "User"); !ok || got != "first" {
		t.Errorf("adopted stanza's User = %q (ok=%v), want %q (first stanza in file order)", got, ok, "first")
	}

	// The second stanza (User: second) must remain top-level, unmanaged, and byte-unchanged.
	remaining := findHostBlock(t, parsed.Nodes, "shared")
	if remaining == nil {
		t.Fatal("second \"Host shared\" stanza not found at top level after adopt — it must remain unmanaged")
	}
	if got, ok := directiveValue(remaining, "User"); !ok || got != "second" {
		t.Errorf("remaining top-level stanza's User = %q (ok=%v), want %q (second stanza untouched)", got, ok, "second")
	}
	if !bytes.Contains(written, []byte(secondStanza)) {
		t.Errorf("second stanza not preserved byte-identically: %q", written)
	}
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
}

// TestAdoptHostUseCase_Plan_PreviewIsWindowedOnLargeFile covers the windowing fix (see diff.go's
// lineDiff/windowDiff): adopt host builds its WriteRegion Change with Before/After spanning the
// entire config file (this file's own doc comment, above, explains why), so on a real-sized config
// Preview().Diff must not degenerate into a near-whole-file dump of unchanged context — it must stay
// short, with the bulk of the unaffected file collapsed into DiffElided marker(s).
func TestAdoptHostUseCase_Plan_PreviewIsWindowedOnLargeFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	var b strings.Builder
	for i := 0; i < 75; i++ {
		fmt.Fprintf(&b, "Host server%d\n    HostName server%d.example.com\n", i, i)
	}
	fmt.Fprintf(&b, "Host target\n    HostName target.example.com\n")
	writeFile(t, configPath, b.String()) // 152 lines total

	plan, err := (AdoptHostUseCase{}).Plan(AdoptHostRequest{KeyDir: dir, Pattern: "target"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}

	diff := plan.Changes[0].Preview().Diff
	if len(diff) == 0 {
		t.Fatal("Preview().Diff is empty, want a real diff")
	}
	if len(diff) > 30 {
		t.Errorf("Preview().Diff has %d lines against a 152-line file, want a short windowed diff (an unfixed whole-file diff would be ~150+ lines)", len(diff))
	}

	var sawElided bool
	for _, l := range diff {
		if l.Kind == DiffElided {
			sawElided = true
		}
	}
	if !sawElided {
		t.Error("Preview().Diff has no DiffElided marker, want the large unchanged span collapsed")
	}
}

// TestAdoptHostUseCase_Plan_WitnessCoverage covers required test 12 for the adopt side: every path
// that reads real on-disk bytes (both the "region already exists" and "no region yet" shapes)
// captures a Witness on configPath.
func TestAdoptHostUseCase_Plan_WitnessCoverage(t *testing.T) {
	t.Run("no region yet", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config")
		writeFile(t, configPath, "Host solo\n    HostName solo.example.com\n")

		plan, err := (AdoptHostUseCase{}).Plan(AdoptHostRequest{KeyDir: dir, Pattern: "solo"})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		if len(plan.Witnesses) != 1 || plan.Witnesses[0].Path != configPath {
			t.Fatalf("Witnesses = %+v, want exactly one on %s", plan.Witnesses, configPath)
		}
	})

	t.Run("region already exists", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config")
		writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nHost existing\n    User carol\n"+sshconfig.ManagedRegionEnd+"\nHost second\n    User dave\n")

		plan, err := (AdoptHostUseCase{}).Plan(AdoptHostRequest{KeyDir: dir, Pattern: "second"})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		if len(plan.Witnesses) != 1 || plan.Witnesses[0].Path != configPath {
			t.Fatalf("Witnesses = %+v, want exactly one on %s", plan.Witnesses, configPath)
		}
	})
}
