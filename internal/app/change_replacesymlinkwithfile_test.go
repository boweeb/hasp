package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fswrite"
)

func TestReplaceSymlinkWithFile_HappyPath(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	if err := os.Mkdir(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(profileDir, "id_ed25519")
	writeFile(t, source, "secret bytes")
	alias := filepath.Join(dir, "id_ed25519")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}

	c := ReplaceSymlinkWithFile{Alias: alias, Source: source}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	info, err := os.Lstat(alias)
	if err != nil {
		t.Fatalf("Lstat(%s): %v", alias, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("%s is still a symlink after Apply", alias)
	}
	got, err := os.ReadFile(alias)
	if err != nil {
		t.Fatalf("read %s: %v", alias, err)
	}
	if string(got) != "secret bytes" {
		t.Errorf("alias content = %q, want %q", got, "secret bytes")
	}
}

func TestReplaceSymlinkWithFile_RequiresBackup(t *testing.T) {
	if !(ReplaceSymlinkWithFile{Alias: "/a", Source: "/b"}).RequiresBackup() {
		t.Error("ReplaceSymlinkWithFile.RequiresBackup() = false, want true")
	}
}

func TestReplaceSymlinkWithFile_Preview_NeverExposesKeyBytes(t *testing.T) {
	c := ReplaceSymlinkWithFile{Alias: "/a/id_ed25519", Source: "/a/work/id_ed25519"}
	p := c.Preview()
	if p.Diff != nil {
		t.Errorf("Preview().Diff = %+v, want nil (never rendering key bytes)", p.Diff)
	}
	if p.Summary == "" {
		t.Error("Preview().Summary is empty")
	}
}
