package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fswrite"
)

func TestMoveFile_HappyPath(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "src")
	to := filepath.Join(dir, "dst")
	writeFile(t, from, "secret bytes")

	c := MoveFile{From: from, To: to, Reason: "test"}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if _, err := os.Lstat(from); !os.IsNotExist(err) {
		t.Errorf("source %s still exists after a successful move", from)
	}
	got, err := os.ReadFile(to)
	if err != nil {
		t.Fatalf("read %s: %v", to, err)
	}
	if string(got) != "secret bytes" {
		t.Errorf("destination content = %q, want %q", got, "secret bytes")
	}
}

func TestMoveFile_RequiresBackup(t *testing.T) {
	c := MoveFile{From: "/a", To: "/b"}
	if !c.RequiresBackup() {
		t.Error("MoveFile.RequiresBackup() = false, want true")
	}
}

// TestMoveFile_FailurePartwayLeavesSourceUntouched is D4's move rule guard test (tdd.md §12,
// T15): inject a failure partway through the move (an unwritable destination directory) and
// assert the original file still exists, is readable, and is byte-identical at its original path
// afterward — regardless of which step failed.
func TestMoveFile_FailurePartwayLeavesSourceUntouched(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "src")
	original := "irreplaceable secret"
	writeFile(t, from, original)

	readOnlyDir := filepath.Join(dir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0o500); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o700) }) // let TempDir clean up afterward
	to := filepath.Join(readOnlyDir, "dst")

	c := MoveFile{From: from, To: to}
	if err := c.Apply(fswrite.New()); err == nil {
		t.Fatal("Apply succeeded against an unwritable destination, want an error")
	}

	got, err := os.ReadFile(from)
	if err != nil {
		t.Fatalf("source %s no longer exists after a failed move: %v", from, err)
	}
	if string(got) != original {
		t.Errorf("source content = %q, want unchanged %q", got, original)
	}
	if _, err := os.Lstat(to); !os.IsNotExist(err) {
		t.Errorf("destination %s exists despite a failed move", to)
	}
}

func TestMoveFile_RefusesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "src")
	to := filepath.Join(dir, "dst")
	writeFile(t, from, "source content")
	writeFile(t, to, "unrelated content already here")

	c := MoveFile{From: from, To: to}
	if err := c.Apply(fswrite.New()); err == nil {
		t.Fatal("Apply succeeded against an existing destination, want a refusal")
	}

	got, err := os.ReadFile(to)
	if err != nil {
		t.Fatalf("read %s: %v", to, err)
	}
	if string(got) != "unrelated content already here" {
		t.Errorf("destination content = %q, want it left untouched", got)
	}
}
