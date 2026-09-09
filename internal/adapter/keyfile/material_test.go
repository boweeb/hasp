package keyfile

import (
	"os"
	"path/filepath"
	"testing"
)

// testPassphrase mirrors tools/genfixtures/main.go's own constant of the same name and value —
// a fixed, publicly-known passphrase used only to make an "encrypted" fixture parseable by tests,
// never a secret. Duplicated here rather than imported because tools/genfixtures is a `main`
// package.
const testPassphrase = "hasp-test-fixture-passphrase"

// TestOpenMaterial_PlainKey_PrivateAndPublicPopulated proves the "unencrypted is readable
// regardless of passphrase" half of OpenMaterial's contract: passphrase == nil must not be
// mistaken for "the key is encrypted."
func TestOpenMaterial_PlainKey_PrivateAndPublicPopulated(t *testing.T) {
	dir := testdataKeysDir(t)
	mat, err := OpenMaterial(filepath.Join(dir, "rsa-pem-plain-pub"), nil)
	if err != nil {
		t.Fatalf("OpenMaterial: %v", err)
	}
	if mat.Private == nil {
		t.Error("Private is nil for an unencrypted key; want populated regardless of passphrase")
	}
	if mat.Public == nil {
		t.Error("Public is nil for a key with a .pub sidecar; want populated")
	}
}

// TestOpenMaterial_EncryptedKey_NoPassphrase_PublicHalfOnly proves the "passphrase == nil means
// public half only, never decrypt" half of the contract (D19, P3's custody rule): Private must
// stay nil, and no error is returned merely because decryption was never attempted.
func TestOpenMaterial_EncryptedKey_NoPassphrase_PublicHalfOnly(t *testing.T) {
	dir := testdataKeysDir(t)
	mat, err := OpenMaterial(filepath.Join(dir, "rsa-openssh-encrypted-pub"), nil)
	if err != nil {
		t.Fatalf("OpenMaterial: %v", err)
	}
	if mat.Private != nil {
		t.Error("Private is populated with passphrase == nil; must stay nil (D19: never decrypt without consent)")
	}
	if mat.Public == nil {
		t.Error("Public is nil for an OpenSSH-format encrypted key with a .pub sidecar; want populated")
	}
}

// TestOpenMaterial_EncryptedKey_WrongPassphrase_DegradesNotErrors proves a wrong passphrase is
// reported as Material.Private == nil, never as an error — §5.1's "unknown, not aborted" stance.
func TestOpenMaterial_EncryptedKey_WrongPassphrase_DegradesNotErrors(t *testing.T) {
	dir := testdataKeysDir(t)
	mat, err := OpenMaterial(filepath.Join(dir, "rsa-openssh-encrypted-pub"), []byte("definitely-not-the-passphrase"))
	if err != nil {
		t.Fatalf("OpenMaterial with wrong passphrase returned an error, want a degraded Material: %v", err)
	}
	if mat.Private != nil {
		t.Error("Private is populated despite a wrong passphrase")
	}
}

// TestOpenMaterial_EncryptedKey_CorrectPassphrase_PrivatePopulated proves the consent-gated half
// of the contract actually works: given the fixture's known passphrase (tools/genfixtures),
// Private is populated and usable.
func TestOpenMaterial_EncryptedKey_CorrectPassphrase_PrivatePopulated(t *testing.T) {
	dir := testdataKeysDir(t)
	mat, err := OpenMaterial(filepath.Join(dir, "rsa-openssh-encrypted-pub"), []byte(testPassphrase))
	if err != nil {
		t.Fatalf("OpenMaterial: %v", err)
	}
	if mat.Private == nil {
		t.Error("Private is nil despite the correct passphrase")
	}
	if mat.Public == nil {
		t.Error("Public is nil despite the correct passphrase")
	}
}

// TestOpenMaterial_NotAKeyFile proves OpenMaterial's error return is reserved for path itself
// being unreadable or unparseable, the same failure shape Inspect already uses.
func TestOpenMaterial_NotAKeyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "not-a-key")
	if err := os.WriteFile(path, []byte("not a key at all"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenMaterial(path, nil); err == nil {
		t.Error("OpenMaterial on a non-key file: want an error, got nil")
	}
}
