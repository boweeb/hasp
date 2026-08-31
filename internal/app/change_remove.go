package app

import "fmt"

// Remove deletes an artifact hasp itself authored — a .hasp marker, an alias symlink (T22). It
// never applies to a key file: giving Change a general delete primitive would put the mechanism
// for destroying key material into the codebase, which is the opposite of how D4's canon is kept
// structural rather than disciplinary; restricting Remove's real-world callers to hasp-authored
// artifacts is a later use case's obligation (its constructor never reaches for Remove against a
// key path), not something this data type can enforce by its field shape alone.
type Remove struct{ Path, Reason string }

func (c Remove) Preview() Preview {
	return Preview{Summary: fmt.Sprintf("remove %s (%s)", c.Path, c.Reason)}
}

// RequiresBackup is always true (T4's own comment on this Change kind): nothing hasp deletes is
// deleted without a prior copy (P4).
func (c Remove) RequiresBackup() bool { return true }

func (c Remove) backupPath() string { return c.Path }

func (c Remove) Apply(fsys WriteFS) error {
	return fsys.Remove(c.Path)
}
