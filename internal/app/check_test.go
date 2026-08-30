package app

import (
	"path/filepath"
	"testing"
)

func TestCheck_DuplicateKeyConfirmed(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "rsa-openssh-plain-nopub", filepath.Join(dir, "id_rsa_old"))
	copyFixture(t, "rsa-openssh-plain-nopub", filepath.Join(dir, "legacy", "id_rsa"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	findings := Check(m)
	if !hasFinding(findings, FindingDuplicateKeyConfirmed) {
		t.Errorf("findings = %+v, want a duplicate-key-confirmed finding", findings)
	}
}

func TestCheck_DuplicateKeyUnconfirmed(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "id_rsa_old"))
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "legacy", "id_rsa"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	findings := Check(m)
	count := countFindings(findings, FindingDuplicateKeyUnconfirmed)
	if count != 2 {
		t.Fatalf("got %d duplicate-key-unconfirmed findings, want 2 (one per undecidable key): %+v", count, findings)
	}
}

func TestCheck_KeyMissingPublicHalf(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !hasFinding(Check(m), FindingKeyMissingPublicHalf) {
		t.Error("want a key-missing-public-half finding")
	}
}

func TestCheck_KeyMissingPublicHalf_AbsentWhenPubExists(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-pub", filepath.Join(dir, "id_ed25519"))
	copyFixture(t, "ed25519-openssh-plain-pub.pub", filepath.Join(dir, "id_ed25519.pub"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if hasFinding(Check(m), FindingKeyMissingPublicHalf) {
		t.Error("want no key-missing-public-half finding when a .pub sidecar exists")
	}
}

func TestCheck_KeyNoProfile(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !hasFinding(Check(m), FindingKeyNoProfile) {
		t.Error("want a key-no-profile finding for a top-level key")
	}
}

func TestCheck_FingerprintUnknown(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "id_rsa_old"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !hasFinding(Check(m), FindingFingerprintUnknown) {
		t.Error("want a fingerprint-unknown finding")
	}
}

func TestCheck_EmptyProfileDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "empty-profile", ".hasp"), "# marker\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !hasFinding(Check(m), FindingEmptyProfileDir) {
		t.Error("want an empty-profile-dir finding")
	}
}

func TestCheck_UnmarkedProfileDir(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "work", "id_ed25519"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !hasFinding(Check(m), FindingUnmarkedProfileDir) {
		t.Error("want an unmarked-profile-dir finding")
	}
}

func TestCheck_CleanMachineHasNoFindings(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "work", "id_ed25519")
	copyFixture(t, "ed25519-openssh-plain-pub", realPath)
	copyFixture(t, "ed25519-openssh-plain-pub.pub", realPath+".pub")
	writeFile(t, filepath.Join(dir, "work", ".hasp"), "# marker\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if findings := Check(m); len(findings) != 0 {
		t.Errorf("got %d findings on a clean machine, want 0: %+v", len(findings), findings)
	}
}

func hasFinding(findings []Finding, id FindingID) bool {
	return countFindings(findings, id) > 0
}

func countFindings(findings []Finding, id FindingID) int {
	n := 0
	for _, f := range findings {
		if f.ID == id {
			n++
		}
	}
	return n
}
