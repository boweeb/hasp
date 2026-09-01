package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestMalformedMarker_ReadableButUnwritable is roadmap.md §5 exit criterion 4 (T18): a file with
// a malformed marker pair is readable — findings reported, defects flagged by kind and line, and
// any host stanzas that parsed cleanly outside the malformed region still correctly surfaced — and
// unwritable — every write verb refuses, fail-closed, before a single byte is touched. Both
// stances are proven against the same fixture in one test, per T18's own amendment to T2: parsing
// never fails (a malformed marker is data, not a Parse error), but a non-empty MarkerDefects makes
// writes to that file fail closed.
//
// The fixture is an unmatched begin marker with no matching end (region_test.go's own
// TestScanMarkers_UnmatchedBegin shape) followed by a Host stanza that, per Parse's own fallback
// when beginIdx/endIdx don't both resolve (no region is carved at all — the whole file parses as
// flat top-level nodes), still parses cleanly as an ordinary top-level HostBlock: exactly the
// "readable" half of T18's asymmetry, proven on content real enough that a naive implementation
// dropping everything after a malformed marker would fail this test.
func TestMalformedMarker_ReadableButUnwritable(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	fixture := "# >>> hasp:managed >>>\nHost clean\n    HostName clean.example.com\n"
	writeFile(t, configPath, fixture)

	// --- Readable: Derive/check succeed, the defect is flagged by kind and line, and the host
	// stanza that parsed cleanly outside the malformed marker is still reported. ---
	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v (a malformed marker must not abort a read, T18/P5)", err)
	}
	if findHostByPattern(m.Hosts, "clean") == nil {
		t.Fatalf("host %q, cleanly parsed outside the malformed region, is missing from Derive's output: %+v", "clean", m.Hosts)
	}

	findings := Check(m)
	var defect *Finding
	for i, f := range findings {
		if f.ID == FindingMarkerDefect {
			defect = &findings[i]
			break
		}
	}
	if defect == nil {
		t.Fatalf("no marker-defect finding produced: %+v", findings)
	}
	if kind, _ := defect.Detail["kind"].(string); kind != "unmatched-begin" {
		t.Errorf("marker-defect kind = %v, want %q: %+v", defect.Detail["kind"], "unmatched-begin", defect.Detail)
	}
	if line, _ := defect.Detail["line"].(int); line != 1 {
		t.Errorf("marker-defect line = %v, want 1: %+v", defect.Detail["line"], defect.Detail)
	}
	if defect.Severity != SeverityError {
		t.Errorf("marker-defect severity = %v, want error (T29)", defect.Severity)
	}

	// --- Unwritable: new host refuses against the same file, fail-closed, before any byte is
	// written. ---
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	uc := NewHostUseCase{}
	if _, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"anything"}}); err == nil {
		t.Fatal("new host succeeded against a file with a malformed marker, want a fail-closed refusal (T18)")
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("file changed after a refused write:\nbefore: %q\nafter:  %q", before, after)
	}
	if !bytes.Equal(after, []byte(fixture)) {
		t.Fatalf("file = %q, want the original fixture untouched: %q", after, fixture)
	}
}
