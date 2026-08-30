package scan

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// KeyCandidate is one file Keys found that might be a private key file — actual classification
// (is it really a key, and what kind) is keyfile.Inspect's job (T1), applied downstream by the
// app derivation pipeline. Directories, .pub sidecars, and .hasp markers are never candidates.
type KeyCandidate struct {
	Path         string // absolute path as discovered — the symlink's own path, if a symlink
	IsSymlink    bool
	ResolvedPath string // filepath.EvalSymlinks(Path); equals Path for a non-symlink
}

// Keys walks keyDir recursively — keys may live inside a profile directory at any depth (D13) —
// and returns every candidate file: every regular file and symlink except .pub sidecars and
// .hasp markers. A dangling symlink (its target does not exist) is skipped: fail-open, since
// scan's job is to gather what's readable (§11), not to report on what's broken — M1's finding
// set (T29) has no id for a dangling key alias specifically.
func Keys(keyDir string) ([]KeyCandidate, error) {
	var out []KeyCandidate

	err := filepath.WalkDir(keyDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil // fail-open: an unreadable entry mid-walk is skipped, never aborts the survey
		}
		if d.IsDir() {
			return nil
		}

		name := d.Name()
		if name == ".hasp" || strings.HasSuffix(name, ".pub") {
			return nil
		}

		isSymlink := d.Type()&fs.ModeSymlink != 0
		resolved := path
		if isSymlink {
			r, evalErr := filepath.EvalSymlinks(path)
			if evalErr != nil {
				return nil // dangling symlink target; skip
			}
			resolved = r
		}

		out = append(out, KeyCandidate{Path: path, IsSymlink: isSymlink, ResolvedPath: resolved})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
