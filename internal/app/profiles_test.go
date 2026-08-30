package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDerive_KeyProfilesUnionAcrossAliasLocation(t *testing.T) {
	dir := t.TempDir()

	// The real file lives in work/foobarco/ (managed); an alias symlink lives in personal/
	// (also managed) — Key.Profiles must be the union of both (T19, D2).
	realPath := filepath.Join(dir, "work", "foobarco", "id_ed25519")
	copyFixture(t, "ed25519-openssh-plain-nopub", realPath)
	writeFile(t, filepath.Join(dir, "work", "foobarco", ".hasp"), "# profile marker\n")
	writeFile(t, filepath.Join(dir, "personal", ".hasp"), "# profile marker\n")

	aliasPath := filepath.Join(dir, "personal", "id_ed25519")
	if err := os.Symlink(realPath, aliasPath); err != nil {
		t.Fatal(err)
	}

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Keys) != 1 {
		t.Fatalf("got %d keys, want 1: %+v", len(m.Keys), m.Keys)
	}
	profiles := m.Keys[0].Profiles
	if len(profiles) != 2 {
		t.Fatalf("got %d profiles, want 2 (work.foobarco and personal): %v", len(profiles), profiles)
	}
	got := map[string]bool{profiles[0].String(): true, profiles[1].String(): true}
	if !got["work.foobarco"] || !got["personal"] {
		t.Errorf("Profiles = %v, want [personal work.foobarco]", profiles)
	}
}

func TestDerive_TopLevelKeyHasNoProfile(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Keys) != 1 {
		t.Fatalf("got %d keys, want 1", len(m.Keys))
	}
	if len(m.Keys[0].Profiles) != 0 {
		t.Errorf("Profiles = %v, want none — a top-level key belongs to no profile (legitimate, D13)", m.Keys[0].Profiles)
	}
}

func TestDerive_UnmarkedDirectoryContributesNoTaxonomy(t *testing.T) {
	dir := t.TempDir()
	// work/ has no .hasp marker — it's a container, not itself a profile (§5.3).
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "work", "id_ed25519"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Keys) != 1 {
		t.Fatalf("got %d keys, want 1", len(m.Keys))
	}
	if len(m.Keys[0].Profiles) != 0 {
		t.Errorf("Profiles = %v, want none — the containing directory carries no .hasp marker", m.Keys[0].Profiles)
	}

	if len(m.Profiles) != 1 || m.Profiles[0].Managed {
		t.Errorf("m.Profiles = %+v, want one unmanaged candidate (work)", m.Profiles)
	}
}
