package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fswrite"
)

func TestRemove_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".hasp")
	writeFile(t, path, "# marker\n")

	c := Remove{Path: path, Reason: "release profile"}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("%s still exists after Remove.Apply", path)
	}
}

func TestRemove_RequiresBackup(t *testing.T) {
	if !(Remove{}).RequiresBackup() {
		t.Error("Remove.RequiresBackup() = false, want true (T4: always true)")
	}
}
