package fpscheme

import (
	"crypto/md5" //nolint:gosec // MD5 is the scheme itself, not hasp's cryptography — T35/T36.
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // SHA-1 is the scheme itself, not hasp's cryptography — T35.
	"crypto/x509"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/domain"
)

// MaterialRequirement is T35's "load-bearing field": which key material a scheme needs before it
// can compute anything at all. It is what lets a caller (D19's consent gate; chunk M3.6.2's
// passphrase prompt) enumerate, in advance, which schemes are already computable from what has
// been read and which remain unknown until the user consents to more.
type MaterialRequirement int

const (
	// RequiresPublicHalf means the scheme only ever reads keyfile.Material.Public.
	RequiresPublicHalf MaterialRequirement = iota
	// RequiresDecryptedPrivateKey means the scheme reads keyfile.Material.Private, which is
	// only ever populated under D19's consent gate (keyfile.OpenMaterial's own doc comment).
	RequiresDecryptedPrivateKey
)

// Scheme is one registered fingerprint scheme (T35): a stable id, its declared material
// requirement, and the two functions that decide whether it applies to a given key's material
// and, when it does, compute its value. Applicable and Compute are exported struct fields, not
// hidden behind an interface, so the registry itself stays "one new entry" (the open-registry
// rule T35 states explicitly) rather than a switch statement that grows a new case per scheme.
type Scheme struct {
	ID         domain.SchemeID
	Requires   MaterialRequirement
	Applicable func(keyfile.Material) (ok bool, reason domain.ReasonToken)
	Compute    func(keyfile.Material) (value string, err error)
}

// registry is the open, ordered set of every scheme fpscheme knows (T35's table, tdd.md §18).
// Adding a fifth scheme is one new entry here — never a new call site elsewhere in the tree,
// since every caller goes through ComputeAll/ComputeOne/Schemes below.
var registry = []Scheme{
	{
		ID:         domain.SchemeAWSCreatedRSA,
		Requires:   RequiresDecryptedPrivateKey,
		Applicable: createdRSAApplicable,
		Compute:    createdRSACompute,
	},
	{
		ID:         domain.SchemeAWSImportedRSA,
		Requires:   RequiresPublicHalf,
		Applicable: importedRSAApplicable,
		Compute:    importedRSACompute,
	},
	{
		ID:         domain.SchemeSSHNativeSHA256,
		Requires:   RequiresPublicHalf,
		Applicable: publicHalfApplicable,
		Compute:    sshNativeSHA256Compute,
	},
	{
		ID:         domain.SchemeLegacySSHMD5,
		Requires:   RequiresPublicHalf,
		Applicable: publicHalfApplicable,
		Compute:    legacySSHMD5Compute,
	},
}

// Schemes returns a defensive copy of the registered scheme set, in the fixed order above.
func Schemes() []Scheme {
	out := make([]Scheme, len(registry))
	copy(out, registry)
	return out
}

// ComputeAll runs every registered scheme against m and returns one domain.SchemeFingerprint per
// scheme — never fewer. roadmap.md §5.6 exit criterion 2: a scheme that could not be computed
// reports the honest "could not compute" shape (Value == "", Confidence == unknown, a non-empty
// Reason) rather than being left out of the slice; no scheme is ever silently omitted.
func ComputeAll(m keyfile.Material) []domain.SchemeFingerprint {
	out := make([]domain.SchemeFingerprint, 0, len(registry))
	for _, s := range registry {
		out = append(out, computeOne(s, m))
	}
	return out
}

// ComputeOne runs the single named scheme against m. ok is false when id names a scheme this
// registry does not know at all (a genuinely unregistered id, not the "could not compute" case —
// that is still ok == true, with the returned SchemeFingerprint carrying ConfidenceUnknown).
func ComputeOne(id domain.SchemeID, m keyfile.Material) (fp domain.SchemeFingerprint, ok bool) {
	for _, s := range registry {
		if s.ID == id {
			return computeOne(s, m), true
		}
	}
	return domain.SchemeFingerprint{}, false
}

func computeOne(s Scheme, m keyfile.Material) domain.SchemeFingerprint {
	if applicable, reason := s.Applicable(m); !applicable {
		return domain.SchemeFingerprint{Scheme: s.ID, Confidence: domain.ConfidenceUnknown, Reason: reason}
	}
	value, err := s.Compute(m)
	if err != nil {
		// Applicable already vetted algorithm and material presence; a Compute error past
		// that point is Go's own encoder refusing the key's specific encoding — DSA against
		// x509.MarshalPKIXPublicKey is T35's own named example. Honest unknown, not a panic
		// and not an omission (T35).
		return domain.SchemeFingerprint{Scheme: s.ID, Confidence: domain.ConfidenceUnknown, Reason: domain.ReasonKeyEncodingUnsupported}
	}
	return domain.SchemeFingerprint{Scheme: s.ID, Value: value, Confidence: domain.ConfidenceDerived}
}

// --- aws-created-rsa: SHA-1 over the PKCS#8 DER of the decrypted private key ---

func createdRSAApplicable(m keyfile.Material) (bool, domain.ReasonToken) {
	if m.Private == nil {
		return false, domain.ReasonPrivateKeyUnavailable
	}
	if _, ok := m.Private.(*rsa.PrivateKey); !ok {
		return false, domain.ReasonSchemeNotApplicable
	}
	return true, ""
}

func createdRSACompute(m keyfile.Material) (string, error) {
	priv, ok := m.Private.(*rsa.PrivateKey)
	if !ok {
		// Applicable is always checked first (computeOne); reachable only if a future
		// refactor calls Compute directly, so this stays an honest error, not a panic.
		return "", fmt.Errorf("aws-created-rsa: material is not an *rsa.PrivateKey")
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(der)
	return colonHex(sum[:]), nil
}

// --- aws-imported-rsa: MD5 over the PKIX/SPKI DER of the public key ---

func importedRSAApplicable(m keyfile.Material) (bool, domain.ReasonToken) {
	if m.Public == nil {
		return false, domain.ReasonPublicHalfUnavailable
	}
	if m.Public.Type() != ssh.KeyAlgoRSA {
		return false, domain.ReasonSchemeNotApplicable
	}
	return true, ""
}

func importedRSACompute(m keyfile.Material) (string, error) {
	cpk, ok := m.Public.(ssh.CryptoPublicKey)
	if !ok {
		return "", fmt.Errorf("aws-imported-rsa: public key does not expose a crypto.PublicKey")
	}
	der, err := x509.MarshalPKIXPublicKey(cpk.CryptoPublicKey())
	if err != nil {
		// DSA's genuine x509.MarshalPKIXPublicKey failure would land here if it ever
		// reached Compute; in practice importedRSAApplicable's RSA-only check already
		// routes a DSA key to ReasonSchemeNotApplicable first (T35's DSA example is
		// satisfied either way: an honest unknown, never a panic, never an omission).
		return "", err
	}
	sum := md5.Sum(der) //nolint:gosec // the scheme itself is MD5 — T35/T36.
	return colonHex(sum[:]), nil
}

// --- ssh-native-sha256 and legacy-ssh-md5 share the same material requirement ---

func publicHalfApplicable(m keyfile.Material) (bool, domain.ReasonToken) {
	if m.Public == nil {
		return false, domain.ReasonPublicHalfUnavailable
	}
	return true, ""
}

// ssh-native-sha256: SHA-256 over the SSH wire-format public key, rendered in
// ssh.FingerprintSHA256's own "SHA256:<base64>" form (T1, T35).
func sshNativeSHA256Compute(m keyfile.Material) (string, error) {
	return ssh.FingerprintSHA256(m.Public), nil
}

// legacy-ssh-md5: MD5 over the SSH wire-format public key — what `ssh-keygen -E md5` prints, and
// what OpenSSH printed by default before 6.8 (T1, T35, T36).
func legacySSHMD5Compute(m keyfile.Material) (string, error) {
	sum := md5.Sum(m.Public.Marshal()) //nolint:gosec // the scheme itself is MD5 — T35/T36.
	return colonHex(sum[:]), nil
}

// colonHex renders b as lowercase colon-separated hex — the SHA-1 and MD5 schemes' shared
// display form (T35's table; contrast ssh-native-sha256's own "SHA256:<base64>" form).
func colonHex(b []byte) string {
	parts := make([]string, len(b))
	for i, c := range b {
		parts[i] = fmt.Sprintf("%02x", c)
	}
	return strings.Join(parts, ":")
}
