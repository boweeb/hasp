package keyfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/boweeb/hasp/internal/domain"
)

// testdataKeysDir is defined in fixtures_test.go (M0) and reused here.

// blockFormatFor mirrors fixtures_test.go's blockTypeFor, but returns the domain.KeyFormat this
// package must derive from that block type — the two tests are independent proofs of the same
// fixture matrix, one against x/crypto/ssh directly (M0), one against this package (M1).
func blockFormatFor(t *testing.T, format, algo string) domain.KeyFormat {
	t.Helper()
	switch format {
	case "openssh":
		return domain.FormatOpenSSH
	case "pem":
		switch algo {
		case "rsa":
			return domain.FormatPKCS1
		case "ecdsa":
			return domain.FormatSEC1
		case "dsa":
			return domain.FormatDSA
		case "ed25519":
			return domain.FormatPKCS8
		}
	}
	t.Fatalf("unhandled format/algo combination %q/%q", format, algo)
	return domain.FormatUnknown
}

func wantAlgorithm(algo string) string {
	switch algo {
	case "rsa":
		return "ssh-rsa"
	case "ecdsa":
		return "" // curve-dependent; checked separately by prefix
	case "ed25519":
		return "ssh-ed25519"
	case "dsa":
		return "ssh-dss"
	default:
		return ""
	}
}

func TestInspect_AllFixtures(t *testing.T) {
	dir := testdataKeysDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	var fixtures []string
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".pub") {
			continue
		}
		fixtures = append(fixtures, e.Name())
	}
	if len(fixtures) != 28 {
		t.Fatalf("got %d private-key fixtures, want 28", len(fixtures))
	}

	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			parts := strings.Split(name, "-")
			if len(parts) != 4 {
				t.Fatalf("fixture name %q does not parse as algo-format-encryption-pubhalf", name)
			}
			algo, format, encryption, pubhalf := parts[0], parts[1], parts[2], parts[3]

			info, err := Inspect(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("Inspect(%s): %v", name, err)
			}

			wantFormat := blockFormatFor(t, format, algo)
			if info.Format != wantFormat {
				t.Errorf("Format = %v, want %v", info.Format, wantFormat)
			}

			wantEncrypted := encryption == "encrypted"
			if info.Encrypted != wantEncrypted {
				t.Errorf("Encrypted = %v, want %v", info.Encrypted, wantEncrypted)
			}

			wantPub := pubhalf == "pub"
			if info.HasPublicHalf != wantPub {
				t.Errorf("HasPublicHalf = %v, want %v", info.HasPublicHalf, wantPub)
			}

			// The derivation gap (T1, D12): exactly legacy-PEM + encrypted + no public half
			// yields an unknown fingerprint. Every other cell must derive one.
			isDerivationGap := format == "pem" && encryption == "encrypted" && pubhalf == "nopub"
			if isDerivationGap {
				if info.Fingerprint != "" {
					t.Errorf("Fingerprint = %q, want \"\" (derivation gap)", info.Fingerprint)
				}
				if info.Algorithm != "" {
					t.Errorf("Algorithm = %q, want \"\" (derivation gap)", info.Algorithm)
				}
			} else {
				if info.Fingerprint == "" {
					t.Errorf("Fingerprint is empty, want a derived value")
				}
				if !strings.HasPrefix(string(info.Fingerprint), "SHA256:") {
					t.Errorf("Fingerprint = %q, want SHA256: prefix", info.Fingerprint)
				}
				if info.Algorithm == "" {
					t.Errorf("Algorithm is empty, want a derived value")
				}
				if want := wantAlgorithm(algo); want != "" && info.Algorithm != want {
					t.Errorf("Algorithm = %q, want %q", info.Algorithm, want)
				}
				if algo == "ecdsa" && !strings.HasPrefix(info.Algorithm, "ecdsa-sha2-nistp") {
					t.Errorf("Algorithm = %q, want an ecdsa-sha2-nistp* prefix", info.Algorithm)
				}
				if info.Bits == 0 {
					t.Errorf("Bits = 0, want a derived value")
				}
			}
		})
	}
}

// TestInspect_CommentFromPubSidecar confirms the comment is sourced from the .pub sibling only
// (no OpenSSH-wire comment recovery in M1) by asserting a fixture with a public half yields the
// same comment ssh.ParseAuthorizedKey would.
func TestInspect_CommentFromPubSidecar(t *testing.T) {
	dir := testdataKeysDir(t)
	info, err := Inspect(filepath.Join(dir, "ed25519-openssh-plain-pub"))
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	pubBytes, err := os.ReadFile(filepath.Join(dir, "ed25519-openssh-plain-pub.pub"))
	if err != nil {
		t.Fatalf("read .pub: %v", err)
	}
	_, wantComment, _, _, err := ssh.ParseAuthorizedKey(pubBytes)
	if err != nil {
		t.Fatalf("parse .pub: %v", err)
	}
	if info.Comment != wantComment {
		t.Errorf("Comment = %q, want %q", info.Comment, wantComment)
	}
}

func TestInspect_NonKeyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "not-a-key")
	if err := os.WriteFile(path, []byte("hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(path); err == nil {
		t.Error("Inspect on a non-key file: got nil error, want an error (fail-open at the scan layer)")
	}
}
