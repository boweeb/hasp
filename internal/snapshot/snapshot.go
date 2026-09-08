// Package snapshot captures a directory tree's content as a comparable map, and reports
// human-readable differences between two captures — the mechanical proof behind every
// byte-identical / write-nothing guard in this codebase (roadmap.md §3 exit criterion 4, §4 exit
// criterion 1, §5.5 exit criterion 6). It has no *testing.T coupling: Snapshot and Diff are plain
// functions returning data, so a test caller wraps them in its own t.Helper()+t.Errorf glue, and
// a non-test caller (tools/cleanroom) uses them directly.
//
// This package consolidates what were three near-identical implementations before it existed:
// internal/app/adopt_release_roundtrip_test.go's snapshotDir/assertSnapshotsEqual,
// internal/cli/writeguard_test.go's snapshotTree/assertTreeUnchanged (internal/cli's own
// package cli_test file), and internal/cli/writeguard_alltree_test.go's
// snapshotTreeForGuard/assertTreeUnchangedForGuard (package cli itself, duplicated a second time
// within internal/cli purely because a package-cli file cannot see symbols defined in a
// package-cli_test file). internal/app's version included each file's permission mode in the
// digest; the other two didn't. This package keeps the stronger, mode-including form: a write
// that leaves a file's bytes untouched but flips its mode (e.g. a key file going from 0600 to
// 0644) is exactly the kind of regression a "write nothing" or "byte-identical" guard exists to
// catch, and migrating the weaker two callers onto the stronger digest costs nothing since none
// of them depend on the old, mode-blind digest string's exact shape.
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Snapshot returns path (relative to root) -> content digest for every entry under root: files,
// directories, and symlinks alike, so a structural change (an added or removed entry, a
// symlink-vs-file swap) is caught even when no file's own bytes changed. A regular file's digest
// covers both its permission mode and a SHA-256 of its contents; a directory's digest covers only
// its permission mode; a symlink's digest is its target.
func Snapshot(root string) (map[string]string, error) {
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

		info, lstatErr := os.Lstat(path)
		if lstatErr != nil {
			return lstatErr
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			target, linkErr := os.Readlink(path)
			if linkErr != nil {
				return linkErr
			}
			snap[rel] = "symlink:" + target
			return nil
		}
		if d.IsDir() {
			snap[rel] = "dir:" + info.Mode().Perm().String()
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(b)
		snap[rel] = "file:" + info.Mode().Perm().String() + ":" + hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("snapshot %s: %w", root, err)
	}
	return snap, nil
}

// Diff compares two Snapshot results and returns one human-readable description per mismatch: a
// path present in before but missing from after, a path present in after but not before, or a
// path whose digest changed between the two. A nil/empty return means the two snapshots are
// identical. The result is sorted, so it is directly usable as deterministic output (a caller
// does not need to sort it again before printing or asserting on it).
func Diff(before, after map[string]string) []string {
	var mismatches []string
	for path, sum := range before {
		got, ok := after[path]
		if !ok {
			mismatches = append(mismatches, fmt.Sprintf("%s was removed", path))
			continue
		}
		if got != sum {
			mismatches = append(mismatches, fmt.Sprintf("%s changed: before=%s after=%s", path, sum, got))
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			mismatches = append(mismatches, fmt.Sprintf("%s was created", path))
		}
	}
	sort.Strings(mismatches)
	return mismatches
}
