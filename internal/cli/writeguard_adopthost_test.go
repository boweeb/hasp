package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestWriteNothingOutsideKeyDir_AdoptHost extends tdd.md §12's write-nothing guard to `adopt
// host` (M3's second slice): adopting a hand-written stanza in ~/.ssh/config must write only
// inside --key-dir, and the settings file hasp only ever reads must never be created (D16).
// The unmanaged stanza is written directly to disk (not through hasp) as fixture setup, mirroring
// how mkGuardMarkedDir bypasses the CLI in writeguard_alltree_test.go — `adopt host`'s whole point
// is to operate on a stanza hasp did not itself author.
func TestWriteNothingOutsideKeyDir_AdoptHost(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	configPath := filepath.Join(keyDir, "config")
	if err := os.WriteFile(configPath, []byte("Host foobarco-prod\n    HostName foobarco.example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	before := snapshotTree(t, home)

	root := cli.NewRootCmd()
	root.SetArgs([]string{"adopt", "host", "foobarco-prod", "--yes", "--key-dir", keyDir})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	if err := root.Execute(); err != nil {
		t.Fatalf("hasp adopt host foobarco-prod: %v", err)
	}

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

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(configBytes), "hasp:managed") {
		t.Errorf("config = %q, want the adopted stanza wrapped in hasp's marked region", configBytes)
	}
}

// TestAdoptHost_NoTTYNoYes_FailsClosed confirms `adopt host` refuses to apply without a TTY or
// --yes (tdd.md §11), the same gate every other write verb goes through.
func TestAdoptHost_NoTTYNoYes_FailsClosed(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	configPath := filepath.Join(keyDir, "config")
	if err := os.WriteFile(configPath, []byte("Host foobarco-prod\n    HostName foobarco.example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	before := snapshotTree(t, home)

	root := cli.NewRootCmd()
	root.SetArgs([]string{"adopt", "host", "foobarco-prod", "--key-dir", keyDir})
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	err := root.Execute()
	if err == nil {
		t.Fatal("hasp adopt host foobarco-prod (no TTY, no --yes): want an error, got nil")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Errorf("ExitCode(%v) = %d, want 2 (ErrUsage)", err, got)
	}
	if out.Len() == 0 {
		t.Error("no preview was rendered before the no-TTY refusal")
	}

	after := snapshotTree(t, home)
	assertTreeUnchanged(t, before, after)
}
