package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fswrite"
)

func TestWriteKeyFile_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")

	c := WriteKeyFile{Path: path, Contents: []byte("new key bytes"), Mode: 0o600}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != "new key bytes" {
		t.Errorf("content = %q, want %q", got, "new key bytes")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

// TestWriteKeyFile_NeverSilentlyOverwrites is the named guard test (tdd.md §12, T22): pre-create
// a file at the target path, run Apply with AllowOverwrite false, and assert both that it fails
// closed and that the pre-existing file's bytes are byte-identical afterward.
func TestWriteKeyFile_NeverSilentlyOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	original := "pre-existing, irreplaceable key material"
	writeFile(t, path, original)

	c := WriteKeyFile{Path: path, Contents: []byte("attempted overwrite"), Mode: 0o600, AllowOverwrite: false}
	err := c.Apply(fswrite.New())
	if !errors.Is(err, ErrKeyFileExists) {
		t.Fatalf("Apply error = %v, want ErrKeyFileExists", err)
	}

	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read %s: %v", path, readErr)
	}
	if string(got) != original {
		t.Errorf("content = %q, want unchanged %q", got, original)
	}
}

func TestWriteKeyFile_AllowOverwriteReplaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	writeFile(t, path, "old material")

	c := WriteKeyFile{Path: path, Contents: []byte("replacement material"), Mode: 0o600, AllowOverwrite: true}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != "replacement material" {
		t.Errorf("content = %q, want %q", got, "replacement material")
	}
}

func TestWriteKeyFile_RequiresBackupMirrorsAllowOverwrite(t *testing.T) {
	if (WriteKeyFile{AllowOverwrite: false}).RequiresBackup() {
		t.Error("AllowOverwrite=false: RequiresBackup() = true, want false")
	}
	if !(WriteKeyFile{AllowOverwrite: true}).RequiresBackup() {
		t.Error("AllowOverwrite=true: RequiresBackup() = false, want true")
	}
}

// TestWriteKeyFile_PreviewNeverCarriesADiff is T26's guard: Preview().Diff must be nil
// unconditionally — hasp never renders private key bytes, even bytes it is about to write itself.
func TestWriteKeyFile_PreviewNeverCarriesADiff(t *testing.T) {
	for _, allowOverwrite := range []bool{false, true} {
		c := WriteKeyFile{Path: "/whatever", Contents: []byte("super secret"), AllowOverwrite: allowOverwrite}
		p := c.Preview()
		if p.Diff != nil {
			t.Errorf("AllowOverwrite=%v: Preview().Diff = %+v, want nil", allowOverwrite, p.Diff)
		}
		if p.Summary == "" {
			t.Errorf("AllowOverwrite=%v: Preview().Summary is empty", allowOverwrite)
		}
	}
}
