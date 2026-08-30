package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestListKey_OneScreen guards P7's "answer the question in one screen" against regression: a
// table row must never grow unbounded with the number of keys/hosts scanned, or a personal-scale
// ~/.ssh (P8 — tens of keys, not thousands) stops being readable in a normal terminal. This
// fixture mirrors design.md §2's own motivating inventory: several algorithms, both formats,
// a mix of managed and unmanaged, and an alias.
func TestListKey_OneScreen(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "id_ed25519"), fixtureKeyPEM)
	realFoobarCo := filepath.Join(dir, "work", "foobarco", "id_ed25519_foobarco")
	writeFile(t, realFoobarCo, fixtureKeyPEM)
	writeFile(t, filepath.Join(dir, "work", "foobarco", ".hasp"), "# marker\n")
	if err := os.Symlink(realFoobarCo, filepath.Join(dir, "id_ed25519_foobarco")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	root := cli.NewRootCmd()
	root.SetArgs([]string{"list", "key", "--key-dir", dir})
	root.SetOut(&out)
	if err := root.Execute(); err != nil {
		t.Fatalf("list key: %v", err)
	}

	const maxWidth = 120 // a common terminal default; the table must never exceed it per row
	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		if len(line) > maxWidth {
			t.Errorf("list key row exceeds %d columns (%d): %q", maxWidth, len(line), line)
		}
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

const fixtureKeyPEM = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACAEEd2lOVWo2Yf1SBuQBQ6kipAJm/tP0lwuxYKzGZ0nYgAAAIguUITVLlCE
1QAAAAtzc2gtZWQyNTUxOQAAACAEEd2lOVWo2Yf1SBuQBQ6kipAJm/tP0lwuxYKzGZ0nYg
AAAEDeq64CYCWqT5OaGWlM4yFGnPx2Oi600gc1LeQFjUfbFQQR3aU5VajZh/VIG5AFDqSK
kAmb+0/SXC7FgrMZnSdiAAAAAAECAwQF
-----END OPENSSH PRIVATE KEY-----
`
