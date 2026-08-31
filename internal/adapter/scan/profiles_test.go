package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfiles_NestedManagedAndUnmanaged(t *testing.T) {
	dir := t.TempDir()

	// work/ is an unmarked container; work/foobarco/ and work/acme/ are marked profiles.
	writeFile(t, filepath.Join(dir, "work", "foobarco", ".hasp"), "# profile marker\n")
	writeFile(t, filepath.Join(dir, "work", "acme", ".hasp"), "# profile marker\n")
	// personal/ is a marked profile with no children.
	writeFile(t, filepath.Join(dir, "personal", ".hasp"), "# profile marker\n")

	got, err := Profiles(dir)
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}

	byRelPath := map[string]ProfileCandidate{}
	for _, c := range got {
		byRelPath[filepath.Join(c.Path...)] = c
	}

	if len(got) != 4 {
		t.Fatalf("got %d candidates, want 4 (work, work/foobarco, work/acme, personal): %+v", len(got), got)
	}

	work, ok := byRelPath[filepath.Join("work")]
	if !ok || work.Managed {
		t.Errorf("work = %+v, want present and unmanaged", work)
	}
	foobarco, ok := byRelPath[filepath.Join("work", "foobarco")]
	if !ok || !foobarco.Managed {
		t.Errorf("work/foobarco = %+v, want present and managed", foobarco)
	}
	if len(foobarco.Path) != 2 || foobarco.Path[0] != "work" || foobarco.Path[1] != "foobarco" {
		t.Errorf("work/foobarco.Path = %v, want [work foobarco]", foobarco.Path)
	}
	personal, ok := byRelPath["personal"]
	if !ok || !personal.Managed {
		t.Errorf("personal = %+v, want present and managed", personal)
	}
}

func TestProfiles_KeyDirItselfNotACandidate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".hasp"), []byte("# marker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Profiles(dir)
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d candidates, want 0 — the key directory itself is never a profile candidate: %+v", len(got), got)
	}
}

// TestProfiles_ExcludesBackupStore is the profile-side mirror of TestKeys_ExcludesBackupStore:
// hasp's own backup directory must never surface as a phantom, unmanaged profile candidate.
func TestProfiles_ExcludesBackupStore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "work", ".hasp"), "# profile marker\n")
	writeFile(t, filepath.Join(dir, ".hasp-backups", "id_ed25519.20260828T140501Z"), "backed up key bytes")

	got, err := Profiles(dir)
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	for _, c := range got {
		if c.Path[0] == ".hasp-backups" {
			t.Errorf("Profiles returned a candidate for .hasp-backups: %+v", c)
		}
	}
	if len(got) != 1 {
		t.Fatalf("got %d candidates, want exactly 1 (work, not .hasp-backups): %+v", len(got), got)
	}
}

func TestProfiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	got, err := Profiles(dir)
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d candidates in an empty dir, want 0", len(got))
	}
}
