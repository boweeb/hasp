package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestWriteNothingGuard_Smoke proves the guard mechanism itself works, using the only command
// that exists ahead of the noun grid. It is deliberately minimal — the full guard, exercised
// against every read command across a realistic fixture tree, is Stage 22's close-out pass;
// wiring it here first means every later stage can extend the same test rather than invent it
// from scratch under deadline.
func TestWriteNothingGuard_Smoke(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keyDir, "id_ed25519"), []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	before := snapshotTree(t, home)

	for _, args := range [][]string{
		{"version"},
		{"list", "key", "--key-dir", keyDir},
		{"list", "key", "--key-dir", keyDir, "--json"},
		{"show", "key", "id_ed25519", "--key-dir", keyDir},
		{"find", "key", "id_ed25519", "--key-dir", keyDir},
		{"check", "key", "--key-dir", keyDir},
		{"list", "host", "--key-dir", keyDir},
		{"find", "host", "id_ed25519", "--key-dir", keyDir},
		{"check", "host", "--key-dir", keyDir},
	} {
		root := cli.NewRootCmd()
		root.SetArgs(args)
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		_ = root.Execute() // check legitimately returns a non-nil error (findings); ignored here
	}

	after := snapshotTree(t, home)
	assertTreeUnchanged(t, before, after)
}
