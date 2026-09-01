package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestWriteNothingOutsideKeyDir_NewKey extends tdd.md §12's write-nothing guard to the first real
// write path M2 adds: `hasp new key` must create files only inside --key-dir, and must never
// create the settings file it only ever reads (D16) — a test needing a settings fixture writes
// one as setup, never as a side effect of the code under test, so the settings path's own
// non-existence here is asserted, not merely assumed.
func TestWriteNothingOutsideKeyDir_NewKey(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	// Pin XDG_CONFIG_HOME under the sandboxed HOME too, so settings.Path() resolves inside this
	// test's own tree on every platform, regardless of what the ambient test environment has set.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	before := snapshotTree(t, home)

	root := cli.NewRootCmd()
	root.SetArgs([]string{"new", "key", "id_ed25519_guard", "--no-passphrase", "--yes", "--key-dir", keyDir})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	if err := root.Execute(); err != nil {
		t.Fatalf("hasp new key: %v", err)
	}

	after := snapshotTree(t, home)

	sshPrefix := ".ssh" + string(filepath.Separator)
	for path := range after {
		if _, existed := before[path]; existed {
			continue // pre-existing entries are covered by the ordinary content-diff below
		}
		if path != ".ssh" && !strings.HasPrefix(path, sshPrefix) {
			t.Errorf("write-nothing-outside-key-dir guard: new path %q was created outside the key directory", path)
		}
	}

	// Every pre-existing entry outside .ssh (there are none here beyond HOME itself, but the
	// assertion is general) must also be byte-identical — the same discipline
	// assertTreeUnchanged applies, scoped to entries new key had no business touching.
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
		t.Errorf("write-nothing guard: %s was created by new key, but hasp never writes settings (D16)", settingsPath)
	}
}

// TestWriteNothingOutsideKeyDir_NewKey_RejectsTraversal is the reviewer's CRITICAL-finding
// regression test, run end to end through the real CLI rather than internal/app directly: `hasp
// new key ../../../tmp/evil_key` must be rejected before anything is written, anywhere — not just
// a well-formed name confined to --key-dir, which is all the sibling test above exercises. Asserts
// both the write-nothing-outside-key-dir invariant (T8, tdd.md §12) and the exit code (ErrUsage ->
// 2, tdd.md §10: a malformed name is a usage mistake).
func TestWriteNothingOutsideKeyDir_NewKey_RejectsTraversal(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	// The traversal target: a sibling of home, well outside keyDir, that must not exist afterward.
	evilDir := filepath.Join(home, "tmp")
	if err := os.MkdirAll(evilDir, 0o700); err != nil {
		t.Fatal(err)
	}

	before := snapshotTree(t, home)

	root := cli.NewRootCmd()
	root.SetArgs([]string{"new", "key", "../tmp/evil_key", "--no-passphrase", "--yes", "--key-dir", keyDir})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	err := root.Execute()
	if err == nil {
		t.Fatal("hasp new key ../tmp/evil_key: want an error, got nil")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Errorf("ExitCode(%v) = %d, want 2 (ErrUsage)", err, got)
	}

	after := snapshotTree(t, home)
	assertTreeUnchanged(t, before, after)

	for _, name := range []string{"evil_key", "evil_key.pub"} {
		if _, statErr := os.Stat(filepath.Join(evilDir, name)); statErr == nil {
			t.Errorf("write-nothing-outside-key-dir guard: %s was created outside --key-dir by a traversal name", filepath.Join(evilDir, name))
		}
	}
}
