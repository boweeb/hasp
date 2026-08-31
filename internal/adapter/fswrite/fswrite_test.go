package fswrite

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func inode(t *testing.T, path string) uint64 {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%s): %v", path, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("Sys() for %s did not return *syscall.Stat_t", path)
	}
	return stat.Ino
}

// TestWriteFile_SymlinkedTargetWrittenThrough is the named guard test (tdd.md §12, T15): a
// symlinked ~/.ssh/config is resolved and written through — the *symlink's own inode* (its
// directory entry at linkPath, as distinct from whatever it points at) must be unchanged
// afterward, not merely the resulting content correct. A buggy implementation could unlink the
// symlink and write a plain file at linkPath with the right bytes — content-correct, while having
// silently converted the user's dotfiles-managed config into an orphaned copy. A content-only
// assertion would not catch that; comparing linkPath's own Lstat inode before and after does.
func TestWriteFile_SymlinkedTargetWrittenThrough(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "dotfiles")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	realPath := filepath.Join(realDir, "config")
	if err := os.WriteFile(realPath, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(dir, "config")
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatal(err)
	}

	beforeLinkInode := inode(t, linkPath)

	fsys := New()
	if err := fsys.WriteFile(linkPath, []byte("rewritten\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// The symlink itself must be untouched: same directory entry, same inode, still a symlink,
	// still pointing at the same real path.
	afterLinkInode := inode(t, linkPath)
	if beforeLinkInode != afterLinkInode {
		t.Errorf("symlink inode changed: before=%d after=%d — the symlink was replaced, not written through", beforeLinkInode, afterLinkInode)
	}
	linkInfo, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat(%s): %v", linkPath, err)
	}
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is no longer a symlink after WriteFile", linkPath)
	}
	gotTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink(%s): %v", linkPath, err)
	}
	if gotTarget != realPath {
		t.Errorf("symlink target = %q, want %q", gotTarget, realPath)
	}

	// The write went through to the resolved real file.
	got, err := os.ReadFile(realPath)
	if err != nil {
		t.Fatalf("read %s: %v", realPath, err)
	}
	if string(got) != "rewritten\n" {
		t.Errorf("content = %q, want %q", got, "rewritten\n")
	}
}

func TestWriteFile_PreservesExistingMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	fsys := New()
	// Pass a different mode; the existing file's own mode must win.
	if err := fsys.WriteFile(path, []byte("rewritten\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want preserved 0600", info.Mode().Perm())
	}
}

func TestWriteFile_UsesSuppliedModeForNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "brand-new")

	fsys := New()
	if err := fsys.WriteFile(path, []byte("hello\n"), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640", info.Mode().Perm())
	}
}

func TestWriteFile_AtomicNoPartialFileOnFailure(t *testing.T) {
	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o700) })
	path := filepath.Join(readOnlyDir, "config")

	fsys := New()
	if err := fsys.WriteFile(path, []byte("hello\n"), 0o644); err == nil {
		t.Fatal("WriteFile succeeded against a read-only directory, want an error")
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("%s exists despite a failed write", path)
	}

	entries, err := os.ReadDir(readOnlyDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("readonly dir has %d entries, want 0 (no leftover temp file)", len(entries))
	}
}

func TestSymlink_RefusesExistingPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "alias")
	if err := os.WriteFile(path, []byte("already here"), 0o644); err != nil {
		t.Fatal(err)
	}

	fsys := New()
	if err := fsys.Symlink(path, target); err == nil {
		t.Fatal("Symlink succeeded against an existing path, want a refusal")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "already here" {
		t.Errorf("content = %q, want unchanged", got)
	}
}

func TestMove_RefusesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "from")
	to := filepath.Join(dir, "to")
	if err := os.WriteFile(from, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("dest"), 0o644); err != nil {
		t.Fatal(err)
	}

	fsys := New()
	if err := fsys.Move(from, to); err == nil {
		t.Fatal("Move succeeded against an existing destination, want a refusal")
	}
	if _, err := os.Lstat(from); err != nil {
		t.Errorf("source %s missing after refused move: %v", from, err)
	}
}
