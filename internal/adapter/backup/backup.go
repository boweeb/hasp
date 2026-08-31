package backup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// backupDirName is T8's own name: "~/.ssh/.hasp-backups/", relative to whatever key directory a
// Store is rooted at.
const backupDirName = ".hasp-backups"

// Store implements internal/app's BackupStore port. KeyDir is always injected explicitly, never
// defaulted (docs/tdd.md §12) — there is no fallback to ~/.ssh anywhere in this package.
type Store struct {
	KeyDir string
	Now    func() time.Time // injected clock; New sets this to time.Now
}

// New returns a Store rooted at keyDir.
func New(keyDir string) *Store {
	return &Store{KeyDir: keyDir, Now: time.Now}
}

// Snapshot copies path's current bytes into KeyDir/.hasp-backups/ as a timestamped file — T8's
// own example is "config.20260828T140501Z" — inheriting the source file's own mode. A source that
// does not exist is not an error: P4 requires the *prior* state be preserved before a destructive
// write, and a path with nothing at it has no prior state to preserve.
func (s *Store) Snapshot(path string) error {
	now := s.Now
	if now == nil {
		now = time.Now
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("backup: read %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("backup: stat %s: %w", path, err)
	}

	dir := filepath.Join(s.KeyDir, backupDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("backup: create %s: %w", dir, err)
	}

	dest, err := uniqueBackupPath(dir, filepath.Base(path), now())
	if err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	if err := os.WriteFile(dest, data, info.Mode()); err != nil {
		return fmt.Errorf("backup: write %s: %w", dest, err)
	}
	return nil
}

// uniqueBackupPath names a snapshot "<basename>.<timestamp>", T8's own convention, and guards
// against the timestamp-collision case — two snapshots of the same source within the same second
// — by appending an increasing suffix rather than silently overwriting a prior, still-wanted
// backup (T8: backups are never pruned or overwritten by hasp).
func uniqueBackupPath(dir, base string, at time.Time) (string, error) {
	stamp := at.UTC().Format("20060102T150405Z")
	candidate := filepath.Join(dir, base+"."+stamp)
	for i := 1; i <= 10000; i++ {
		_, err := os.Lstat(candidate)
		if errors.Is(err, fs.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("stat %s: %w", candidate, err)
		}
		candidate = filepath.Join(dir, fmt.Sprintf("%s.%s.%d", base, stamp, i))
	}
	return "", fmt.Errorf("could not find a free backup name for %s under %s", base, dir)
}
