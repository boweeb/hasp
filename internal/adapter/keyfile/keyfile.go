package keyfile

import (
	"crypto/dsa" //nolint:staticcheck // DSA is a legacy fixture-only case; see T23.
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"

	"github.com/boweeb/hasp/internal/domain"
)

// Info is everything Inspect can determine about a private key file without ever decrypting it
// (T1, P3).
type Info struct {
	Format        domain.KeyFormat
	Algorithm     string             // ssh wire algorithm name, e.g. "ssh-ed25519"; "" if undecidable
	Bits          int                // 0 when not applicable or undecidable
	Encrypted     bool
	HasPublicHalf bool               // a .pub sibling file exists next to the private key
	Fingerprint   domain.Fingerprint // "" means undecidable (§5.1); populated below
	Comment       string             // from the .pub sibling; "" if absent
	SizeBytes     int64              // the private key file's own byte length — always derivable
	// without decrypting anything, and the "same-size" half of T12's "same-format, same-size"
	// unconfirmed-duplicate heuristic for undecidable keys, whose Bits stays 0 (Algorithm is
	// unavailable too in that case, so bit-size can't stand in for it).
}

// Inspect reads the private key file at path and derives everything about it that does not
// require a passphrase: format, algorithm, size, encryption state, fingerprint, and comment
// (T1). It never attempts to decrypt anything — no code path here calls anything that supplies
// a passphrase to an *existing* key (P3) — and the derivation-gap case (legacy PEM, encrypted,
// no public half) is reported as Fingerprint == "", never guessed and never prompted for (D12).
//
// Inspect returns an error when path does not contain a recognizable private key. Callers scan-
// ning a directory of candidate files must treat that as fail-open, per §11: an unreadable key
// file mid-scan is reported and the survey continues, never aborted.
func Inspect(path string) (Info, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Info{}, fmt.Errorf("read %s: %w", path, err)
	}

	block, _ := pem.Decode(raw)
	if block == nil {
		return Info{}, fmt.Errorf("%s: not a PEM-encoded private key", path)
	}

	info := Info{Format: formatForBlockType(block.Type), SizeBytes: int64(len(raw))}

	pubPath := path + ".pub"
	pubBytes, pubErr := os.ReadFile(pubPath)
	if pubErr == nil {
		info.HasPublicHalf = true
	}

	key, err := ssh.ParseRawPrivateKey(raw)
	switch {
	case err == nil:
		info.Encrypted = false
		info.Algorithm, info.Bits = classify(key)
		if signer, sErr := ssh.NewSignerFromKey(key); sErr == nil {
			info.Fingerprint = domain.Fingerprint(ssh.FingerprintSHA256(signer.PublicKey()))
		}
	default:
		var missing *ssh.PassphraseMissingError
		if !errors.As(err, &missing) {
			return Info{}, fmt.Errorf("%s: %w", path, err)
		}
		info.Encrypted = true
		if missing.PublicKey != nil {
			// OpenSSH wire format embeds the public half unencrypted (T1's second derivation-
			// gap row) — algorithm, size, and fingerprint are all still derivable.
			info.Algorithm = missing.PublicKey.Type()
			info.Bits = bitsFromPublicKey(missing.PublicKey)
			info.Fingerprint = domain.Fingerprint(ssh.FingerprintSHA256(missing.PublicKey))
		}
		// missing.PublicKey == nil is the derivation gap itself (T1's third row: legacy PEM,
		// encrypted, no public half): Algorithm/Bits/Fingerprint stay at their zero values.
	}

	if info.HasPublicHalf {
		if pub, comment, _, _, aErr := ssh.ParseAuthorizedKey(pubBytes); aErr == nil {
			info.Comment = comment
			// The .pub sidecar is authoritative for fingerprint whenever it parses — it never
			// requires a passphrase and is the documented fast path for the common case (T1).
			info.Fingerprint = domain.Fingerprint(ssh.FingerprintSHA256(pub))
			if info.Algorithm == "" {
				info.Algorithm = pub.Type()
			}
			if info.Bits == 0 {
				info.Bits = bitsFromPublicKey(pub)
			}
		}
	}

	return info, nil
}

func formatForBlockType(t string) domain.KeyFormat {
	switch t {
	case "OPENSSH PRIVATE KEY":
		return domain.FormatOpenSSH
	case "RSA PRIVATE KEY":
		return domain.FormatPKCS1
	case "EC PRIVATE KEY":
		return domain.FormatSEC1
	case "DSA PRIVATE KEY":
		return domain.FormatDSA
	case "PRIVATE KEY":
		return domain.FormatPKCS8 // bare PKCS8 — ed25519's only PEM shape (no dedicated block type)
	default:
		return domain.FormatUnknown
	}
}

// classify derives the SSH wire algorithm name and bit size from ssh.ParseRawPrivateKey's
// concrete return type — only reachable for a plain (unencrypted) key.
func classify(key any) (algorithm string, bits int) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return ssh.KeyAlgoRSA, k.N.BitLen()
	case *ecdsa.PrivateKey:
		bitSize := k.Curve.Params().BitSize
		return ecdsaAlgoForBits(bitSize), bitSize
	case *ed25519.PrivateKey: // OpenSSH wire format's ed25519 shape
		return ssh.KeyAlgoED25519, 256
	case ed25519.PrivateKey: // bare PKCS8 PEM's ed25519 shape
		return ssh.KeyAlgoED25519, 256
	case *dsa.PrivateKey:
		return ssh.KeyAlgoDSA, k.P.BitLen() //nolint:staticcheck // DSA is a legacy read-only case; see T23.
	default:
		return "", 0
	}
}

func ecdsaAlgoForBits(bits int) string {
	switch bits {
	case 256:
		return ssh.KeyAlgoECDSA256
	case 384:
		return ssh.KeyAlgoECDSA384
	case 521:
		return ssh.KeyAlgoECDSA521
	default:
		return ""
	}
}

// bitsFromPublicKey extracts a bit size from an ssh.PublicKey via the optional
// ssh.CryptoPublicKey interface, when the public key was recovered without any private
// material (the encrypted-OpenSSH derivation-gap row, or a parsed .pub sidecar).
func bitsFromPublicKey(pk ssh.PublicKey) int {
	cpk, ok := pk.(ssh.CryptoPublicKey)
	if !ok {
		return 0
	}
	switch k := cpk.CryptoPublicKey().(type) {
	case *rsa.PublicKey:
		return k.N.BitLen()
	case *ecdsa.PublicKey:
		return k.Curve.Params().BitSize
	case ed25519.PublicKey:
		return 256
	case *dsa.PublicKey: //nolint:staticcheck // DSA is a legacy read-only case; see T23.
		return k.P.BitLen()
	default:
		return 0
	}
}
