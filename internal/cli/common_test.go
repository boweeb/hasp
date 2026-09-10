package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/app"
)

// commonTestFixturesDir mirrors fullinventory_test.go's own repoFixturesDir (internal/cli's
// cli_test external test package): a package-cli file cannot see symbols a package cli_test file
// defines even though both live under the same directory (Go's one-way test-package visibility,
// writeguard_alltree_test.go's own comment explains it too), so this small helper is duplicated
// here rather than exported across that boundary for one line of logic.
func commonTestFixturesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "keys")
}

func commonTestCopyFixture(t *testing.T, fixturesDir, name, destPath string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixturesDir, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destPath, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestResolveKeyClue_EncryptedKeyUnderCreatedRSAClue_ReportsUnevaluableScheme is item 1's
// regression test (M3.6.3 review): resolveKeyClue backs adopt/release/edit/new --key, all
// single-key lookups, not just find key's own multi-match report. Before the fix, a clue shaped
// like an AWS-console SHA-1 (aws-created-rsa, T35) against an encrypted RSA key — which
// aws-created-rsa can never evaluate without a passphrase find/resolveKeyClue never ask for
// (T48, T39) — landed on FindKeys' case-0 branch with T48's warning silently discarded, producing
// a bare "not found" indistinguishable from "you don't have this key" (D20's own failure mode,
// reproduced one layer in, docs/decision-log.md:1259-1262). It must instead name the unevaluable
// scheme.
func TestResolveKeyClue_EncryptedKeyUnderCreatedRSAClue_ReportsUnevaluableScheme(t *testing.T) {
	fixturesDir := commonTestFixturesDir(t)
	dir := t.TempDir()
	commonTestCopyFixture(t, fixturesDir, "rsa-openssh-encrypted-nopub", filepath.Join(dir, "id_rsa_encrypted"))

	m, err := app.Derive(app.DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}

	// A syntactically well-formed 40-hex-digit (59-character colon-hex) clue — T36's own
	// created-RSA shape — that will not actually match this key even if it could be evaluated;
	// the point is proving the unevaluable-scheme signal survives to the caller, not that it
	// matches.
	const clue = "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd"

	_, err = resolveKeyClue(m, clue)
	if err == nil {
		t.Fatal("resolveKeyClue: want an error, got nil")
	}
	if !strings.Contains(err.Error(), "aws-created-rsa") {
		t.Errorf("resolveKeyClue error = %q, want it to name the unevaluable scheme (aws-created-rsa, T48) rather than a bare not-found", err.Error())
	}
}
