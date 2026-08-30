package domain

import (
	"encoding/json"
	"testing"
)

func TestKeyIdentity_FingerprintKind(t *testing.T) {
	id := NewFingerprintIdentity("SHA256:abc123")
	if id.Kind() != IdentityFingerprint {
		t.Errorf("Kind() = %v, want IdentityFingerprint", id.Kind())
	}
	if id.Value() != "SHA256:abc123" {
		t.Errorf("Value() = %q, want %q", id.Value(), "SHA256:abc123")
	}
	if id.String() != id.Value() {
		t.Errorf("String() = %q, want %q", id.String(), id.Value())
	}
}

func TestKeyIdentity_PathKind(t *testing.T) {
	id := NewPathIdentity("/home/jesse/.ssh/id_rsa_old")
	if id.Kind() != IdentityPath {
		t.Errorf("Kind() = %v, want IdentityPath", id.Kind())
	}
	if id.Value() != "/home/jesse/.ssh/id_rsa_old" {
		t.Errorf("Value() = %q, want the resolved path", id.Value())
	}
}

// TestKeyIdentity_Uniqueness confirms two identities are indistinguishable as map keys iff they
// carry the same kind and value — the mechanism app's derivation pipeline (Stage 11) uses to
// group key candidates sharing a fingerprint into one Key with multiple Locations (T12).
func TestKeyIdentity_Uniqueness(t *testing.T) {
	set := map[KeyIdentity]bool{}
	a := NewFingerprintIdentity("SHA256:same")
	b := NewFingerprintIdentity("SHA256:same")
	c := NewPathIdentity("SHA256:same") // same string, different kind: must NOT collide with a/b
	set[a] = true
	set[b] = true
	set[c] = true
	if len(set) != 2 {
		t.Fatalf("expected 2 distinct identities (fingerprint vs. path), got %d", len(set))
	}
}

func TestKeyIdentity_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		id   KeyIdentity
		want string
	}{
		{"fingerprint", NewFingerprintIdentity("SHA256:abc123"), `{"kind":"fingerprint","value":"SHA256:abc123"}`},
		{"path", NewPathIdentity("/home/jesse/.ssh/id_rsa_old"), `{"kind":"path","value":"/home/jesse/.ssh/id_rsa_old"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.id)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("json.Marshal = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestIdentityKind_String(t *testing.T) {
	if IdentityFingerprint.String() != "fingerprint" {
		t.Errorf("IdentityFingerprint.String() = %q", IdentityFingerprint.String())
	}
	if IdentityPath.String() != "path" {
		t.Errorf("IdentityPath.String() = %q", IdentityPath.String())
	}
}
