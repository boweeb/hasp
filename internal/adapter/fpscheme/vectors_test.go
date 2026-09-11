package fpscheme_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fpscheme"
	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/domain"
)

func fixturesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// this file lives at internal/adapter/fpscheme/vectors_test.go
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "testdata", "keys")
}

func openMaterial(t *testing.T, fixtureName string) keyfile.Material {
	t.Helper()
	mat, err := keyfile.OpenMaterial(filepath.Join(fixturesDir(t), fixtureName), nil)
	if err != nil {
		t.Fatalf("OpenMaterial(%s): %v", fixtureName, err)
	}
	return mat
}

// TestFingerprintVectors_CommittedValues asserts the fingerprint-scheme table against real,
// committed vectors (tdd.md §12, T35) — not invented ones. These four values were verified
// against AWS's own documented algorithms and confirmed on this repository's own fixtures during
// T35's own reassessment session; roadmap.md M3.6.1's spec repeats them verbatim as the exit
// bar for this chunk.
func TestFingerprintVectors_CommittedValues(t *testing.T) {
	rsaMat := openMaterial(t, "rsa-pem-plain-pub")
	ed25519Mat := openMaterial(t, "ed25519-openssh-plain-pub")

	cases := []struct {
		name string
		m    keyfile.Material
		id   domain.SchemeID
		want string
	}{
		{
			name: "rsa-pem-plain-pub/aws-created-rsa",
			m:    rsaMat,
			id:   domain.SchemeAWSCreatedRSA,
			want: "97:47:11:3c:af:56:47:b3:f9:a9:89:36:6d:ca:be:0b:33:a0:05:f7",
		},
		{
			name: "rsa-pem-plain-pub/aws-imported-rsa",
			m:    rsaMat,
			id:   domain.SchemeAWSImportedRSA,
			want: "a8:e7:45:95:5f:a3:f0:b1:79:6c:c2:f1:d2:80:57:ea",
		},
		{
			name: "rsa-pem-plain-pub/legacy-ssh-md5",
			m:    rsaMat,
			id:   domain.SchemeLegacySSHMD5,
			want: "34:29:f4:da:3c:db:49:4b:35:ba:c1:c2:cd:2e:75:a8",
		},
		{
			name: "ed25519-openssh-plain-pub/ssh-native-sha256",
			m:    ed25519Mat,
			id:   domain.SchemeSSHNativeSHA256,
			want: "SHA256:lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := fpscheme.ComputeOne(tc.id, tc.m)
			if !ok {
				t.Fatalf("scheme %s is not registered", tc.id)
			}
			if got.Confidence != domain.ConfidenceDerived {
				t.Fatalf("Confidence = %s, want derived (reason=%q)", got.Confidence, got.Reason)
			}
			if got.Value != tc.want {
				t.Errorf("Value = %q, want %q", got.Value, tc.want)
			}
		})
	}
}

// TestMD5Collision_ImportedRSAAndLegacyMD5 is T36's own guard: aws-imported-rsa and
// legacy-ssh-md5 must produce different values of identical shape for the same key, and a
// 47-character clue matching *either* value must route to a candidate set containing both
// schemes. A future refactor that "optimizes" the MD5 shape down to a single scheme fails this
// test loudly instead of silently missing every legacy clue (roadmap.md §5.6 exit criterion 1a).
func TestMD5Collision_ImportedRSAAndLegacyMD5(t *testing.T) {
	m := openMaterial(t, "rsa-pem-plain-pub")

	imported, ok := fpscheme.ComputeOne(domain.SchemeAWSImportedRSA, m)
	if !ok || imported.Confidence != domain.ConfidenceDerived {
		t.Fatalf("aws-imported-rsa did not compute: ok=%v confidence=%s reason=%q", ok, imported.Confidence, imported.Reason)
	}
	legacy, ok := fpscheme.ComputeOne(domain.SchemeLegacySSHMD5, m)
	if !ok || legacy.Confidence != domain.ConfidenceDerived {
		t.Fatalf("legacy-ssh-md5 did not compute: ok=%v confidence=%s reason=%q", ok, legacy.Confidence, legacy.Reason)
	}

	if imported.Value == legacy.Value {
		t.Fatalf("aws-imported-rsa and legacy-ssh-md5 must differ in value (T35/T36); both = %q", imported.Value)
	}
	if len(imported.Value) != len(legacy.Value) {
		t.Fatalf("expected identical shape: aws-imported-rsa=%d chars (%q), legacy-ssh-md5=%d chars (%q)",
			len(imported.Value), imported.Value, len(legacy.Value), legacy.Value)
	}

	for _, val := range []string{imported.Value, legacy.Value} {
		normalized := fpscheme.Normalize(val)
		if len(normalized) != 32 {
			t.Fatalf("Normalize(%q) = %q, want 32 hex chars", val, normalized)
		}
		candidates := fpscheme.CandidateSchemes(normalized)
		want := map[domain.SchemeID]bool{domain.SchemeAWSImportedRSA: true, domain.SchemeLegacySSHMD5: true}
		if len(candidates) != 2 {
			t.Fatalf("a 47-character clue (%q) must route to both MD5 schemes, got %v", val, candidates)
		}
		for _, c := range candidates {
			if !want[c] {
				t.Errorf("unexpected candidate scheme %s for clue %q", c, val)
			}
			delete(want, c)
		}
		if len(want) != 0 {
			t.Errorf("candidate set for clue %q is missing: %v", val, want)
		}
	}
}
