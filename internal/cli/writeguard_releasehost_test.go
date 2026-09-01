package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestWriteNothingOutsideKeyDir_ReleaseHost extends tdd.md §12's write-nothing guard to `release
// host` (M3's second slice, D14/D18): releasing a managed stanza back out of ~/.ssh/config's
// marked region must write only inside --key-dir, and the settings file hasp only ever reads must
// never be created (D16).
func TestWriteNothingOutsideKeyDir_ReleaseHost(t *testing.T) {
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

	run := func(args ...string) {
		t.Helper()
		root := cli.NewRootCmd()
		root.SetArgs(append(args, "--yes", "--key-dir", keyDir))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		if err := root.Execute(); err != nil {
			t.Fatalf("hasp %v: %v", args, err)
		}
	}
	run("adopt", "host", "foobarco-prod")

	before := snapshotTree(t, home)

	root := cli.NewRootCmd()
	root.SetArgs([]string{"release", "host", "foobarco-prod", "--yes", "--key-dir", keyDir})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	if err := root.Execute(); err != nil {
		t.Fatalf("hasp release host foobarco-prod: %v", err)
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
	if !strings.Contains(string(configBytes), "Host foobarco-prod") {
		t.Errorf("config = %q, want the released stanza still present as plain text", configBytes)
	}
}

// TestReleaseHost_NoTTYNoYes_FailsClosed confirms `release host` refuses to apply without a TTY
// or --yes (tdd.md §11), the same gate every other write verb goes through.
func TestReleaseHost_NoTTYNoYes_FailsClosed(t *testing.T) {
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

	adoptRoot := cli.NewRootCmd()
	adoptRoot.SetArgs([]string{"adopt", "host", "foobarco-prod", "--yes", "--key-dir", keyDir})
	adoptRoot.SetOut(&bytes.Buffer{})
	adoptRoot.SetErr(&bytes.Buffer{})
	if err := adoptRoot.Execute(); err != nil {
		t.Fatalf("guard setup (adopt host): %v", err)
	}

	before := snapshotTree(t, home)

	root := cli.NewRootCmd()
	root.SetArgs([]string{"release", "host", "foobarco-prod", "--key-dir", keyDir})
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	err := root.Execute()
	if err == nil {
		t.Fatal("hasp release host foobarco-prod (no TTY, no --yes): want an error, got nil")
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
