package fpscheme_test

import (
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/boweeb/hasp/internal/adapter/fpscheme"
)

// TestIsRSAAlgorithm pins the exported predicate chunk M3.6.4's --investigate origin-evidence
// rule (internal/app/investigate.go's originsForInvestigate) depends on to derive "is RSA" the
// same way importedRSAApplicable already does — so the two checks can never silently disagree.
func TestIsRSAAlgorithm(t *testing.T) {
	cases := map[string]bool{
		ssh.KeyAlgoRSA:     true,
		ssh.KeyAlgoED25519: false,
		"":                 false,
		"ssh-dss":          false,
	}
	for algo, want := range cases {
		if got := fpscheme.IsRSAAlgorithm(algo); got != want {
			t.Errorf("IsRSAAlgorithm(%q) = %v, want %v", algo, got, want)
		}
	}
}
