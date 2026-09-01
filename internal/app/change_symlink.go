package app

import "fmt"

// CreateSymlink materializes an alias (D5) as a symlink. It never replaces an existing filesystem
// entry — WriteFS.Symlink fails closed if Path already exists.
type CreateSymlink struct{ Path, Target string }

func (c CreateSymlink) Preview() Preview {
	return Preview{Summary: fmt.Sprintf("create alias %s -> %s", c.Path, c.Target)}
}

// RequiresBackup is false: an alias is a pure creation, nothing existed at Path before it (Apply
// fails closed if that's not true), so there is no prior state to preserve.
func (c CreateSymlink) RequiresBackup() bool { return false }

func (c CreateSymlink) Apply(fsys WriteFS) error {
	return fsys.Symlink(c.Path, c.Target)
}
