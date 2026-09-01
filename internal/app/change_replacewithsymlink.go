package app

import "fmt"

// ReplaceWithSymlink is `adopt`'s alias-preserving move (tdd.md §4 "Change ordering", §11 point
// 4, T20). It is deliberately a single Change, not two kept in a careful order: §4's own prose
// walks through why the naive two-Change plan — [MoveFile, CreateSymlink], or its reversal —
// cannot be ordered safely (a move-then-symlink leaves the key briefly undiscoverable if the
// symlink step fails; a symlink-then-move risks os.Rename's replace-on-conflict semantics
// destroying the original file the symlink is meant to stand in for). §11 point 4's actual
// mechanism sidesteps the ordering problem entirely by making the terminal step a single atomic
// primitive — copy, verify, then replace the source with a symlink via one os.Rename — so there
// is never a reachable intermediate state where the naive plan's failure mode could occur. From
// is never bare-unlinked; every prefix of the underlying WriteFS.ReplaceWithSymlink call leaves a
// real, readable key at From.
type ReplaceWithSymlink struct{ From, To string }

func (c ReplaceWithSymlink) Preview() Preview {
	return Preview{Summary: fmt.Sprintf("adopt %s into %s, leaving a top-level alias", c.From, c.To)}
}

// RequiresBackup is always true (D4, P4): From is a file hasp did not just author in this Plan,
// and its prior state must be preserved before Apply's copy-verify-replace sequence runs, as a
// safety net independent of that sequence's own copy-before-replace guarantee — the same
// reasoning MoveFile's RequiresBackup gives.
func (c ReplaceWithSymlink) RequiresBackup() bool { return true }

func (c ReplaceWithSymlink) backupPath() string { return c.From }

func (c ReplaceWithSymlink) Apply(fsys WriteFS) error {
	return fsys.ReplaceWithSymlink(c.From, c.To)
}
