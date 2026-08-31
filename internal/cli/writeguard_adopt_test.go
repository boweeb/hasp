package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestWriteNothingOutsideKeyDir_AdoptRelease extends tdd.md §12's write-nothing guard to `adopt
// key` and `release key`: every write either command makes must land inside --key-dir, and the
// settings file hasp only ever reads must never be created (D16).
func TestWriteNothingOutsideKeyDir_AdoptRelease(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	profileDir := filepath.Join(keyDir, "work")
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, ".hasp"), []byte("# profile marker\n"), 0o644); err != nil {
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

	before := snapshotTree(t, home)

	run("adopt", "key", "id_ed25519_guard", "--profile", "work", "--yes", "--key-dir", keyDir)
	run("release", "key", "id_ed25519_guard", "--yes", "--key-dir", keyDir)

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

// TestWriteNothingOutsideKeyDir_AdoptKey_RejectsProfileTraversal is the adopt-side regression
// test mirroring writeguard_newkey_test.go's own CRITICAL-finding guard: a malicious --profile
// value must be rejected before anything is written, anywhere.
func TestWriteNothingOutsideKeyDir_AdoptKey_RejectsProfileTraversal(t *testing.T) {
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

	root := cli.NewRootCmd()
	root.SetArgs([]string{"adopt", "key", "id_ed25519_guard", "--profile", "work/../../../tmp", "--yes", "--key-dir", keyDir})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	err := root.Execute()
	if err == nil {
		t.Fatal("hasp adopt key --profile work/../../../tmp: want an error, got nil")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Errorf("ExitCode(%v) = %d, want 2 (ErrUsage)", err, got)
	}

	after := snapshotTree(t, home)
	assertTreeUnchanged(t, before, after)

	for _, name := range []string{"id_ed25519_guard", "id_ed25519_guard.pub"} {
		if _, statErr := os.Stat(filepath.Join(evilDir, name)); statErr == nil {
			t.Errorf("write-nothing-outside-key-dir guard: %s was created outside --key-dir by a traversal profile", filepath.Join(evilDir, name))
		}
	}
}
