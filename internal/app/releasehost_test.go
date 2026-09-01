package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/sshconfig"
)

// TestReleaseHostUseCase_Plan_SingleStanza_AppearsAsPlainTextAfterEmptyRegion covers required test
// 8: region has one stanza, release it — the stanza now appears as plain text immediately after
// the (now-empty) region, the region itself is still present with Body nil/empty, zero
// MarkerDefects, and the whole file round-trips.
func TestReleaseHostUseCase_Plan_SingleStanza_AppearsAsPlainTextAfterEmptyRegion(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := "# a human wrote this\n\n" +
		sshconfig.ManagedRegionBegin + "\n" +
		"Host solo\n    HostName solo.example.com\n    User deploy\n" +
		sshconfig.ManagedRegionEnd + "\n" +
		"\n# trailing comment\n"
	writeFile(t, configPath, original)

	uc := ReleaseHostUseCase{}
	plan, err := uc.Plan(ReleaseHostRequest{KeyDir: dir, Pattern: "solo"})
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
	parsed := sshconfig.Parse(written)
	if len(parsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %+v, want none", parsed.MarkerDefects)
	}

	region := soleMarkedRegion(t, parsed)
	if len(region.Body) != 0 {
		t.Errorf("region.Body = %+v, want empty (the only stanza was released)", region.Body)
	}

	// The stanza must now be a top-level node, positioned immediately after the region.
	var regionIdx = -1
	var releasedIdx = -1
	for i, n := range parsed.Nodes {
		if _, ok := n.(*sshconfig.MarkedRegion); ok {
			regionIdx = i
		}
		if hb, ok := n.(*sshconfig.HostBlock); ok && len(hb.Patterns) == 1 && hb.Patterns[0] == "solo" {
			releasedIdx = i
		}
	}
	if regionIdx == -1 {
		t.Fatal("no MarkedRegion found")
	}
	if releasedIdx == -1 {
		t.Fatal("released stanza not found as a top-level node")
	}
	if releasedIdx != regionIdx+1 {
		t.Errorf("released stanza at index %d, want immediately after the region (index %d)", releasedIdx, regionIdx+1)
	}

	if !bytes.HasPrefix(written, []byte("# a human wrote this\n\n")) {
		t.Errorf("leading hand-written content not preserved: %q", written[:min(len(written), 30)])
	}
	if !bytes.HasSuffix(written, []byte("\n# trailing comment\n")) {
		t.Errorf("trailing hand-written content not preserved: %q", written[max(0, len(written)-30):])
	}
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
}

// TestReleaseHostUseCase_Plan_EndMarkerAtEOF_NoTrailingNewline is Bug C's own regression test (M3
// close-out, this review round): the end marker line itself is the file's absolute last physical
// line, with no trailing newline — an unusual but reachable precondition (a hand-typed or
// already-malformed marker literally at EOF). newRegion.End is captured verbatim from the original
// parse (region.go's own doc comment: "the exact marker lines hasp wrote, verbatim including their
// terminator"), so it carries that same nil terminator forward; the released stanza is then inserted
// immediately after the region node in the top-level node list (File.Render()/RenderNodes has no
// separator logic of its own between two different top-level nodes — this junction lives outside
// MarkedRegion.render() entirely, so Bug B's own fix does not reach it). Before this fix, the
// released stanza's "Host solo" header glued directly onto the end marker line. Asserts zero
// MarkerDefects and that the released stanza is recovered as its own distinct top-level node.
func TestReleaseHostUseCase_Plan_EndMarkerAtEOF_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	// No trailing newline: the end marker line is the file's absolute last byte.
	original := sshconfig.ManagedRegionBegin + "\n" +
		"Host solo\n    HostName solo.example.com\n" +
		sshconfig.ManagedRegionEnd
	writeFile(t, configPath, original)

	uc := ReleaseHostUseCase{}
	plan, err := uc.Plan(ReleaseHostRequest{KeyDir: dir, Pattern: "solo"})
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

	region := soleMarkedRegion(t, parsed)
	if len(region.Body) != 0 {
		t.Errorf("region.Body = %+v, want empty (the only stanza was released)", region.Body)
	}

	released := findHostBlock(t, parsed.Nodes, "solo")
	if released == nil {
		t.Fatal("released stanza not found as a top-level node — it was likely glued onto the end marker line")
	}
	assertDirectiveValue(t, released, "HostName", "solo.example.com")

	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
}

// TestReleaseHostUseCase_Plan_TwoStanzas_OneReleasedOneStaysManaged covers required test 9: region
// has two stanzas, release one — the other remains inside the region, the released one appears
// after the region as plain text, and hand-written content elsewhere is preserved.
func TestReleaseHostUseCase_Plan_TwoStanzas_OneReleasedOneStaysManaged(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := "# top\n\n" +
		sshconfig.ManagedRegionBegin + "\n" +
		"Host stays\n    User carol\n" +
		"Host leaves\n    User dave\n" +
		sshconfig.ManagedRegionEnd + "\n" +
		"\n# bottom\n"
	writeFile(t, configPath, original)

	uc := ReleaseHostUseCase{}
	plan, err := uc.Plan(ReleaseHostRequest{KeyDir: dir, Pattern: "leaves"})
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

	region := soleMarkedRegion(t, parsed)
	if findHostBlock(t, region.Body, "stays") == nil {
		t.Error("stays stanza should remain inside the region")
	}
	if findHostBlock(t, region.Body, "leaves") != nil {
		t.Error("leaves stanza should no longer be inside the region")
	}

	var found bool
	for _, n := range parsed.Nodes {
		if hb, ok := n.(*sshconfig.HostBlock); ok && len(hb.Patterns) == 1 && hb.Patterns[0] == "leaves" {
			found = true
		}
	}
	if !found {
		t.Error("leaves stanza not found as a top-level (unmanaged) node")
	}

	if !bytes.HasPrefix(written, []byte("# top\n\n")) {
		t.Errorf("leading hand-written content not preserved: %q", written[:min(len(written), 20)])
	}
	if !bytes.HasSuffix(written, []byte("\n# bottom\n")) {
		t.Errorf("trailing hand-written content not preserved: %q", written[max(0, len(written)-20):])
	}
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
}

// TestReleaseHostUseCase_Plan_NotManaged_NoRegion covers part of required test 10: no region at
// all in the file.
func TestReleaseHostUseCase_Plan_NotManaged_NoRegion(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, "Host bare\n    HostName bare.example.com\n")

	_, err := (ReleaseHostUseCase{}).Plan(ReleaseHostRequest{KeyDir: dir, Pattern: "bare"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// TestReleaseHostUseCase_Plan_NotManaged_RegionExistsButLacksPattern covers the rest of required
// test 10: a region exists but does not contain the requested pattern.
func TestReleaseHostUseCase_Plan_NotManaged_RegionExistsButLacksPattern(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nHost other\n    User carol\n"+sshconfig.ManagedRegionEnd+"\n")

	_, err := (ReleaseHostUseCase{}).Plan(ReleaseHostRequest{KeyDir: dir, Pattern: "notpresent"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// TestReleaseHostUseCase_Plan_MissingConfigFile covers the "nothing managed at all" precondition
// when ~/.ssh/config doesn't exist.
func TestReleaseHostUseCase_Plan_MissingConfigFile(t *testing.T) {
	dir := t.TempDir()
	_, err := (ReleaseHostUseCase{}).Plan(ReleaseHostRequest{KeyDir: dir, Pattern: "anything"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// TestReleaseHostUseCase_Plan_FailsClosedOnMarkerDefect covers required test 11's release-side
// coverage: T18 fail-closed on a file with a malformed marker (same fixture shape as
// newhost_test.go's own TestNewHostUseCase_Plan_FailsClosedOnMarkerDefect).
func TestReleaseHostUseCase_Plan_FailsClosedOnMarkerDefect(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, sshconfig.ManagedRegionEnd+"\n")

	_, err := (ReleaseHostUseCase{}).Plan(ReleaseHostRequest{KeyDir: dir, Pattern: "anything"})
	if err == nil {
		t.Fatal("Plan succeeded against a malformed marker, want a refusal")
	}
}

// TestReleaseHostUseCase_Plan_RejectsEmptyRequest covers basic usage-error guards.
func TestReleaseHostUseCase_Plan_RejectsEmptyRequest(t *testing.T) {
	cases := []ReleaseHostRequest{
		{Pattern: "foo"}, // no KeyDir
		{KeyDir: "/tmp"}, // no Pattern
	}
	for i, req := range cases {
		if _, err := (ReleaseHostUseCase{}).Plan(req); !errors.Is(err, ErrUsage) {
			t.Errorf("case %d: err = %v, want ErrUsage", i, err)
		}
	}
}

// TestReleaseHostUseCase_Plan_WitnessCoverage covers required test 12 for the release side: the
// one path that reads real on-disk bytes captures a Witness on configPath.
func TestReleaseHostUseCase_Plan_WitnessCoverage(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nHost solo\n    User carol\n"+sshconfig.ManagedRegionEnd+"\n")

	plan, err := (ReleaseHostUseCase{}).Plan(ReleaseHostRequest{KeyDir: dir, Pattern: "solo"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Witnesses) != 1 || plan.Witnesses[0].Path != configPath {
		t.Fatalf("Witnesses = %+v, want exactly one on %s", plan.Witnesses, configPath)
	}
}
