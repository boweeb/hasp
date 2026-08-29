package keyfile

import (
	"crypto/dsa" //nolint:staticcheck // DSA is a legacy fixture-only case; see T23.
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// This test satisfies exit criterion #3 of docs/roadmap.md §2 ("all 28 key fixtures exist, are
// committed, and each parses to the algorithm, format, and encryption state its filename
// claims") without needing M1's actual keyfile adapter to exist yet — it works directly against
// golang.org/x/crypto/ssh, crypto/x509, and encoding/pem, the same library T1 says hasp's future
// keyfile adapter will use.

// blockTypeFor returns the PEM block Type a fixture's (format, algorithm) pair must produce.
// The block Type is never encrypted (only Bytes are), so this holds for encrypted fixtures too.
func blockTypeFor(t *testing.T, format, algo string) string {
	t.Helper()
	switch format {
	case "openssh":
		return "OPENSSH PRIVATE KEY"
	case "pem":
		switch algo {
		case "rsa":
			return "RSA PRIVATE KEY"
		case "ecdsa":
			return "EC PRIVATE KEY"
		case "dsa":
			return "DSA PRIVATE KEY"
		case "ed25519":
			return "PRIVATE KEY" // bare PKCS8 — the only algorithm using this block type here.
		}
	}
	t.Fatalf("unhandled format/algo combination %q/%q", format, algo)
	return ""
}

// assertRawType asserts ssh.ParseRawPrivateKey's return value has the Go type the fixture's
// algorithm (and, for ed25519, format) claims. x/crypto/ssh returns ed25519 keys as
// *ed25519.PrivateKey for the OpenSSH wire format (parseOpenSSHPrivateKey) but as the bare
// ed25519.PrivateKey value for PKCS8 PEM (x509.ParsePKCS8PrivateKey) — both are "the returned Go
// type matches the claimed algorithm," just via two different concrete shapes depending on
// format.
func assertRawType(t *testing.T, algo, format string, key any) {
	t.Helper()
	switch algo {
	case "rsa":
		if _, ok := key.(*rsa.PrivateKey); !ok {
			t.Errorf("algo %s: got %T, want *rsa.PrivateKey", algo, key)
		}
	case "ecdsa":
		if _, ok := key.(*ecdsa.PrivateKey); !ok {
			t.Errorf("algo %s: got %T, want *ecdsa.PrivateKey", algo, key)
		}
	case "ed25519":
		switch format {
		case "openssh":
			if _, ok := key.(*ed25519.PrivateKey); !ok {
				t.Errorf("algo %s format %s: got %T, want *ed25519.PrivateKey", algo, format, key)
			}
		case "pem":
			if _, ok := key.(ed25519.PrivateKey); !ok {
				t.Errorf("algo %s format %s: got %T, want ed25519.PrivateKey", algo, format, key)
			}
		}
	case "dsa":
		if _, ok := key.(*dsa.PrivateKey); !ok {
			t.Errorf("algo %s: got %T, want *dsa.PrivateKey", algo, key)
		}
	default:
		t.Fatalf("unhandled algorithm %q", algo)
	}
}

func testdataKeysDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// this file lives at internal/adapter/keyfile/fixtures_test.go
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(repoRoot, "testdata", "keys")
}

func TestKeyFixtures(t *testing.T) {
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

	const wantCount = 28
	if len(fixtures) != wantCount {
		t.Fatalf("got %d private-key fixtures in %s, want %d: %v", len(fixtures), dir, wantCount, fixtures)
	}

	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			parts := strings.Split(name, "-")
			if len(parts) != 4 {
				t.Fatalf("fixture name %q does not parse as algo-format-encryption-pubhalf", name)
			}
			algo, format, encryption, pubhalf := parts[0], parts[1], parts[2], parts[3]

			privPath := filepath.Join(dir, name)
			privBytes, err := os.ReadFile(privPath)
			if err != nil {
				t.Fatalf("read %s: %v", privPath, err)
			}

			// .pub sibling existence matches the pubhalf token, and parses if present.
			pubPath := privPath + ".pub"
			pubBytes, pubErr := os.ReadFile(pubPath)
			switch pubhalf {
			case "pub":
				if pubErr != nil {
					t.Fatalf("fixture claims pub but %s is unreadable: %v", pubPath, pubErr)
				}
				if _, _, _, _, err := ssh.ParseAuthorizedKey(pubBytes); err != nil {
					t.Fatalf("ParseAuthorizedKey(%s): %v", pubPath, err)
				}
			case "nopub":
				if pubErr == nil {
					t.Fatalf("fixture claims nopub but %s exists", pubPath)
				}
			default:
				t.Fatalf("fixture name %q has unrecognized pubhalf token %q", name, pubhalf)
			}

			// The PEM block Type matches the claimed format+algorithm.
			block, _ := pem.Decode(privBytes)
			if block == nil {
				t.Fatalf("%s: no PEM block found", privPath)
			}
			wantType := blockTypeFor(t, format, algo)
			if block.Type != wantType {
				t.Fatalf("%s: PEM block Type = %q, want %q", privPath, block.Type, wantType)
			}

			switch encryption {
			case "plain":
				key, err := ssh.ParseRawPrivateKey(privBytes)
				if err != nil {
					t.Fatalf("ParseRawPrivateKey(%s): unexpected error: %v", privPath, err)
				}
				assertRawType(t, algo, format, key)
			case "encrypted":
				_, err := ssh.ParseRawPrivateKey(privBytes)
				var missing *ssh.PassphraseMissingError
				if !errors.As(err, &missing) {
					t.Fatalf("ParseRawPrivateKey(%s): got err %v (%T), want *ssh.PassphraseMissingError", privPath, err, err)
				}
				switch format {
				case "openssh":
					if missing.PublicKey == nil {
						t.Fatalf("%s: PassphraseMissingError.PublicKey is nil, want non-nil for openssh format", privPath)
					}
				case "pem":
					if missing.PublicKey != nil {
						t.Fatalf("%s: PassphraseMissingError.PublicKey is non-nil, want nil for pem format", privPath)
					}
				}
			default:
				t.Fatalf("fixture name %q has unrecognized encryption token %q", name, encryption)
			}
		})
	}
}
