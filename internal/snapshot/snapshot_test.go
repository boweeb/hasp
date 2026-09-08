package snapshot

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatal(err)
	}
}

// TestSnapshot_FileDigestIncludesModeAndContent confirms a regular file's digest changes when
// either its bytes or its permission mode change — the property the package doc comment claims
// distinguishes this package from the two weaker, mode-blind implementations it replaced.
func TestSnapshot_FileDigestIncludesModeAndContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	writeFile(t, path, "secret", 0o600)

	base, err := Snapshot(dir)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	// Same bytes, different mode: digest must differ.
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	modeChanged, err := Snapshot(dir)
	if err != nil {
		t.Fatalf("Snapshot after chmod: %v", err)
	}
	if modeChanged["key"] == base["key"] {
		t.Errorf("digest did not change when mode changed: %q", base["key"])
	}
	if diff := Diff(base, modeChanged); len(diff) != 1 {
		t.Errorf("Diff after mode-only change = %v, want exactly 1 mismatch", diff)
	}

	// Restore mode, change bytes: digest must differ from the original too.
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, "different secret", 0o600)
	contentChanged, err := Snapshot(dir)
	if err != nil {
		t.Fatalf("Snapshot after content change: %v", err)
	}
	if contentChanged["key"] == base["key"] {
		t.Errorf("digest did not change when content changed: %q", base["key"])
	}
}

// TestSnapshot_DirDigestIsModeOnly confirms a directory's digest reflects only its permission
// mode, per the package doc comment ("a directory's digest covers only its permission mode").
func TestSnapshot_DirDigestIsModeOnly(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "profile")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	snap, err := Snapshot(dir)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	want := "dir:-rwxr-xr-x"
	if got := snap["profile"]; got != want {
		t.Errorf("dir digest = %q, want %q", got, want)
	}
}

// TestSnapshot_SymlinkDigestIsTarget confirms a symlink's digest is exactly its target string —
// so a symlink retargeted to a different path is detected even though nothing else about it
// (its own mode, its existence) changed.
func TestSnapshot_SymlinkDigestIsTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	writeFile(t, target, "irreplaceable", 0o600)
	link := filepath.Join(dir, "alias")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	snap, err := Snapshot(dir)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	want := "symlink:" + target
	if got := snap["alias"]; got != want {
		t.Errorf("symlink digest = %q, want %q", got, want)
	}
}

// TestDiff_ThreeMismatchKinds confirms Diff reports each of the three ways two snapshots can
// disagree — a path removed, a path created, and a path whose digest changed — and that an
// identical pair of snapshots produces a nil/empty diff.
func TestDiff_ThreeMismatchKinds(t *testing.T) {
	before := map[string]string{
		"removed": "file:-rw-------:aaa",
		"changed": "file:-rw-------:bbb",
		"same":    "file:-rw-------:ccc",
	}
	after := map[string]string{
		"changed": "file:-rw-------:zzz",
		"same":    "file:-rw-------:ccc",
		"created": "file:-rw-------:ddd",
	}

	diff := Diff(before, after)
	if len(diff) != 3 {
		t.Fatalf("Diff = %v, want exactly 3 mismatches", diff)
	}

	var sawRemoved, sawCreated, sawChanged bool
	for _, m := range diff {
		switch m {
		case "removed was removed":
			sawRemoved = true
		case "created was created":
			sawCreated = true
		case "changed changed: before=file:-rw-------:bbb after=file:-rw-------:zzz":
			sawChanged = true
		}
	}
	if !sawRemoved || !sawCreated || !sawChanged {
		t.Errorf("Diff missing an expected mismatch kind, got: %v", diff)
	}

	if diff := Diff(before, before); len(diff) != 0 {
		t.Errorf("Diff(x, x) = %v, want empty for identical snapshots", diff)
	}
}

// TestDiff_IsSorted confirms the package doc comment's promise that Diff's result is already
// sorted, so a caller need not sort it again before printing or asserting on it.
func TestDiff_IsSorted(t *testing.T) {
	before := map[string]string{"z": "1", "a": "1", "m": "1"}
	after := map[string]string{}

	diff := Diff(before, after)
	want := []string{"a was removed", "m was removed", "z was removed"}
	if len(diff) != len(want) {
		t.Fatalf("Diff = %v, want %v", diff, want)
	}
	for i, w := range want {
		if diff[i] != w {
			t.Errorf("Diff[%d] = %q, want %q (result not sorted)", i, diff[i], w)
		}
	}
}
