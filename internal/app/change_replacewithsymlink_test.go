package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fswrite"
)

func TestReplaceWithSymlink_HappyPath(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	if err := os.Mkdir(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	from := filepath.Join(dir, "id_ed25519")
	to := filepath.Join(profileDir, "id_ed25519")
	writeFile(t, from, "secret bytes")

	c := ReplaceWithSymlink{From: from, To: to}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	info, err := os.Lstat(from)
	if err != nil {
		t.Fatalf("Lstat(%s): %v", from, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("%s is not a symlink after Apply", from)
	}
	got, err := os.ReadFile(to)
	if err != nil {
		t.Fatalf("read %s: %v", to, err)
	}
	if string(got) != "secret bytes" {
		t.Errorf("destination content = %q, want %q", got, "secret bytes")
	}
}

func TestReplaceWithSymlink_RequiresBackup(t *testing.T) {
	if !(ReplaceWithSymlink{From: "/a", To: "/b"}).RequiresBackup() {
		t.Error("ReplaceWithSymlink.RequiresBackup() = false, want true")
	}
}

func TestReplaceWithSymlink_Preview_NeverExposesKeyBytes(t *testing.T) {
	c := ReplaceWithSymlink{From: "/a/id_ed25519", To: "/a/work/id_ed25519"}
	p := c.Preview()
	if p.Diff != nil {
		t.Errorf("Preview().Diff = %+v, want nil (never rendering key bytes)", p.Diff)
	}
	if p.Summary == "" {
		t.Error("Preview().Summary is empty")
	}
}
