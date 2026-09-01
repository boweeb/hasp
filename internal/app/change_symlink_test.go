package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fswrite"
)

func TestCreateSymlink_HappyPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "id_ed25519_foobarco")
	writeFile(t, target, "key bytes")
	aliasPath := filepath.Join(dir, "id_ed25519")

	c := CreateSymlink{Path: aliasPath, Target: target}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.Readlink(aliasPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if got != target {
		t.Errorf("link target = %q, want %q", got, target)
	}
}

func TestCreateSymlink_RefusesExistingPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "id_ed25519_foobarco")
	writeFile(t, target, "key bytes")
	aliasPath := filepath.Join(dir, "id_ed25519")
	writeFile(t, aliasPath, "something already here")

	c := CreateSymlink{Path: aliasPath, Target: target}
	if err := c.Apply(fswrite.New()); err == nil {
		t.Fatal("Apply succeeded against an existing path, want a refusal")
	}

	got, err := os.ReadFile(aliasPath)
	if err != nil {
		t.Fatalf("read %s: %v", aliasPath, err)
	}
	if string(got) != "something already here" {
		t.Errorf("content = %q, want unchanged", got)
	}
}

func TestCreateSymlink_RequiresBackup(t *testing.T) {
	if (CreateSymlink{}).RequiresBackup() {
		t.Error("CreateSymlink.RequiresBackup() = true, want false")
	}
}
