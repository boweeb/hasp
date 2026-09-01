package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestWriteNothingOutsideKeyDir_EditHost extends tdd.md §12's write-nothing guard to `edit host`
// (M3's third slice): changing a directive, rebinding, and moving a managed stanza between host
// groups must all write only inside --key-dir, and the settings file hasp only ever reads must
// never be created (D16).
func TestWriteNothingOutsideKeyDir_EditHost(t *testing.T) {
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
		root.SetArgs(append(append([]string{}, args...), "--yes", "--key-dir", keyDir))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		if err := root.Execute(); err != nil {
			t.Fatalf("hasp %v: %v", args, err)
		}
	}
	run("new", "host", "foobarco-prod", "--hostname", "foobarco.example.com")

	before := snapshotTree(t, home)

	root := cli.NewRootCmd()
	root.SetArgs([]string{
		"edit", "host", "foobarco-prod",
		"--hostname", "foobarco2.example.com",
		"--group", "work",
		"--yes", "--key-dir", keyDir,
	})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	if err := root.Execute(); err != nil {
		t.Fatalf("hasp edit host foobarco-prod: %v", err)
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

	groupFile := filepath.Join(keyDir, "work.sshconfig")
	groupBytes, err := os.ReadFile(groupFile)
	if err != nil {
		t.Fatalf("read %s: %v", groupFile, err)
	}
	if !strings.Contains(string(groupBytes), "foobarco2.example.com") {
		t.Errorf("group file = %q, want the moved, edited stanza", groupBytes)
	}
}

// TestEditHost_NoTTYNoYes_FailsClosed confirms `edit host` refuses to apply without a TTY or
// --yes (tdd.md §11), the same gate every other write verb goes through.
func TestEditHost_NoTTYNoYes_FailsClosed(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	newRoot := cli.NewRootCmd()
	newRoot.SetArgs([]string{"new", "host", "foobarco-prod", "--hostname", "foobarco.example.com", "--yes", "--key-dir", keyDir})
	newRoot.SetOut(&bytes.Buffer{})
	newRoot.SetErr(&bytes.Buffer{})
	if err := newRoot.Execute(); err != nil {
		t.Fatalf("guard setup (new host): %v", err)
	}

	before := snapshotTree(t, home)

	root := cli.NewRootCmd()
	root.SetArgs([]string{"edit", "host", "foobarco-prod", "--hostname", "foobarco2.example.com", "--key-dir", keyDir})
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	err := root.Execute()
	if err == nil {
		t.Fatal("hasp edit host foobarco-prod (no TTY, no --yes): want an error, got nil")
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
