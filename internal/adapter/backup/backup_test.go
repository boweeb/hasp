package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshot_CopiesFileIntoBackupDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 8, 28, 14, 5, 1, 0, time.UTC) }

	if err := s.Snapshot(path); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	backupPath := filepath.Join(dir, ".hasp-backups", "config.20260828T140501Z")
	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read %s: %v", backupPath, err)
	}
	if string(got) != "original\n" {
		t.Errorf("backup content = %q, want %q", got, "original\n")
	}
}

func TestSnapshot_MissingSourceIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	if err := s.Snapshot(filepath.Join(dir, "does-not-exist")); err != nil {
		t.Fatalf("Snapshot for a missing source returned %v, want nil (nothing to preserve)", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, ".hasp-backups")); !os.IsNotExist(err) {
		t.Error(".hasp-backups was created for a snapshot of a nonexistent file")
	}
}

func TestSnapshot_CollisionGetsAUniqueName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(dir)
	fixed := time.Date(2026, 8, 28, 14, 5, 1, 0, time.UTC)
	s.Now = func() time.Time { return fixed }

	if err := s.Snapshot(path); err != nil {
		t.Fatalf("Snapshot 1: %v", err)
	}
	if err := os.WriteFile(path, []byte("v2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.Snapshot(path); err != nil {
		t.Fatalf("Snapshot 2: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, ".hasp-backups"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d backup files, want 2 (no overwrite on a timestamp collision)", len(entries))
	}
}

func TestSnapshot_NeverPrunesPriorBackups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	s := New(dir)

	for i, content := range []string{"v1\n", "v2\n", "v3\n"} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		s.Now = func() time.Time { return time.Date(2026, 8, 28, 14, 5, i, 0, time.UTC) }
		if err := s.Snapshot(path); err != nil {
			t.Fatalf("Snapshot %d: %v", i, err)
		}
	}

	entries, err := os.ReadDir(filepath.Join(dir, ".hasp-backups"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("got %d backup files, want 3 — hasp never prunes (T8)", len(entries))
	}
}
