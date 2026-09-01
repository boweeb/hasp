package app

import "fmt"

// Remove deletes an artifact hasp itself authored — a .hasp marker, an alias symlink (T22). It
// never applies to a key file: giving Change a general delete primitive would put the mechanism
// for destroying key material into the codebase, which is the opposite of how D4's canon is kept
// structural rather than disciplinary; restricting Remove's real-world callers to hasp-authored
// artifacts is a later use case's obligation (its constructor never reaches for Remove against a
// key path), not something this data type can enforce by its field shape alone.
type Remove struct {
	Path, Reason string

	// PriorContent, if non-nil, is the target's own bytes as they stood when a use case's Plan
	// captured them — a read performed by the caller building the Plan (P5), never by Remove
	// itself. When set, Preview() renders it as an all-removed diff (mirroring WriteRegion's own
	// Before/After -> lineDiff convention) so a renderer can show what is about to disappear before
	// Apply ever runs — release profile's use of this is D15 elaboration 4's own requirement:
	// "release must show the file's contents in its preview, so that removing a marker the user
	// has written in is never a silent loss." This is display only: Remove.Apply never inspects
	// PriorContent, and nothing about Remove reads the target to populate it — rendering a file's
	// raw bytes in a preview is not the same act as hasp parsing or acting on them, which is what
	// D15's presence-not-contents rule actually forbids (D15's own text: "hasp never parses,
	// validates, or reports on it" — a verbatim preview is none of those three).
	PriorContent []byte
}

func (c Remove) Preview() Preview {
	p := Preview{Summary: fmt.Sprintf("remove %s (%s)", c.Path, c.Reason)}
	if c.PriorContent != nil {
		p.Diff = lineDiff(c.PriorContent, nil)
	}
	return p
}

// RequiresBackup is always true (T4's own comment on this Change kind): nothing hasp deletes is
// deleted without a prior copy (P4).
func (c Remove) RequiresBackup() bool { return true }

func (c Remove) backupPath() string { return c.Path }

func (c Remove) Apply(fsys WriteFS) error {
	return fsys.Remove(c.Path)
}
