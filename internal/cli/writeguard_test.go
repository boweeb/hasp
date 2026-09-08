package cli_test

import (
	"testing"

	"github.com/boweeb/hasp/internal/snapshot"
)

// snapshotTree is a thin t.Helper() wrapper over internal/snapshot.Snapshot — the mechanical
// proof behind roadmap.md §3's exit criterion 4 and tdd.md §12's guard test: "hasp writes nothing
// outside the key directory." internal/app/adopt_release_roundtrip_test.go's own snapshotDir
// wraps the same shared package (not imported directly here: internal/cli imports internal/app,
// so the dependency can't run the other way), rather than each reimplementing the digest logic.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	snap, err := snapshot.Snapshot(root)
	if err != nil {
		t.Fatalf("snapshotTree(%s): %v", root, err)
	}
	return snap
}

// assertTreeUnchanged fails the test if before and after differ in any path or content.
func assertTreeUnchanged(t *testing.T, before, after map[string]string) {
	t.Helper()
	for _, mismatch := range snapshot.Diff(before, after) {
		t.Errorf("write-nothing guard: %s", mismatch)
	}
}
