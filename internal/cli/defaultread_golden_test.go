package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestDefaultRead_ListKey_Golden is M3.6.0's whole purpose (roadmap.md §5.6, chunk M3.6.0):
// capture hasp list key's pre-M3.6 output — human and --json — as a committed golden, while the
// tree still predates any Investigation code. Exit criterion 7 ("hasp list key without
// --investigate is byte-identical to its pre-M3.6 output") is asserted against this golden from
// M3.6.1 onward; this test is the proof the golden actually predates the feature, since it is
// committed in its own commit before any M3.6.1 code exists.
//
// The fixture is built from committed material only, the same discipline
// internal/cli/testdata/script/key-list.txtar already uses for its inline ed25519 key, plus one
// RSA key copied from testdata/keys/rsa-pem-plain-pub (+ .pub) so the golden exercises a key
// every later fingerprint scheme (T35) applies to, even though no scheme runs yet.
//
// Judgment call: --json's data carries each key's absolute Locations[].Path (domain.Key,
// internal/domain/key.go), which necessarily varies with t.TempDir()'s per-run directory name.
// Byte-for-byte golden comparison of --json therefore needs one normalization step — the
// fixture directory's absolute path is replaced with the literal token "<KEYDIR>" before
// comparing against (and generating) the golden file. Nothing else in either output form is
// normalized; the human table carries no paths at all, and everything else --json emits
// (fingerprints, sizes, names, algorithms) is a deterministic function of the fixture bytes.
func TestDefaultRead_ListKey_Golden(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "id_ed25519"), ed25519FixtureKeyPEM)

	rsaPriv, err := os.ReadFile(filepath.Join(repoFixturesDir(t), "rsa-pem-plain-pub"))
	if err != nil {
		t.Fatalf("read rsa-pem-plain-pub fixture: %v", err)
	}
	rsaPub, err := os.ReadFile(filepath.Join(repoFixturesDir(t), "rsa-pem-plain-pub.pub"))
	if err != nil {
		t.Fatalf("read rsa-pem-plain-pub.pub fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "id_rsa_legacy"), rsaPriv, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "id_rsa_legacy.pub"), rsaPub, 0o644); err != nil {
		t.Fatal(err)
	}

	human := runGolden(t, dir, "list", "key")
	jsonOut := normalizeKeyDir(runGolden(t, dir, "list", "key", "--json"), dir)

	assertGolden(t, filepath.Join("testdata", "golden", "list-key.human.golden"), human)
	assertGolden(t, filepath.Join("testdata", "golden", "list-key.json.golden"), jsonOut)
}

// ed25519FixtureKeyPEM is the same inline, throwaway OpenSSH ed25519 key
// internal/cli/testdata/script/key-list.txtar embeds — committed material, not generated at test
// time, so the fixture (and therefore the golden it produces) is stable across runs and across
// machines.
const ed25519FixtureKeyPEM = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACAEEd2lOVWo2Yf1SBuQBQ6kipAJm/tP0lwuxYKzGZ0nYgAAAIguUITVLlCE
1QAAAAtzc2gtZWQyNTUxOQAAACAEEd2lOVWo2Yf1SBuQBQ6kipAJm/tP0lwuxYKzGZ0nYg
AAAEDeq64CYCWqT5OaGWlM4yFGnPx2Oi600gc1LeQFjUfbFQQR3aU5VajZh/VIG5AFDqSK
kAmb+0/SXC7FgrMZnSdiAAAAAAECAwQF
-----END OPENSSH PRIVATE KEY-----
`

func runGolden(t *testing.T, keyDir string, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	root := cli.NewRootCmd()
	root.SetArgs(append(args, "--key-dir", keyDir))
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatalf("hasp %v: %v\n%s", args, err, out.String())
	}
	return out.String()
}

func normalizeKeyDir(s, keyDir string) string {
	return strings.ReplaceAll(s, keyDir, "<KEYDIR>")
}

// assertGolden compares got against the committed file at path, byte-for-byte. Set
// HASP_UPDATE_GOLDEN=1 to (re)write the golden from got — the same escape hatch this project's
// generated-reference staleness check (T34) leaves for a deliberate, reviewed regeneration; never
// set for an ordinary `go test` or CI run.
func assertGolden(t *testing.T, path, got string) {
	t.Helper()
	if os.Getenv("HASP_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with HASP_UPDATE_GOLDEN=1 to create it)", path, err)
	}
	if got != string(want) {
		t.Errorf("%s: output does not match golden (M3.6 roadmap exit criterion 7 — the default read must stay byte-identical)\n--- got ---\n%s\n--- want ---\n%s", path, got, string(want))
	}
}
