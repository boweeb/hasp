package fswrite

import (
	"os"
	"path/filepath"
	"testing"
)

// --- ReplaceWithSymlink (adopt) ---

func TestReplaceWithSymlink_HappyPath(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	if err := os.Mkdir(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(dir, "id_ed25519")
	newTarget := filepath.Join(profileDir, "id_ed25519")
	if err := os.WriteFile(oldPath, []byte("irreplaceable secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	fsys := New()
	if err := fsys.ReplaceWithSymlink(oldPath, newTarget); err != nil {
		t.Fatalf("ReplaceWithSymlink: %v", err)
	}

	// oldPath is now a symlink to newTarget.
	info, err := os.Lstat(oldPath)
	if err != nil {
		t.Fatalf("Lstat(%s): %v", oldPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink after ReplaceWithSymlink", oldPath)
	}
	gotTarget, err := os.Readlink(oldPath)
	if err != nil {
		t.Fatalf("Readlink(%s): %v", oldPath, err)
	}
	if gotTarget != newTarget {
		t.Errorf("symlink target = %q, want %q", gotTarget, newTarget)
	}

	// newTarget holds the original bytes, mode preserved.
	got, err := os.ReadFile(newTarget)
	if err != nil {
		t.Fatalf("read %s: %v", newTarget, err)
	}
	if string(got) != "irreplaceable secret" {
		t.Errorf("newTarget content = %q, want %q", got, "irreplaceable secret")
	}
	newInfo, err := os.Stat(newTarget)
	if err != nil {
		t.Fatal(err)
	}
	if newInfo.Mode().Perm() != 0o600 {
		t.Errorf("newTarget mode = %v, want preserved 0600", newInfo.Mode().Perm())
	}

	// Reading through the alias reaches the same bytes (ssh's own default-identity probing depends
	// on exactly this).
	gotThroughAlias, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatalf("read through alias %s: %v", oldPath, err)
	}
	if string(gotThroughAlias) != "irreplaceable secret" {
		t.Errorf("content read through alias = %q, want %q", gotThroughAlias, "irreplaceable secret")
	}
}

func TestReplaceWithSymlink_RefusesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "id_ed25519")
	newTarget := filepath.Join(dir, "dest")
	if err := os.WriteFile(oldPath, []byte("source content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newTarget, []byte("unrelated content already here"), 0o600); err != nil {
		t.Fatal(err)
	}

	fsys := New()
	if err := fsys.ReplaceWithSymlink(oldPath, newTarget); err == nil {
		t.Fatal("ReplaceWithSymlink succeeded against an existing destination, want a refusal")
	}

	got, err := os.ReadFile(newTarget)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "unrelated content already here" {
		t.Errorf("destination content = %q, want unchanged", got)
	}
	info, err := os.Lstat(oldPath)
	if err != nil {
		t.Fatalf("source %s missing after refused replace: %v", oldPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("source was turned into a symlink despite the refusal")
	}
}

// TestReplaceWithSymlink_FailureDuringCopyLeavesSourceUntouched injects a failure at the copy
// step (an unwritable destination directory) — the first of §11 point 4's steps — and asserts
// oldPath still holds the complete, unmodified original file afterward, exactly as D4's move rule
// requires (mirroring TestMoveFile_FailurePartwayLeavesSourceUntouched's own style, one directory
// up in internal/app).
func TestReplaceWithSymlink_FailureDuringCopyLeavesSourceUntouched(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "id_ed25519")
	original := "irreplaceable secret"
	if err := os.WriteFile(oldPath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	readOnlyDir := filepath.Join(dir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o700) })
	newTarget := filepath.Join(readOnlyDir, "id_ed25519")

	fsys := New()
	if err := fsys.ReplaceWithSymlink(oldPath, newTarget); err == nil {
		t.Fatal("ReplaceWithSymlink succeeded against an unwritable destination, want an error")
	}

	got, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatalf("source %s no longer exists after a failed replace: %v", oldPath, err)
	}
	if string(got) != original {
		t.Errorf("source content = %q, want unchanged %q", got, original)
	}
	if info, err := os.Lstat(oldPath); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Error("source was turned into a symlink despite the copy step failing")
	}
	if _, err := os.Lstat(newTarget); !os.IsNotExist(err) {
		t.Errorf("destination %s exists despite a failed copy", newTarget)
	}
}

// TestReplaceWithSymlink_FailureDuringReplaceLeavesSourceUntouched injects a failure at the
// terminal replace step — after the copy has already been verified byte-for-byte at newTarget —
// by making oldPath's own directory unwritable so the scratch symlink can never be created there.
// This is exactly roadmap.md §4 exit criterion 5's own scenario: a failure between "copy verified"
// and "source replaced," and oldPath must still hold a complete, readable key.
func TestReplaceWithSymlink_FailureDuringReplaceLeavesSourceUntouched(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir() // a separate, writable directory — the copy step must succeed
	oldPath := filepath.Join(sourceDir, "id_ed25519")
	newTarget := filepath.Join(destDir, "id_ed25519")
	original := "irreplaceable secret"
	if err := os.WriteFile(oldPath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(sourceDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sourceDir, 0o700) })

	fsys := New()
	if err := fsys.ReplaceWithSymlink(oldPath, newTarget); err == nil {
		t.Fatal("ReplaceWithSymlink succeeded despite an unwritable source directory, want an error")
	}

	// The copy step (in the separate, writable destDir) succeeded and was verified — newTarget
	// holds a complete, valid copy — but oldPath was never replaced.
	got, err := os.ReadFile(newTarget)
	if err != nil {
		t.Fatalf("verified copy at %s is missing after the replace step failed: %v", newTarget, err)
	}
	if string(got) != original {
		t.Errorf("newTarget content = %q, want %q", got, original)
	}

	sourceGot, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatalf("source %s no longer readable after a failed replace: %v", oldPath, err)
	}
	if string(sourceGot) != original {
		t.Errorf("source content = %q, want unchanged %q", sourceGot, original)
	}
	if info, err := os.Lstat(oldPath); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Error("source was turned into a symlink despite the terminal replace step failing")
	}
}

// --- ReplaceSymlinkWithFile (release) ---

func TestReplaceSymlinkWithFile_HappyPath(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	if err := os.Mkdir(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(profileDir, "id_ed25519")
	original := "irreplaceable secret"
	if err := os.WriteFile(source, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "id_ed25519")
	if err := os.Symlink(source, path); err != nil {
		t.Fatal(err)
	}

	fsys := New()
	if err := fsys.ReplaceSymlinkWithFile(path, source); err != nil {
		t.Fatalf("ReplaceSymlinkWithFile: %v", err)
	}

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%s): %v", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%s is still a symlink after ReplaceSymlinkWithFile", path)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("path mode = %v, want source's preserved 0600", info.Mode().Perm())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != original {
		t.Errorf("path content = %q, want %q", got, original)
	}

	// source itself is untouched — removing it is the caller's own, separate step.
	sourceGot, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("source %s no longer readable: %v", source, err)
	}
	if string(sourceGot) != original {
		t.Errorf("source content = %q, want unchanged %q", sourceGot, original)
	}
}

func TestReplaceSymlinkWithFile_RefusesNonSymlinkPath(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.WriteFile(source, []byte("bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "not-a-symlink")
	if err := os.WriteFile(path, []byte("already a real file"), 0o600); err != nil {
		t.Fatal(err)
	}

	fsys := New()
	if err := fsys.ReplaceSymlinkWithFile(path, source); err == nil {
		t.Fatal("ReplaceSymlinkWithFile succeeded against a non-symlink path, want a refusal")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "already a real file" {
		t.Errorf("path content = %q, want unchanged", got)
	}
}

// TestReplaceSymlinkWithFile_FailureDuringCopyLeavesAliasUntouched injects a failure before the
// terminal rename — by making path's own directory unwritable, so the scratch temp file can never
// be created there — and asserts path is still exactly the original, working symlink afterward,
// and source is untouched: release's own mirror of exit criterion 5.
func TestReplaceSymlinkWithFile_FailureDuringCopyLeavesAliasUntouched(t *testing.T) {
	aliasDir := t.TempDir()
	profileDir := t.TempDir() // a separate, writable directory holding the real file
	source := filepath.Join(profileDir, "id_ed25519")
	original := "irreplaceable secret"
	if err := os.WriteFile(source, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(aliasDir, "id_ed25519")
	if err := os.Symlink(source, path); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(aliasDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(aliasDir, 0o700) })

	fsys := New()
	if err := fsys.ReplaceSymlinkWithFile(path, source); err == nil {
		t.Fatal("ReplaceSymlinkWithFile succeeded despite an unwritable alias directory, want an error")
	}

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("alias %s missing after a failed replace: %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("alias is no longer a symlink after a failed replace")
	}
	gotTarget, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	if gotTarget != source {
		t.Errorf("alias target = %q, want unchanged %q", gotTarget, source)
	}

	sourceGot, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("source %s no longer readable: %v", source, err)
	}
	if string(sourceGot) != original {
		t.Errorf("source content = %q, want unchanged %q", sourceGot, original)
	}
}
