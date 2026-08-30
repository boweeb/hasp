package scan

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestKeys_FindsFilesAndSymlinksRecursively(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "id_ed25519"), "top-level key")
	writeFile(t, filepath.Join(dir, "id_ed25519.pub"), "top-level pub")
	writeFile(t, filepath.Join(dir, "work", "id_rsa_foobarco"), "nested key")
	writeFile(t, filepath.Join(dir, "work", ".hasp"), "# profile marker\n")

	// A top-level alias symlink pointing at the nested key (D5's ordinary case).
	aliasPath := filepath.Join(dir, "id_rsa_foobarco")
	if err := os.Symlink(filepath.Join(dir, "work", "id_rsa_foobarco"), aliasPath); err != nil {
		t.Fatal(err)
	}

	// A dangling symlink, which must be skipped rather than erroring the whole scan.
	if err := os.Symlink(filepath.Join(dir, "does-not-exist"), filepath.Join(dir, "broken_alias")); err != nil {
		t.Fatal(err)
	}

	got, err := Keys(dir)
	if err != nil {
		t.Fatalf("Keys: %v", err)
	}

	var paths []string
	for _, c := range got {
		paths = append(paths, c.Path)
	}
	sort.Strings(paths)

	want := []string{
		filepath.Join(dir, "id_ed25519"),
		filepath.Join(dir, "id_rsa_foobarco"),
		filepath.Join(dir, "work", "id_rsa_foobarco"),
	}
	sort.Strings(want)

	if len(paths) != len(want) {
		t.Fatalf("got %d candidates %v, want %d %v", len(paths), paths, len(want), want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("paths[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestKeys_SymlinkResolvedPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "work", "id_rsa_foobarco")
	writeFile(t, target, "nested key")

	alias := filepath.Join(dir, "id_rsa_foobarco")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}

	got, err := Keys(dir)
	if err != nil {
		t.Fatalf("Keys: %v", err)
	}

	var aliasCandidate *KeyCandidate
	for i := range got {
		if got[i].Path == alias {
			aliasCandidate = &got[i]
		}
	}
	if aliasCandidate == nil {
		t.Fatal("alias candidate not found")
	}
	if !aliasCandidate.IsSymlink {
		t.Error("IsSymlink = false, want true")
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	if aliasCandidate.ResolvedPath != resolvedTarget {
		t.Errorf("ResolvedPath = %q, want %q", aliasCandidate.ResolvedPath, resolvedTarget)
	}
}

func TestKeys_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	got, err := Keys(dir)
	if err != nil {
		t.Fatalf("Keys: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d candidates in an empty dir, want 0", len(got))
	}
}
