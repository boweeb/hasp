// Command genfixtures generates the 28 key fixtures under testdata/keys/ that gate M1's entire
// read path (docs/tdd.md §12, docs/roadmap.md §2). It is a development-time convenience, run
// via `go run ./tools/genfixtures` from the repo root — never imported by hasp itself, and never
// exec'd by hasp at runtime (T1's "no subprocess" rule governs hasp's own call graph, not how
// its test fixtures were produced once).
//
// The matrix: 3 algorithms (ed25519, rsa, ecdsa) x 2 formats (openssh, pem) x 2 encryption
// states (plain, encrypted) x 2 public-half states (pub, nopub) = 24, plus 4 hand-built DSA
// fixtures (PEM-only, T23).
//
// Every cell shells out to ssh-keygen except ed25519 x pem, which ssh-keygen cannot produce at
// all on modern OpenSSH (PEM has no Ed25519 representation; ssh-keygen never emits the bare
// PKCS8 "PRIVATE KEY" block that is Ed25519's actual PEM form) and DSA, which ssh-keygen refuses
// to generate outright (T23). Both are constructed directly via crypto/x509, encoding/pem, and
// (for DSA) encoding/asn1, following the same encrypted-legacy-PEM construction
// (x509.EncryptPEMBlock, Proc-Type/DEK-Info) that real ssh-keygen -m PEM output uses, so the
// derivation-gap behavior x/crypto/ssh exposes for encrypted legacy PEM (T1) is exercised
// identically regardless of which path produced the bytes.
package main

import (
	"crypto/dsa" //nolint:staticcheck // DSA is a legacy fixture-only case; see T23.
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509" //nolint:staticcheck // x509.EncryptPEMBlock is deprecated but is exactly the
	// mechanism T1's decryption-gap derivation depends on for legacy PEM; see the decision
	// record for why it is used deliberately here.
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// testPassphrase is a fixed, publicly-known constant used only to make an "encrypted" fixture
// genuinely encrypted, and to let the generator be re-run deterministically. hasp itself never
// decrypts a private key at runtime (P3/T1) — this constant has no other purpose.
const testPassphrase = "hasp-test-fixture-passphrase"

const outDir = "testdata/keys"

// pubHalf describes whether a fixture's .pub sibling should be written.
type pubHalf struct {
	name   string // "pub" or "nopub"
	suffix bool
}

var pubHalves = []pubHalf{{"pub", true}, {"nopub", false}}

// encState describes a fixture's encryption axis.
type encState struct {
	name       string // "plain" or "encrypted"
	encrypted  bool
	passphrase string
}

var encStates = []encState{
	{"plain", false, ""},
	{"encrypted", true, testPassphrase},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genfixtures:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	// 24-cell matrix: algorithm x format x encryption x public-half.
	for _, algo := range []string{"ed25519", "rsa", "ecdsa"} {
		for _, format := range []string{"openssh", "pem"} {
			for _, enc := range encStates {
				for _, pub := range pubHalves {
					name := fmt.Sprintf("%s-%s-%s-%s", algo, format, enc.name, pub.name)
					if err := generateMatrixCell(algo, format, enc, pub, name); err != nil {
						return fmt.Errorf("%s: %w", name, err)
					}
					fmt.Println("wrote", name)
				}
			}
		}
	}

	// 4 hand-built DSA fixtures, PEM-only (T23).
	for _, enc := range encStates {
		for _, pub := range pubHalves {
			name := fmt.Sprintf("dsa-pem-%s-%s", enc.name, pub.name)
			if err := generateDSA(enc, pub, name); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			fmt.Println("wrote", name)
		}
	}

	return nil
}

// generateMatrixCell dispatches a single (algo, format, enc, pub) cell to ssh-keygen, except
// ed25519 x pem, which is hand-constructed (see package doc).
func generateMatrixCell(algo, format string, enc encState, pub pubHalf, name string) error {
	if algo == "ed25519" && format == "pem" {
		return generateEd25519PEM(enc, pub, name)
	}
	return generateViaSSHKeygen(algo, format, enc, pub, name)
}

// generateViaSSHKeygen shells out to ssh-keygen for every matrix cell except ed25519 x pem and
// DSA (see package doc for why those two are hand-built instead).
func generateViaSSHKeygen(algo, format string, enc encState, pub pubHalf, name string) error {
	privPath := filepath.Join(outDir, name)
	pubPath := privPath + ".pub"

	// Clean any prior run's output so re-generation is idempotent.
	_ = os.Remove(privPath)
	_ = os.Remove(pubPath)

	args := []string{"-t", algo, "-f", privPath, "-N", enc.passphrase, "-C", ""}
	if format == "pem" {
		args = append(args, "-m", "PEM")
	}

	cmd := exec.Command("ssh-keygen", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-keygen %v: %w\n%s", args, err, out)
	}

	if !pub.suffix {
		if err := os.Remove(pubPath); err != nil {
			return fmt.Errorf("remove %s: %w", pubPath, err)
		}
	}

	return nil
}

// generateEd25519PEM hand-constructs an ed25519-pem-* fixture (see package doc): plain is bare
// PKCS8 ("PRIVATE KEY"); encrypted applies the legacy RFC 1423 Proc-Type/DEK-Info mechanism to
// that same block type, which is what makes x/crypto/ssh's ParseRawPrivateKey hit the encrypted-
// legacy-PEM derivation-gap row (PublicKey == nil) instead of the OpenSSH row.
func generateEd25519PEM(enc encState, pub pubHalf, name string) error {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate ed25519 key: %w", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("marshal pkcs8: %w", err)
	}

	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	if enc.encrypted {
		block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(enc.passphrase), x509.PEMCipherAES256) //nolint:staticcheck // deliberate legacy-PEM encryption, see package doc.
		if err != nil {
			return fmt.Errorf("encrypt pem block: %w", err)
		}
	}

	if err := writePrivateKey(name, block); err != nil {
		return err
	}
	if !pub.suffix {
		return nil
	}

	sshPub, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("new public key: %w", err)
	}
	return writePublicKey(name, sshPub)
}

// dsaOpenSSLPrivateKey mirrors x/crypto/ssh's own ParseDSAPrivateKey struct exactly (verified
// against ssh/keys.go this session): the ASN.1 DER shape OpenSSL's "DSA PRIVATE KEY" PEM type
// uses.
type dsaOpenSSLPrivateKey struct {
	Version int
	P, Q, G *big.Int
	Pub     *big.Int
	Priv    *big.Int
}

// generateDSA hand-constructs a dsa-pem-* fixture. ssh-keygen refuses to generate DSA at all on
// modern OpenSSH (T23), so this goes directly through crypto/dsa + encoding/asn1.
func generateDSA(enc encState, pub pubHalf, name string) error {
	var params dsa.Parameters
	if err := dsa.GenerateParameters(&params, rand.Reader, dsa.L1024N160); err != nil {
		return fmt.Errorf("generate dsa parameters: %w", err)
	}

	priv := &dsa.PrivateKey{PublicKey: dsa.PublicKey{Parameters: params}}
	if err := dsa.GenerateKey(priv, rand.Reader); err != nil {
		return fmt.Errorf("generate dsa key: %w", err)
	}

	der, err := asn1.Marshal(dsaOpenSSLPrivateKey{
		Version: 0,
		P:       priv.P,
		Q:       priv.Q,
		G:       priv.G,
		Pub:     priv.Y,
		Priv:    priv.X,
	})
	if err != nil {
		return fmt.Errorf("marshal dsa asn1: %w", err)
	}

	block := &pem.Block{Type: "DSA PRIVATE KEY", Bytes: der}
	if enc.encrypted {
		block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(enc.passphrase), x509.PEMCipherAES256) //nolint:staticcheck // deliberate legacy-PEM encryption, see package doc.
		if err != nil {
			return fmt.Errorf("encrypt pem block: %w", err)
		}
	}

	if err := writePrivateKey(name, block); err != nil {
		return err
	}
	if !pub.suffix {
		return nil
	}

	sshPub, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		return fmt.Errorf("new public key: %w", err)
	}
	return writePublicKey(name, sshPub)
}

func writePrivateKey(name string, block *pem.Block) error {
	path := filepath.Join(outDir, name)
	return os.WriteFile(path, pem.EncodeToMemory(block), 0o600)
}

func writePublicKey(name string, pub ssh.PublicKey) error {
	path := filepath.Join(outDir, name+".pub")
	return os.WriteFile(path, ssh.MarshalAuthorizedKey(pub), 0o644)
}
