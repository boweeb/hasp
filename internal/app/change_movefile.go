package app

import "fmt"

// MoveFile relocates a file hasp did not create, via D4's move rule (§11 point 4: copy, verify,
// then unlink the source — never unlink first). This is the mechanism behind a plain relocation
// (e.g. edit key --profile) with no alias required; adopt's alias-preserving refinement (§4
// "Change ordering", T20) is a later phase built on top of this same primitive, not implemented
// here.
type MoveFile struct{ From, To, Reason string }

func (c MoveFile) Preview() Preview {
	return Preview{Summary: fmt.Sprintf("move %s to %s (%s)", c.From, c.To, c.Reason)}
}

// RequiresBackup is always true: From is a file hasp did not just author in this Plan, and P4
// requires its prior state be preserved before Apply's copy-verify-unlink sequence runs, as a
// safety net independent of that sequence's own copy-before-unlink guarantee.
func (c MoveFile) RequiresBackup() bool { return true }

func (c MoveFile) backupPath() string { return c.From }

func (c MoveFile) Apply(fsys WriteFS) error {
	return fsys.Move(c.From, c.To)
}
