package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ProfileCandidate is one directory under the key directory that might be a profile — every
// directory is a candidate; Managed reflects only whether a .hasp marker file is present (D13,
// D15 — presence, never contents).
type ProfileCandidate struct {
	Dir     string   // absolute path
	Path    []string // segments relative to the key directory, e.g. ["work", "foobarco"]
	Managed bool
}

// Profiles walks keyDir recursively and returns every subdirectory as a candidate. The key
// directory itself is never a candidate — a profile is always a named, addressable sub-persona.
// A symlinked directory is not descended into (filepath.WalkDir does not follow symlinks) and is
// not reported as a candidate either, matching D5/D13: a profile is a real directory, never an
// alias.
func Profiles(keyDir string) ([]ProfileCandidate, error) {
	var out []ProfileCandidate

	err := filepath.WalkDir(keyDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() || path == keyDir {
			return nil
		}

		rel, relErr := filepath.Rel(keyDir, path)
		if relErr != nil {
			return nil
		}
		segments := strings.Split(rel, string(filepath.Separator))

		_, statErr := os.Stat(filepath.Join(path, ".hasp"))
		managed := statErr == nil

		out = append(out, ProfileCandidate{Dir: path, Path: segments, Managed: managed})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
