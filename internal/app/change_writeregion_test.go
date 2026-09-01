package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fswrite"
)

func TestWriteRegion_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	writeFile(t, path, "Host outside\n    User me\n# >>> hasp:managed >>>\nInclude foo\n# <<< hasp:managed <<<\n")

	c := WriteRegion{
		File:   path,
		Marker: "managed",
		Before: []byte("Include foo\n"),
		After:  []byte("Include foo\nInclude bar\n"),
	}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	want := "Host outside\n    User me\n# >>> hasp:managed >>>\nInclude foo\nInclude bar\n# <<< hasp:managed <<<\n"
	if string(got) != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestWriteRegion_EmptyBeforeAppendsToEndOfFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	writeFile(t, path, "Host outside\n")

	c := WriteRegion{
		File:   path,
		Marker: "managed",
		Before: nil,
		After:  []byte("# >>> hasp:managed >>>\nInclude foo\n# <<< hasp:managed <<<\n"),
	}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	want := "Host outside\n# >>> hasp:managed >>>\nInclude foo\n# <<< hasp:managed <<<\n"
	if string(got) != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestWriteRegion_RequiresBackup(t *testing.T) {
	if !(WriteRegion{}).RequiresBackup() {
		t.Error("WriteRegion.RequiresBackup() = false, want true")
	}
}

func TestWriteRegion_PreviewCarriesARealDiff(t *testing.T) {
	c := WriteRegion{
		File:   "/whatever/config",
		Marker: "managed",
		Before: []byte("Include foo\nInclude bar\n"),
		After:  []byte("Include foo\nInclude baz\n"),
	}
	p := c.Preview()
	if p.Diff == nil {
		t.Fatal("Preview().Diff = nil, want a real diff")
	}

	var added, removed, context int
	for _, line := range p.Diff {
		switch line.Kind {
		case DiffAdded:
			added++
		case DiffRemoved:
			removed++
		case DiffContext:
			context++
		}
	}
	if added != 1 || removed != 1 || context != 1 {
		t.Errorf("diff = %+v, want 1 added, 1 removed, 1 context line", p.Diff)
	}
}

// TestWriteRegion_EmptyBeforeCreatesFileThatDoesNotExist covers the M3 extension to Apply: a
// brand-new whole-file-owned group file (D7) doesn't exist on disk yet, so os.ReadFile(c.File)
// fails with ENOENT — Apply must tolerate that exactly when Before is also empty, treating it the
// same as the existing-file append case, and create the file with exactly After's bytes.
func TestWriteRegion_EmptyBeforeCreatesFileThatDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "work.sshconfig")

	c := WriteRegion{
		File:   path,
		Marker: "file",
		Before: nil,
		After:  []byte("# hasp:owned\nHost work\n    HostName work.example.com\n"),
	}
	if err := c.Apply(fswrite.New()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != string(c.After) {
		t.Errorf("content = %q, want %q", got, c.After)
	}
}

// TestWriteRegion_NonEmptyBeforeStillErrorsOnMissingFile confirms the ENOENT tolerance above is
// narrowly scoped to the empty-Before case: a non-empty Before describing a span that should have
// existed in a file that isn't even there at all must still be a hard error, not silently treated
// as "nothing to splice against."
func TestWriteRegion_NonEmptyBeforeStillErrorsOnMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	c := WriteRegion{
		File:   path,
		Marker: "managed",
		Before: []byte("Include foo\n"),
		After:  []byte("Include bar\n"),
	}
	if err := c.Apply(fswrite.New()); err == nil {
		t.Fatal("Apply succeeded against a nonexistent file with a non-empty Before, want an error")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("Apply's refusal still created the file, want no file at all")
	}
}

// TestWriteRegion_RefusesAmbiguousBefore covers the safety refusal in spliceRegion: if Before
// occurs more than once in the current file, Apply must refuse rather than guess which occurrence
// to replace.
func TestWriteRegion_RefusesAmbiguousBefore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	writeFile(t, path, "Include foo\nInclude foo\n")

	c := WriteRegion{File: path, Marker: "managed", Before: []byte("Include foo\n"), After: []byte("Include bar\n")}
	if err := c.Apply(fswrite.New()); err == nil {
		t.Fatal("Apply succeeded against an ambiguous Before span, want a refusal")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != "Include foo\nInclude foo\n" {
		t.Errorf("content changed despite the refusal: %q", got)
	}
}
