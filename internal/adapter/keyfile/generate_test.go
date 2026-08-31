package keyfile

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/boweeb/hasp/internal/domain"
)

func TestGenerate_NoPassphrase_RoundTripsThroughInspect(t *testing.T) {
	got, err := Generate(GenerateOptions{Comment: "test@hasp"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path, got.PrivateKeyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".pub", got.PublicKeyLine, 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := Inspect(path)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if info.Format != domain.FormatOpenSSH {
		t.Errorf("Format = %v, want FormatOpenSSH", info.Format)
	}
	if info.Algorithm != ssh.KeyAlgoED25519 {
		t.Errorf("Algorithm = %q, want %q", info.Algorithm, ssh.KeyAlgoED25519)
	}
	if info.Encrypted {
		t.Error("Encrypted = true, want false")
	}
	if !info.HasPublicHalf {
		t.Error("HasPublicHalf = false, want true")
	}
	if info.Fingerprint == "" {
		t.Error("Fingerprint is empty, want a derived value")
	}
	if info.Comment != "test@hasp" {
		t.Errorf("Comment = %q, want %q", info.Comment, "test@hasp")
	}
}

func TestGenerate_WithPassphrase_RoundTripsThroughInspect(t *testing.T) {
	passphrase := []byte("correct horse battery staple")
	got, err := Generate(GenerateOptions{Passphrase: passphrase})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path, got.PrivateKeyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	// Deliberately no .pub sidecar written here — this proves the encrypted-OpenSSH derivation
	// path (T1's second matrix row) still recovers algorithm and fingerprint from the embedded,
	// unencrypted public half, without ever decrypting anything.

	info, err := Inspect(path)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if !info.Encrypted {
		t.Error("Encrypted = false, want true")
	}
	if info.Algorithm != ssh.KeyAlgoED25519 {
		t.Errorf("Algorithm = %q, want %q", info.Algorithm, ssh.KeyAlgoED25519)
	}
	if info.Fingerprint == "" {
		t.Error("Fingerprint is empty, want a derived value (OpenSSH embeds the public half unencrypted, T1)")
	}
}

// TestGenerate_ZeroesThePassphraseBuffer is T6/D17's safety-rule guard: the passphrase buffer is
// zeroed immediately after use, not merely left for garbage collection.
func TestGenerate_ZeroesThePassphraseBuffer(t *testing.T) {
	passphrase := []byte("zero me please")
	if _, err := Generate(GenerateOptions{Passphrase: passphrase}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for i, b := range passphrase {
		if b != 0 {
			t.Fatalf("passphrase[%d] = %v, want 0 (buffer not zeroed)", i, b)
		}
	}
}

func TestGenerate_NoPassphraseIsANoOpForZeroing(t *testing.T) {
	if _, err := Generate(GenerateOptions{Passphrase: nil}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
}
