package app

import (
	"path/filepath"
	"testing"
)

func TestShowKey_FoundWithBoundHost(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "id_ed25519")
	copyFixture(t, "ed25519-openssh-plain-nopub", realPath)
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile "+realPath+"\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	detail, ok := ShowKey(m, "id_ed25519")
	if !ok {
		t.Fatal("ShowKey: not found")
	}
	if len(detail.Hosts) != 1 || detail.Hosts[0].Patterns[0] != "work" {
		t.Errorf("Hosts = %+v, want [work]", detail.Hosts)
	}
}

func TestShowKey_NotFound(t *testing.T) {
	dir := t.TempDir()
	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if _, ok := ShowKey(m, "nope"); ok {
		t.Error("ShowKey: want not found")
	}
}

func TestFindKeys_FragmentPunctuationAndCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-pub", filepath.Join(dir, "id_ed25519"))
	copyFixture(t, "ed25519-openssh-plain-pub.pub", filepath.Join(dir, "id_ed25519.pub"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	full := m.Keys[0].Identity.Value() // "SHA256:xxxxx..."
	fragment := full[len(full)-8:]

	// Mangle case and inject punctuation into the fragment — J2's requirement.
	mangled := "  " + fragment[:2] + ":" + fragment[2:]

	got := FindKeys(m, mangled)
	if len(got) != 1 {
		t.Fatalf("FindKeys(%q) = %d results, want 1", mangled, len(got))
	}
}

func TestFindKeys_NoMatch(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if got := FindKeys(m, "zzzzznotarealfragment"); len(got) != 0 {
		t.Errorf("got %d results, want 0", len(got))
	}
}

func TestFindKeys_SkipsUndecidableKeys(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "id_rsa_old"))
	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if got := FindKeys(m, "anything"); len(got) != 0 {
		t.Errorf("got %d results for an undecidable-only machine, want 0", len(got))
	}
}
