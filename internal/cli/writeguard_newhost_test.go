package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestWriteNothingOutsideKeyDir_NewHost extends tdd.md §12's write-nothing guard to `new host`
// (M3's first slice): both the default-group case (writes only ~/.ssh/config) and the custom
// --group case (writes ~/.ssh/config plus a new <group>.sshconfig) must land only inside
// --key-dir, and the settings file hasp only ever reads must never be created (D16).
func TestWriteNothingOutsideKeyDir_NewHost(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	before := snapshotTree(t, home)

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
	run("new", "host", "foobarco-prod", "--hostname", "foobarco.example.com", "--yes", "--key-dir", keyDir)
	run("new", "host", "foobarco-staging", "--group", "work", "--user", "deploy", "--yes", "--key-dir", keyDir)

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

	configBytes, err := os.ReadFile(filepath.Join(keyDir, "config"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(configBytes), "Host foobarco-prod") {
		t.Errorf("config = %q, want it to contain the default-group stanza", configBytes)
	}
	if !strings.Contains(string(configBytes), "Include "+filepath.Join(keyDir, "work.sshconfig")) {
		t.Errorf("config = %q, want it to include the new custom group", configBytes)
	}

	groupBytes, err := os.ReadFile(filepath.Join(keyDir, "work.sshconfig"))
	if err != nil {
		t.Fatalf("read work.sshconfig: %v", err)
	}
	if !strings.Contains(string(groupBytes), "Host foobarco-staging") {
		t.Errorf("work.sshconfig = %q, want it to contain the custom-group stanza", groupBytes)
	}
}

// TestNewHost_NoTTYNoYes_FailsClosed confirms `new host` refuses to apply without a TTY or --yes
// (tdd.md §11), the same gate every other write verb goes through — exercised end to end via the
// real cobra command tree, mirroring writeguard_newkey_test.go's own traversal-guard style rather
// than reaching into write.go's unexported isStdinTTY (only package cli itself can do that; see
// writeguard_alltree_test.go's own doc comment on why). Since the sandboxed test process's own
// stdin is never a TTY, no injection is needed here at all.
func TestNewHost_NoTTYNoYes_FailsClosed(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	before := snapshotTree(t, home)

	root := cli.NewRootCmd()
	root.SetArgs([]string{"new", "host", "foobarco-prod", "--key-dir", keyDir})
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	err := root.Execute()
	if err == nil {
		t.Fatal("hasp new host foobarco-prod (no TTY, no --yes): want an error, got nil")
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

// TestNewHost_ZeroArgs_UsageErrorShape is the M3 close-out review's finding 1 regression test:
// `new host`'s positional args (one-or-more Host patterns) were wired straight to
// cobra.MinimumNArgs(1) in internal/cli/new.go, unlike every other positional-arg command in this
// CLI (edit.go, profile.go, release.go, adopt.go, host.go, key.go, and new.go's own `new key`),
// which all go through minimumArgs/exactArgs (internal/cli/root.go) so a cobra arg-count error
// maps onto app.ErrUsage (exit code 2, tdd.md §10) with the "usage error: ..." message shape,
// rather than falling through to cobra's raw, unprefixed error text and the generic exit code 3.
// `edit host` (which already goes through exactArgs(1)) is exercised alongside `new host` here as
// the live comparison point: both zero-arg refusals must share the same exit code and message
// shape now that `new host` goes through minimumArgs(1) too.
func TestNewHost_ZeroArgs_UsageErrorShape(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	run := func(args ...string) error {
		root := cli.NewRootCmd()
		root.SetArgs(append(append([]string{}, args...), "--key-dir", keyDir))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		return root.Execute()
	}

	newHostErr := run("new", "host")
	if newHostErr == nil {
		t.Fatal("hasp new host (zero patterns): want an error, got nil")
	}
	if got := cli.ExitCode(newHostErr); got != 2 {
		t.Errorf("ExitCode(new host) = %d, want 2 (ErrUsage)", got)
	}
	if !strings.Contains(newHostErr.Error(), "usage error:") {
		t.Errorf("new host error = %q, want it to contain %q", newHostErr.Error(), "usage error:")
	}

	editHostErr := run("edit", "host")
	if editHostErr == nil {
		t.Fatal("hasp edit host (zero args): want an error, got nil")
	}
	if got := cli.ExitCode(editHostErr); got != 2 {
		t.Errorf("ExitCode(edit host) = %d, want 2 (ErrUsage)", got)
	}
	if !strings.Contains(editHostErr.Error(), "usage error:") {
		t.Errorf("edit host error = %q, want it to contain %q", editHostErr.Error(), "usage error:")
	}
}
