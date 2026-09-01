package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fswrite"
)

func TestCreateMarker_HappyPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	c := CreateMarker{Dir: dir, Header: []byte("# managed by hasp\n")}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, ".hasp"))
	if err != nil {
		t.Fatalf("read .hasp: %v", err)
	}
	if string(got) != "# managed by hasp\n" {
		t.Errorf("content = %q, want %q", got, "# managed by hasp\n")
	}
}

func TestCreateMarker_RefusesExistingMarker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".hasp"), "# a note I wrote myself\n")

	c := CreateMarker{Dir: dir, Header: []byte("# managed by hasp\n")}
	err := c.Apply(fswrite.New())
	if !errors.Is(err, ErrMarkerExists) {
		t.Fatalf("Apply error = %v, want ErrMarkerExists", err)
	}

	got, readErr := os.ReadFile(filepath.Join(dir, ".hasp"))
	if readErr != nil {
		t.Fatalf("read .hasp: %v", readErr)
	}
	if string(got) != "# a note I wrote myself\n" {
		t.Errorf("content = %q, want unchanged", got)
	}
}

func TestCreateMarker_RequiresBackup(t *testing.T) {
	if (CreateMarker{}).RequiresBackup() {
		t.Error("CreateMarker.RequiresBackup() = true, want false")
	}
}
