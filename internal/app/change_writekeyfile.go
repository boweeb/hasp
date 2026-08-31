package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// WriteKeyFile authors private key bytes — the sole D14 exception to "hasp never writes to a
// private key file." AllowOverwrite is false for `new key`, always; true only for
// `edit key --replace-material` (T22), whose RequiresBackup() is unconditionally true so the
// material being replaced is snapshotted before Apply ever reaches the write that replaces it.
type WriteKeyFile struct {
	Path           string
	Contents       []byte
	Mode           fs.FileMode
	AllowOverwrite bool
}

// Preview always returns a Summary only, Diff == nil, regardless of AllowOverwrite — hasp does
// not render private key bytes to a terminal or a log, even bytes it is about to write itself
// (T26).
func (c WriteKeyFile) Preview() Preview {
	verb := "generate"
	if c.AllowOverwrite {
		verb = "replace"
	}
	return Preview{Summary: fmt.Sprintf("%s key file %s", verb, c.Path)}
}

// RequiresBackup mirrors AllowOverwrite exactly (T22): a fresh generation has no prior key
// material to preserve; a replace does, unconditionally.
func (c WriteKeyFile) RequiresBackup() bool { return c.AllowOverwrite }

func (c WriteKeyFile) backupPath() string { return c.Path }

// Apply fails closed — returns ErrKeyFileExists — the instant Path already exists, before a temp
// file is even written, when AllowOverwrite is false (T4, T22, D4). hasp never overwrites key
// material it did not just verify it authored in this same Plan.
func (c WriteKeyFile) Apply(fsys WriteFS) error {
	if !c.AllowOverwrite {
		if _, err := os.Lstat(c.Path); err == nil {
			return fmt.Errorf("%w: %s", ErrKeyFileExists, c.Path)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("check %s: %w", c.Path, err)
		}
	}
	return fsys.WriteFile(c.Path, c.Contents, c.Mode)
}
