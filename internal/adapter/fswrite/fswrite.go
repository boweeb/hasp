package fswrite

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// FS implements internal/app's WriteFS port. It carries no state — every method operates purely
// on the paths it is given.
type FS struct{}

// New returns an FS ready to use.
func New() FS { return FS{} }

// WriteFile implements §11's write mechanics 1-3: atomic write, symlink-through resolution, and
// mode preservation.
func (fsys FS) WriteFile(path string, data []byte, mode fs.FileMode) error {
	target, existingMode, exists, err := resolveWriteTarget(path)
	if err != nil {
		return err
	}
	writeMode := mode
	if exists {
		writeMode = existingMode
	}
	return atomicWrite(target, data, writeMode, exists)
}

// resolveWriteTarget implements §11 point 2: if path is itself a symlink, the real write target
// is its resolved destination — the symlink itself is never replaced. It also reports whether a
// file already exists at the resolved target and, if so, its current mode, so WriteFile can
// preserve it (§11 point 3).
func resolveWriteTarget(path string) (target string, mode fs.FileMode, exists bool, err error) {
	lst, lerr := os.Lstat(path)
	switch {
	case lerr == nil && lst.Mode()&fs.ModeSymlink != 0:
		resolved, evalErr := filepath.EvalSymlinks(path)
		if evalErr != nil {
			return "", 0, false, fmt.Errorf("resolve symlink %s: %w", path, evalErr)
		}
		target = resolved
	case lerr == nil, errors.Is(lerr, fs.ErrNotExist):
		target = path
	default:
		return "", 0, false, fmt.Errorf("stat %s: %w", path, lerr)
	}

	info, statErr := os.Stat(target)
	switch {
	case statErr == nil:
		return target, info.Mode(), true, nil
	case errors.Is(statErr, fs.ErrNotExist):
		return target, 0, false, nil
	default:
		return "", 0, false, fmt.Errorf("stat %s: %w", target, statErr)
	}
}

// atomicWrite is §11 point 1, literally: a temp file in the same directory as target, fsync the
// temp file, os.Rename onto target, fsync the containing directory. preserveOwner mirrors mode
// preservation for ownership, best-effort (§11 point 3: "and ownership, where the process has
// permission to") — a failed chown is never fatal, since hasp is very often already running as
// the file's own owner, for whom re-chowning to the same uid/gid is a costless no-op.
func atomicWrite(target string, data []byte, mode fs.FileMode, preserveOwner bool) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".hasp-write-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file %s: %w", tmpPath, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("fsync temp file %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmpPath, err)
	}

	writeMode := mode
	if writeMode == 0 {
		writeMode = 0o600
	}
	if err := os.Chmod(tmpPath, writeMode); err != nil {
		return fmt.Errorf("chmod temp file %s: %w", tmpPath, err)
	}
	if preserveOwner {
		preserveOwnership(target, tmpPath)
	}

	if err := os.Rename(tmpPath, target); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpPath, target, err)
	}
	renamed = true

	if err := fsyncDir(dir); err != nil {
		return fmt.Errorf("fsync directory %s: %w", dir, err)
	}
	return nil
}

// preserveOwnership copies target's current uid/gid onto tmpPath before the rename that replaces
// target, so a write never silently reassigns ownership. Errors are deliberately swallowed: this
// is the "where the process has permission to" half of §11 point 3, not a hard requirement, and a
// process running as the file's own owner (the overwhelmingly common case for a single-user tool)
// needs no elevated privilege for it to succeed anyway.
func preserveOwnership(target, tmpPath string) {
	info, err := os.Lstat(target)
	if err != nil {
		return
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	_ = os.Chown(tmpPath, int(stat.Uid), int(stat.Gid))
}

// fsyncDir fsyncs dir's own directory entry so a preceding os.Rename into it is durable across a
// crash on Linux (§11 point 1) — easy to skip, easy to regret.
func fsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// Symlink implements WriteFS.Symlink: creates path as a new symlink to target, refusing if path
// already exists — WriteFS never silently replaces an existing filesystem entry with a symlink.
func (fsys FS) Symlink(path, target string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("symlink: %s already exists", path)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("symlink: stat %s: %w", path, err)
	}
	if err := os.Symlink(target, path); err != nil {
		return fmt.Errorf("symlink %s -> %s: %w", path, target, err)
	}
	return nil
}

// Remove implements WriteFS.Remove: deletes the file at path.
func (fsys FS) Remove(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

// Move implements WriteFS.Move: D4's move rule (§11 point 4), literally — copy from's bytes to
// to, verify the copy byte-for-byte, then unlink from. from is never unlinked first: every error
// return below happens before the final os.Remove, so from is guaranteed to still exist, readable
// and unchanged, at its original path whenever Move fails. Move refuses outright if to already
// exists, so a relocation never silently clobbers an unrelated file sitting at the destination.
func (fsys FS) Move(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		return fmt.Errorf("move %s to %s: destination already exists", from, to)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("move: stat %s: %w", to, err)
	}

	data, err := os.ReadFile(from)
	if err != nil {
		return fmt.Errorf("move: read %s: %w", from, err)
	}
	info, err := os.Stat(from)
	if err != nil {
		return fmt.Errorf("move: stat %s: %w", from, err)
	}

	if err := fsys.WriteFile(to, data, info.Mode()); err != nil {
		return fmt.Errorf("move: copy %s to %s: %w", from, to, err)
	}

	// Verification is byte-for-byte here: this primitive has no notion of "key" or "fingerprint"
	// — a plain byte comparison is the undecidable-case check T15 names, and is at least as
	// strong as a fingerprint comparison for any content, key or otherwise.
	written, err := os.ReadFile(to)
	if err != nil {
		return fmt.Errorf("move: verify %s: %w", to, err)
	}
	if !bytes.Equal(data, written) {
		return fmt.Errorf("move: %s does not match %s byte-for-byte after copy, refusing to remove source", to, from)
	}

	if err := os.Remove(from); err != nil {
		return fmt.Errorf("move: copy of %s verified at %s, but removing the source failed: %w", from, to, err)
	}
	return nil
}

// ReplaceWithSymlink implements WriteFS.ReplaceWithSymlink: `adopt`'s alias-preserving move
// (§11 point 4, T20). It refines Move's copy-verify-unlink sequence for the one case Move itself
// cannot serve: the terminal step must leave a top-level alias behind (D13), and creating that
// alias before the destination exists would risk os.Rename's replace-on-conflict semantics
// destroying the very file it's supposed to stand in for (§4 "Change ordering"). So the sequence
// is copy → verify → atomically replace oldPath with a symlink to newTarget, in that order, with
// the replace itself a single os.Rename — never a bare unlink followed by a separate symlink
// creation, which would leave a window where oldPath resolves to nothing at all.
func (fsys FS) ReplaceWithSymlink(oldPath, newTarget string) error {
	if _, err := os.Lstat(newTarget); err == nil {
		return fmt.Errorf("adopt: %s already exists", newTarget)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("adopt: stat %s: %w", newTarget, err)
	}

	data, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("adopt: read %s: %w", oldPath, err)
	}
	info, err := os.Stat(oldPath)
	if err != nil {
		return fmt.Errorf("adopt: stat %s: %w", oldPath, err)
	}

	if err := fsys.WriteFile(newTarget, data, info.Mode()); err != nil {
		return fmt.Errorf("adopt: copy %s to %s: %w", oldPath, newTarget, err)
	}

	// Verification, byte-for-byte, exactly as Move's own comment explains: this primitive has no
	// notion of "key" or "fingerprint," so a plain byte comparison is the undecidable-case check
	// T15 names, and is at least as strong as a fingerprint comparison for any content.
	written, err := os.ReadFile(newTarget)
	if err != nil {
		return fmt.Errorf("adopt: verify %s: %w", newTarget, err)
	}
	if !bytes.Equal(data, written) {
		return fmt.Errorf("adopt: %s does not match %s byte-for-byte after copy, refusing to replace %s", newTarget, oldPath, oldPath)
	}

	// Terminal step: build the replacement symlink under a scratch name in oldPath's own
	// directory, then os.Rename it onto oldPath. os.Rename replaces an existing target atomically,
	// so this single call both removes the now-redundant copy at oldPath and installs the alias —
	// oldPath is never unlinked on its own. Up to this point, every returned error leaves oldPath
	// completely untouched; from here on, a failure still leaves oldPath holding either its
	// original file (if the rename itself never ran) or the new alias (if it did) — never neither.
	dir := filepath.Dir(oldPath)
	tmpLink, err := reserveTempName(dir, ".hasp-adopt")
	if err != nil {
		return fmt.Errorf("adopt: %w", err)
	}
	if err := os.Symlink(newTarget, tmpLink); err != nil {
		return fmt.Errorf("adopt: create replacement symlink: %w", err)
	}
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tmpLink)
		}
	}()

	if err := os.Rename(tmpLink, oldPath); err != nil {
		return fmt.Errorf("adopt: copy of %s verified at %s, but replacing %s with an alias failed: %w", oldPath, newTarget, oldPath, err)
	}
	renamed = true

	if err := fsyncDir(dir); err != nil {
		return fmt.Errorf("adopt: fsync directory %s: %w", dir, err)
	}
	return nil
}

// ReplaceSymlinkWithFile implements WriteFS.ReplaceSymlinkWithFile: `release`'s mirror of
// ReplaceWithSymlink. path must currently be a symlink — the shape adopt leaves behind — or this
// refuses outright, before anything is read or written, exactly as Symlink refuses an existing
// path and Move refuses an existing destination: WriteFS never silently replaces a filesystem
// entry that isn't the shape its own contract expects.
func (fsys FS) ReplaceSymlinkWithFile(path, source string) error {
	lst, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("release: stat %s: %w", path, err)
	}
	if lst.Mode()&fs.ModeSymlink == 0 {
		return fmt.Errorf("release: %s is not a symlink, refusing to replace it", path)
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("release: read %s: %w", source, err)
	}
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("release: stat %s: %w", source, err)
	}

	// Unlike WriteFile, the destination here (path) already holds something on purpose — a
	// symlink to source — so the copy is verified in a scratch temp file *before* path is ever
	// touched, rather than verified after a live write the way Move's own comment describes for
	// a destination that didn't previously exist.
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".hasp-release-*")
	if err != nil {
		return fmt.Errorf("release: create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("release: write temp file %s: %w", tmpPath, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("release: fsync temp file %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("release: close temp file %s: %w", tmpPath, err)
	}
	if err := os.Chmod(tmpPath, info.Mode()); err != nil {
		return fmt.Errorf("release: chmod temp file %s: %w", tmpPath, err)
	}
	preserveOwnership(path, tmpPath) // best-effort, mirrors §11 point 3

	written, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("release: verify %s: %w", tmpPath, err)
	}
	if !bytes.Equal(data, written) {
		return fmt.Errorf("release: temp copy of %s does not match byte-for-byte, refusing to replace %s", source, path)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("release: replace %s with a copy of %s: %w", path, source, err)
	}
	renamed = true

	if err := fsyncDir(dir); err != nil {
		return fmt.Errorf("release: fsync directory %s: %w", dir, err)
	}
	return nil
}

// reserveTempName finds a name not currently in use in dir, suitable for a scratch symlink about
// to be renamed onto its final destination — os.CreateTemp guarantees the name was unique at
// least momentarily, and removing the placeholder file it created leaves that name free. There is
// an inherent, accepted TOCTOU gap between this call returning and os.Symlink using the name it
// returned, bounded by the same single-user, single-process trust model
// internal/app's requireWithinKeyDir doc comment already accepts for this codebase (P8).
func reserveTempName(dir, prefix string) (string, error) {
	f, err := os.CreateTemp(dir, prefix+"-*")
	if err != nil {
		return "", fmt.Errorf("reserve temp name in %s: %w", dir, err)
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close reserved temp file %s: %w", name, err)
	}
	if err := os.Remove(name); err != nil {
		return "", fmt.Errorf("remove reserved temp file %s: %w", name, err)
	}
	return name, nil
}
