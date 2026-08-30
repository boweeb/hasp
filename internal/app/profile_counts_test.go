package app

import (
	"path/filepath"
	"testing"
)

// TestListProfiles_KeyAndHostCounts guards tdd.md §9's command grid ("Every directory carrying
// .hasp, with key/host counts") — found missing entirely by an independent code review of M1
// (ListProfiles returned bare domain.Profile values with no count anywhere, human or --json,
// despite the command's own --help text promising them).
func TestListProfiles_KeyAndHostCounts(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "work", "id_ed25519")
	copyFixture(t, "ed25519-openssh-plain-nopub", realPath)
	writeFile(t, filepath.Join(dir, "work", ".hasp"), "# marker\n")
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile "+realPath+"\n")

	// A second, empty managed profile — must report zero counts, not be dropped.
	writeFile(t, filepath.Join(dir, "personal", ".hasp"), "# marker\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}

	summaries := ListProfiles(m)
	byPath := map[string]ProfileSummary{}
	for _, s := range summaries {
		byPath[s.Profile.Path.String()] = s
	}

	work, ok := byPath["work"]
	if !ok {
		t.Fatal("work profile missing from ListProfiles")
	}
	if work.KeyCount != 1 {
		t.Errorf("work.KeyCount = %d, want 1", work.KeyCount)
	}
	if work.HostCount != 1 {
		t.Errorf("work.HostCount = %d, want 1", work.HostCount)
	}

	personal, ok := byPath["personal"]
	if !ok {
		t.Fatal("personal profile missing from ListProfiles")
	}
	if personal.KeyCount != 0 || personal.HostCount != 0 {
		t.Errorf("personal = %+v, want zero counts", personal)
	}
}
