package cli_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// snapshotTree returns path -> content digest for every entry under root (files, directories,
// and symlinks alike, so a structural change — an added or removed entry — is caught even when
// no file's own bytes changed). This is the mechanical proof behind roadmap.md §3's exit
// criterion 4 and tdd.md §12's guard test: "hasp writes nothing outside the key directory."
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}

		if d.Type()&fs.ModeSymlink != 0 {
			target, linkErr := os.Readlink(path)
			if linkErr != nil {
				return linkErr
			}
			snap[rel] = "symlink:" + target
			return nil
		}
		if d.IsDir() {
			snap[rel] = "dir"
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(b)
		snap[rel] = "file:" + hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotTree(%s): %v", root, err)
	}
	return snap
}

// assertTreeUnchanged fails the test if before and after differ in any path or content.
func assertTreeUnchanged(t *testing.T, before, after map[string]string) {
	t.Helper()
	for path, sum := range before {
		got, ok := after[path]
		if !ok {
			t.Errorf("write-nothing guard: %s was removed", path)
			continue
		}
		if got != sum {
			t.Errorf("write-nothing guard: %s changed", path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			t.Errorf("write-nothing guard: %s was created", path)
		}
	}
}
