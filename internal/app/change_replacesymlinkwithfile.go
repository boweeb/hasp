package app

import "fmt"

// ReplaceSymlinkWithFile is `release`'s mirror of ReplaceWithSymlink (tdd.md §4, §11 point 4 read
// in reverse): it restores a real, independent file at Alias by copying Source's bytes there,
// verified byte-for-byte, replacing the symlink that currently occupies Alias — atomically, and
// without touching Source. Source's own removal (the now-redundant profile-directory copy) is a
// deliberately separate, later Change in release's Plan (Remove), ordered after this one per §4's
// rule that the least recoverable step goes last: if this Change fails, Alias still resolves
// through its original symlink to Source — exactly the pre-release (adopted) state, working, not
// damaged. If this Change succeeds but the later Remove fails, Alias already holds a real,
// complete, independent copy and the key remains fully discoverable; Source is merely a
// redundant, check-flaggable duplicate, never a broken or undiscoverable state.
type ReplaceSymlinkWithFile struct{ Alias, Source string }

func (c ReplaceSymlinkWithFile) Preview() Preview {
	return Preview{Summary: fmt.Sprintf("release %s: restore a real file at %s", c.Source, c.Alias)}
}

// RequiresBackup is always true (D4, P4): the data reachable through Alias is snapshotted before
// it is replaced, the same discipline every other Change touching a pre-existing filesystem entry
// follows. BackupStore.Snapshot reads through the symlink (it copies bytes, not link structure —
// see internal/adapter/backup), so what's preserved is a copy of the key material Alias currently
// resolves to, not the symlink entry itself; that's already redundant with Source, but it keeps
// this Change consistent with every other backed-up write rather than a special case.
func (c ReplaceSymlinkWithFile) RequiresBackup() bool { return true }

func (c ReplaceSymlinkWithFile) backupPath() string { return c.Alias }

func (c ReplaceSymlinkWithFile) Apply(fsys WriteFS) error {
	return fsys.ReplaceSymlinkWithFile(c.Alias, c.Source)
}
