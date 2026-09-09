package fpscheme_test

import (
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fpscheme"
)

// TestCreatedRSAPromptWorthy is the reviewer-flagged fix this chunk (M3.6.2) makes: deciding, from
// public material alone, whether a passphrase would plausibly unlock aws-created-rsa (T39 rule
// 2) — without ever type-asserting m.Private the way createdRSAApplicable does, since that field
// is only ever populated after the very decrypt this function exists to help decide whether to
// attempt. Every case here is built with a nil passphrase (keyfile.OpenMaterial's own "public
// half only, never decrypt" contract), matching exactly how the passphrase-gate policy layer
// calls this before ever prompting.
func TestCreatedRSAPromptWorthy(t *testing.T) {
	cases := []struct {
		name            string
		fixture         string
		wantWorth       bool
		wantAmbiguous   bool
		wantDescription string
	}{
		{
			name:            "openssh-format encrypted RSA, no .pub: algorithm known from the embedded public half",
			fixture:         "rsa-openssh-encrypted-nopub",
			wantWorth:       true,
			wantAmbiguous:   false,
			wantDescription: "OpenSSH wire format embeds the public half unencrypted even when the key itself is encrypted (T1) — no ambiguity, no need to decrypt to find out",
		},
		{
			name:            "openssh-format encrypted ED25519, no .pub: known non-RSA, never worth prompting",
			fixture:         "ed25519-openssh-encrypted-nopub",
			wantWorth:       false,
			wantAmbiguous:   false,
			wantDescription: "same derivation-gap-free case as above, but the known algorithm answers 'no'",
		},
		{
			name:            "legacy PEM encrypted RSA, no .pub: algorithm undiscoverable without the passphrase",
			fixture:         "rsa-pem-encrypted-nopub",
			wantWorth:       true,
			wantAmbiguous:   true,
			wantDescription: "§5.1's total derivation gap: this function resolves the unavoidable ambiguity conservatively (treat as a candidate) per its own doc comment",
		},
		{
			name:            "legacy PEM encrypted DSA, no .pub: same derivation gap, different algorithm",
			fixture:         "dsa-pem-encrypted-nopub",
			wantWorth:       true,
			wantAmbiguous:   true,
			wantDescription: "the gap is about the *format*, not the algorithm — a DSA key hits it exactly the same way an RSA key does",
		},
		{
			name:            "openssh-format encrypted RSA, with .pub sidecar: sidecar supplies the same answer",
			fixture:         "rsa-openssh-encrypted-pub",
			wantWorth:       true,
			wantAmbiguous:   false,
			wantDescription: "a .pub sidecar is T1's other public-half source; either one closes the gap",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := openMaterial(t, tc.fixture)
			worth, ambiguous := fpscheme.CreatedRSAPromptWorthy(m)
			if worth != tc.wantWorth {
				t.Errorf("worthPrompting = %v, want %v (%s)", worth, tc.wantWorth, tc.wantDescription)
			}
			if ambiguous != tc.wantAmbiguous {
				t.Errorf("ambiguous = %v, want %v (%s)", ambiguous, tc.wantAmbiguous, tc.wantDescription)
			}
		})
	}
}
