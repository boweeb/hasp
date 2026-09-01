package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestWriteNothingOutsideKeyDir_EditKey extends tdd.md §12's write-nothing guard to `edit key`,
// exercising every sub-operation in one pass: rename, move-between-profiles, add-alias,
// remove-alias, and replace-material must all land inside --key-dir, and the settings file hasp
// only ever reads must never be created (D16).
func TestWriteNothingOutsideKeyDir_EditKey(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	workDir := filepath.Join(keyDir, "work")
	personalDir := filepath.Join(keyDir, "personal")
	for _, dir := range []string{keyDir, workDir, personalDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	run := func(args ...string) {
		t.Helper()
		root := cli.NewRootCmd()
		root.SetArgs(args)
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		if err := root.Execute(); err != nil {
			t.Fatalf("hasp %v: %v", args, err)
		}
	}
	run("new", "key", "id_ed25519_guard", "--no-passphrase", "--yes", "--key-dir", keyDir)

	// Replacement material for the --replace-material case below lives entirely outside HOME —
	// it's a read source (P5), not a write target, so it's deliberately not under --key-dir.
	replacementDir := t.TempDir()
	run("new", "key", "id_ed25519_replacement", "--no-passphrase", "--yes", "--key-dir", replacementDir)
	replacementPath := filepath.Join(replacementDir, "id_ed25519_replacement")

	before := snapshotTree(t, home)

	run("edit", "key", "id_ed25519_guard", "--name", "id_ed25519_guard2", "--yes", "--key-dir", keyDir)
	run("edit", "key", "id_ed25519_guard2", "--profile", "work", "--yes", "--key-dir", keyDir)
	run("edit", "key", "id_ed25519_guard2", "--add-alias", "personal/id_ed25519_guard2", "--yes", "--key-dir", keyDir)
	run("edit", "key", "id_ed25519_guard2", "--remove-alias", "personal/id_ed25519_guard2", "--yes", "--key-dir", keyDir)
	run("edit", "key", "id_ed25519_guard2", "--replace-material", replacementPath, "--yes", "--key-dir", keyDir)

	after := snapshotTree(t, home)

	sshPrefix := ".ssh" + string(filepath.Separator)
	for path := range after {
		if _, existed := before[path]; existed {
			continue
		}
		if path != ".ssh" && !strings.HasPrefix(path, sshPrefix) {
			t.Errorf("write-nothing-outside-key-dir guard: new path %q was created outside the key directory", path)
		}
	}
	for path, sum := range before {
		if path == ".ssh" || strings.HasPrefix(path, sshPrefix) {
			continue
		}
		if got := after[path]; got != sum {
			t.Errorf("write-nothing-outside-key-dir guard: %q changed outside the key directory", path)
		}
	}

	settingsPath := filepath.Join(home, ".config", "hasp", "settings.toml")
	if _, err := os.Stat(settingsPath); err == nil {
		t.Errorf("write-nothing guard: %s was created, but hasp never writes settings (D16)", settingsPath)
	}
}

// TestWriteNothingOutsideKeyDir_EditKey_RejectsProfileTraversal mirrors the adopt-side traversal
// regression test: a malicious --profile value on `edit key` must be rejected before anything is
// written, anywhere.
func TestWriteNothingOutsideKeyDir_EditKey_RejectsProfileTraversal(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	run := func(args ...string) {
		t.Helper()
		root := cli.NewRootCmd()
		root.SetArgs(args)
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		if err := root.Execute(); err != nil {
			t.Fatalf("hasp %v: %v", args, err)
		}
	}
	run("new", "key", "id_ed25519_guard", "--no-passphrase", "--yes", "--key-dir", keyDir)

	evilDir := filepath.Join(home, "tmp")
	if err := os.MkdirAll(evilDir, 0o700); err != nil {
		t.Fatal(err)
	}

	before := snapshotTree(t, home)

	cases := [][]string{
		{"edit", "key", "id_ed25519_guard", "--profile", "work/../../../tmp", "--yes", "--key-dir", keyDir},
		{"edit", "key", "id_ed25519_guard", "--add-alias", "work/../../../tmp/evil_alias", "--yes", "--key-dir", keyDir},
		{"edit", "key", "id_ed25519_guard", "--remove-alias", "work/../../../tmp/evil_alias", "--yes", "--key-dir", keyDir},
		{"edit", "key", "id_ed25519_guard", "--name", "../../tmp/evil_name", "--yes", "--key-dir", keyDir},
	}
	for _, args := range cases {
		root := cli.NewRootCmd()
		root.SetArgs(args)
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		err := root.Execute()
		if err == nil {
			t.Fatalf("hasp %v: want an error, got nil", args)
		}
		if got := cli.ExitCode(err); got != 2 {
			t.Errorf("hasp %v: ExitCode(%v) = %d, want 2 (ErrUsage)", args, err, got)
		}
	}

	after := snapshotTree(t, home)
	assertTreeUnchanged(t, before, after)

	entries, err := os.ReadDir(evilDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("write-nothing-outside-key-dir guard: %v was created outside --key-dir by a traversal argument", entries)
	}
}
