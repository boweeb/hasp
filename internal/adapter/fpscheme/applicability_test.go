package fpscheme_test

import (
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fpscheme"
	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/domain"
)

// TestComputeAll_NeverOmitsAScheme is roadmap.md §5.6 exit criterion 2's own guard, exercised
// directly against the registry rather than only through the CLI: every registered scheme must
// appear in ComputeAll's output for every kind of material, never silently dropped.
func TestComputeAll_NeverOmitsAScheme(t *testing.T) {
	cases := []struct {
		name string
		m    keyfile.Material
	}{
		{name: "zero-value material (total derivation gap)", m: keyfile.Material{}},
		{name: "rsa, public+private", m: openMaterial(t, "rsa-pem-plain-pub")},
		{name: "ed25519, public+private", m: openMaterial(t, "ed25519-openssh-plain-pub")},
		{name: "dsa, public half only", m: openMaterial(t, "dsa-pem-plain-pub")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fps := fpscheme.ComputeAll(tc.m)
			if len(fps) != len(fpscheme.Schemes()) {
				t.Fatalf("ComputeAll returned %d fingerprints, want %d (one per registered scheme): %+v", len(fps), len(fpscheme.Schemes()), fps)
			}
			seen := map[domain.SchemeID]bool{}
			for _, fp := range fps {
				seen[fp.Scheme] = true
				// The "could not compute" shape is exact: Value == "" iff Confidence ==
				// unknown, and unknown always carries a non-empty machine-readable reason.
				switch fp.Confidence {
				case domain.ConfidenceUnknown:
					if fp.Value != "" {
						t.Errorf("%s: unknown but Value = %q, want empty", fp.Scheme, fp.Value)
					}
					if fp.Reason == "" {
						t.Errorf("%s: unknown but Reason is empty", fp.Scheme)
					}
				case domain.ConfidenceDerived:
					if fp.Value == "" {
						t.Errorf("%s: derived but Value is empty", fp.Scheme)
					}
					if fp.Reason != "" {
						t.Errorf("%s: derived but Reason = %q, want empty", fp.Scheme, fp.Reason)
					}
				default:
					t.Errorf("%s: Confidence = %s, want derived or unknown from ComputeAll", fp.Scheme, fp.Confidence)
				}
			}
			for _, s := range fpscheme.Schemes() {
				if !seen[s.ID] {
					t.Errorf("scheme %s missing from ComputeAll's output entirely", s.ID)
				}
			}
		})
	}
}

// TestAWSRSASchemes_NonRSAKey_ScopedNotOmitted proves the "applies only to RSA keys" rule
// (T35): an ED25519 key must report both AWS RSA schemes as an honest unknown with a reason,
// never silently drop them and never emit a bogus value.
func TestAWSRSASchemes_NonRSAKey_ScopedNotOmitted(t *testing.T) {
	m := openMaterial(t, "ed25519-openssh-plain-pub")

	for _, id := range []domain.SchemeID{domain.SchemeAWSCreatedRSA, domain.SchemeAWSImportedRSA} {
		fp, ok := fpscheme.ComputeOne(id, m)
		if !ok {
			t.Fatalf("scheme %s not registered", id)
		}
		if fp.Confidence != domain.ConfidenceUnknown {
			t.Errorf("%s against an ED25519 key: Confidence = %s, want unknown", id, fp.Confidence)
		}
		if fp.Value != "" {
			t.Errorf("%s against an ED25519 key: Value = %q, want empty", id, fp.Value)
		}
		if fp.Reason == "" {
			t.Errorf("%s against an ED25519 key: Reason is empty, want a machine-readable token", id)
		}
	}
}

// TestAWSImportedRSA_DSAKey_HonestUnknown is T35's own named example: "DSA public keys will
// fail x509.MarshalPKIXPublicKey — that too is an honest unknown with a reason, not a panic and
// not an omission." Whether DSA is filtered by the RSA-only applicability check or (in a future
// refactor) reaches Compute and fails there, the observable contract is identical and is what
// this test pins: no panic, a populated Reason, an empty Value.
func TestAWSImportedRSA_DSAKey_HonestUnknown(t *testing.T) {
	m := openMaterial(t, "dsa-pem-plain-pub")

	fp, ok := fpscheme.ComputeOne(domain.SchemeAWSImportedRSA, m)
	if !ok {
		t.Fatal("scheme aws-imported-rsa not registered")
	}
	if fp.Confidence != domain.ConfidenceUnknown {
		t.Errorf("aws-imported-rsa against a DSA key: Confidence = %s, want unknown", fp.Confidence)
	}
	if fp.Value != "" {
		t.Errorf("aws-imported-rsa against a DSA key: Value = %q, want empty", fp.Value)
	}
	if fp.Reason == "" {
		t.Error("aws-imported-rsa against a DSA key: Reason is empty, want a machine-readable token")
	}
}

// TestAWSCreatedRSA_NoPrivateKey_HonestUnknown proves the material-availability half of the
// contract: a public-half-only Material (passphrase == nil against an encrypted key, or a plain
// keyfile.Material the caller built with Private left nil) never panics and never omits the
// scheme.
func TestAWSCreatedRSA_NoPrivateKey_HonestUnknown(t *testing.T) {
	m := keyfile.Material{} // no material at all — the total derivation gap

	fp, ok := fpscheme.ComputeOne(domain.SchemeAWSCreatedRSA, m)
	if !ok {
		t.Fatal("scheme aws-created-rsa not registered")
	}
	if fp.Confidence != domain.ConfidenceUnknown {
		t.Errorf("Confidence = %s, want unknown", fp.Confidence)
	}
	if fp.Reason != domain.ReasonPrivateKeyUnavailable {
		t.Errorf("Reason = %q, want %q", fp.Reason, domain.ReasonPrivateKeyUnavailable)
	}
}

// TestPublicHalfSchemes_NoPublicHalf_HonestUnknown proves the same for the three
// public-half-only schemes against §5.1's total derivation gap (legacy PEM, encrypted, no .pub
// sidecar): Public stays nil, and every scheme that needs it reports unknown with a reason.
func TestPublicHalfSchemes_NoPublicHalf_HonestUnknown(t *testing.T) {
	m := keyfile.Material{}

	for _, id := range []domain.SchemeID{domain.SchemeAWSImportedRSA, domain.SchemeSSHNativeSHA256, domain.SchemeLegacySSHMD5} {
		fp, ok := fpscheme.ComputeOne(id, m)
		if !ok {
			t.Fatalf("scheme %s not registered", id)
		}
		if fp.Confidence != domain.ConfidenceUnknown {
			t.Errorf("%s: Confidence = %s, want unknown", id, fp.Confidence)
		}
		if fp.Reason != domain.ReasonPublicHalfUnavailable {
			t.Errorf("%s: Reason = %q, want %q", id, fp.Reason, domain.ReasonPublicHalfUnavailable)
		}
	}
}

// TestComputeOne_UnregisteredScheme_NotOK proves ComputeOne's ok=false case is reserved for a
// genuinely unregistered id, distinct from the "could not compute" case (which is still ok ==
// true, per T35's registry design).
func TestComputeOne_UnregisteredScheme_NotOK(t *testing.T) {
	_, ok := fpscheme.ComputeOne(domain.SchemeID("not-a-real-scheme"), keyfile.Material{})
	if ok {
		t.Error("ComputeOne on an unregistered scheme id: ok = true, want false")
	}
}
