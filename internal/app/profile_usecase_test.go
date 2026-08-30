package app

import (
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/domain"
)

func TestShowProfile_RecursesByDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "work", ".hasp"), "# marker\n")
	writeFile(t, filepath.Join(dir, "work", "foobarco", ".hasp"), "# marker\n")
	writeFile(t, filepath.Join(dir, "work", "acme", ".hasp"), "# marker\n")
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "work", "foobarco", "id_ed25519"))
	copyFixture(t, "rsa-openssh-plain-nopub", filepath.Join(dir, "work", "acme", "id_rsa"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}

	detail, ok := ShowProfile(m, domain.ProfilePath{"work"}, true)
	if !ok {
		t.Fatal("ShowProfile(work, recurse): not found")
	}
	if len(detail.Keys) != 2 {
		t.Fatalf("got %d keys, want 2 (work.foobarco's and work.acme's, aggregated by default per T21): %+v", len(detail.Keys), detail.Keys)
	}
	attribution := map[string]bool{}
	for _, kr := range detail.Keys {
		attribution[kr.FromProfile.String()] = true
	}
	if !attribution["work.foobarco"] || !attribution["work.acme"] {
		t.Errorf("attribution = %v, want work.foobarco and work.acme both represented", attribution)
	}
}

func TestShowProfile_NoRecurseScopesToExactProfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "work", ".hasp"), "# marker\n")
	writeFile(t, filepath.Join(dir, "work", "foobarco", ".hasp"), "# marker\n")
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "work", "foobarco", "id_ed25519"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}

	detail, ok := ShowProfile(m, domain.ProfilePath{"work"}, false)
	if !ok {
		t.Fatal("ShowProfile(work, no-recurse): not found")
	}
	if len(detail.Keys) != 0 {
		t.Errorf("got %d keys, want 0 — --no-recurse must not pull in work.foobarco's key: %+v", len(detail.Keys), detail.Keys)
	}
}

func TestShowProfile_UnmarkedContainerNotAValidSubject(t *testing.T) {
	dir := t.TempDir()
	// work/ is unmarked; work/foobarco/ is marked.
	writeFile(t, filepath.Join(dir, "work", "foobarco", ".hasp"), "# marker\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if _, ok := ShowProfile(m, domain.ProfilePath{"work"}, true); ok {
		t.Error("ShowProfile(work): want not found — work carries no .hasp marker (T21)")
	}
}

func TestFindProfiles_ByNameFragment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "work", "foobarco", ".hasp"), "# marker\n")
	writeFile(t, filepath.Join(dir, "personal", ".hasp"), "# marker\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	got := FindProfiles(m, "foobarco")
	if len(got) != 1 || got[0].Path.String() != "work.foobarco" {
		t.Errorf("FindProfiles(foobarco) = %+v, want [work.foobarco]", got)
	}
}
