package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/app"
)

// snapshotTreeForGuard is a minimal duplicate of writeguard_test.go's own snapshotTree
// (internal/cli's cli_test external test package), reimplemented here because this file lives in
// package cli itself — the only place isStdinTTY (write.go's unexported injectable var) can be
// forced without a real pty — and a package-cli file cannot see symbols defined in a package
// cli_test file even though both live under the same directory (Go's two-test-package
// convention). Duplicating ~20 lines of digesting logic is cheaper and more honest than exporting
// test-only plumbing across that boundary.
func snapshotTreeForGuard(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			target, linkErr := os.Readlink(path)
			if linkErr != nil {
				return linkErr
			}
			snap[rel] = "symlink:" + target
			return nil
		}
		if d.IsDir() {
			snap[rel] = "dir"
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(b)
		snap[rel] = "file:" + hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotTreeForGuard(%s): %v", root, err)
	}
	return snap
}

func assertTreeUnchangedForGuard(t *testing.T, before, after map[string]string) {
	t.Helper()
	for path, sum := range before {
		got, ok := after[path]
		if !ok {
			t.Errorf("write-nothing guard: %s was removed", path)
			continue
		}
		if got != sum {
			t.Errorf("write-nothing guard: %s changed", path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			t.Errorf("write-nothing guard: %s was created", path)
		}
	}
}

// guardEnv is one scratch HOME + --key-dir, sandboxed for a single subtest of
// TestWriteNothingGuard_EveryWriteCommand_FailsClosedWithNoTTYNoYes.
type guardEnv struct {
	home   string
	keyDir string
}

func newGuardEnv(t *testing.T) guardEnv {
	t.Helper()
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	return guardEnv{home: home, keyDir: keyDir}
}

// runGuardSetup executes args (with --key-dir appended) and fails the test if it errors — used
// only to build the prerequisite managed/unmanaged state a given write command's no-TTY refusal
// needs to reach runWritePlan in the first place (an unmet precondition would fail earlier, for an
// unrelated reason, and would not actually exercise the TTY gate this test targets).
func (g guardEnv) runGuardSetup(t *testing.T, args ...string) {
	t.Helper()
	root := NewRootCmd()
	root.SetArgs(append(append([]string{}, args...), "--key-dir", g.keyDir, "--yes"))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	if err := root.Execute(); err != nil {
		t.Fatalf("guard setup %v: %v", args, err)
	}
}

// mkGuardMarkedDir creates dir (relative to g.keyDir) with a .hasp marker, bypassing the CLI —
// this is fixture setup, not part of what the test under it exercises.
func (g guardEnv) mkGuardMarkedDir(t *testing.T, rel string) string {
	t.Helper()
	dir := filepath.Join(g.keyDir, rel)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hasp"), []byte("# managed by hasp\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestWriteNothingGuard_EveryWriteCommand_FailsClosedWithNoTTYNoYes is roadmap.md §4 exit
// criterion 3's milestone-level proof: rather than trusting that every write verb's own phase
// remembered to test "no TTY, no --yes fails closed" once (write_test.go's runWritePlan tests do
// this generically; each of new-key.txtar/adopt-release-key.txtar/adopt-release-profile.txtar/
// edit-key.txtar asserts it for its own verb end to end through a real subprocess), this walks
// every write subcommand currently registered on the CLI tree — new key, adopt key, release key,
// adopt profile, release profile, and every one of edit key's five independent flags — in one
// table, through the real cobra command tree (NewRootCmd, not a hand-built Plan), forcing
// isStdinTTY false and omitting --yes. A future write verb added to root.go without its own
// no-TTY test still gets caught here only if this table is extended to include it — see the
// "Judgment call" note on TestChangeKinds_RequiresBackup_GoldenList (change_requiresbackup_golden_
// test.go) for why Go gives no mechanical way to enumerate "every subcommand on a *cobra.Command
// tree" against a golden count the way it does for a struct's own fields.
//
// Each case also asserts runWritePlan's preview rendered to stdout before the gate refused — the
// literal reading of write.go's own doc comment ("this render call IS the preview... it happens
// whether or not the write proceeds") — across every verb here, not merely the "at least 2" spot
// check the close-out instructions asked for as a minimum.
//
// The write-nothing assertion covers the whole sandboxed $HOME, not just --key-dir, and includes
// the settings-file-never-created check (D16, exit criterion 6) — matching every sibling
// write-guard test's convention, rather than a narrower scope this file's own name and claim of
// milestone-level coverage would otherwise misrepresent.
func TestWriteNothingGuard_EveryWriteCommand_FailsClosedWithNoTTYNoYes(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, g guardEnv) []string // returns the command args, --key-dir excluded
	}{
		{
			name: "new key",
			setup: func(t *testing.T, g guardEnv) []string {
				return []string{"new", "key", "case_new", "--no-passphrase"}
			},
		},
		{
			name: "adopt key",
			setup: func(t *testing.T, g guardEnv) []string {
				g.mkGuardMarkedDir(t, "work")
				g.runGuardSetup(t, "new", "key", "case_adopt", "--no-passphrase")
				return []string{"adopt", "key", "case_adopt", "--profile", "work"}
			},
		},
		{
			name: "release key",
			setup: func(t *testing.T, g guardEnv) []string {
				g.mkGuardMarkedDir(t, "work")
				g.runGuardSetup(t, "new", "key", "case_release", "--no-passphrase")
				g.runGuardSetup(t, "adopt", "key", "case_release", "--profile", "work")
				return []string{"release", "key", "case_release"}
			},
		},
		{
			name: "new host",
			setup: func(t *testing.T, g guardEnv) []string {
				return []string{"new", "host", "case_newhost", "--hostname", "example.com"}
			},
		},
		{
			name: "adopt profile",
			setup: func(t *testing.T, g guardEnv) []string {
				if err := os.MkdirAll(filepath.Join(g.keyDir, "newprofile"), 0o700); err != nil {
					t.Fatal(err)
				}
				return []string{"adopt", "profile", "newprofile"}
			},
		},
		{
			name: "release profile",
			setup: func(t *testing.T, g guardEnv) []string {
				g.mkGuardMarkedDir(t, "relprofile")
				return []string{"release", "profile", "relprofile"}
			},
		},
		{
			name: "edit key --name",
			setup: func(t *testing.T, g guardEnv) []string {
				g.runGuardSetup(t, "new", "key", "case_editname", "--no-passphrase")
				return []string{"edit", "key", "case_editname", "--name", "case_editname2"}
			},
		},
		{
			name: "edit key --profile",
			setup: func(t *testing.T, g guardEnv) []string {
				g.mkGuardMarkedDir(t, "work")
				g.runGuardSetup(t, "new", "key", "case_editprofile", "--no-passphrase")
				return []string{"edit", "key", "case_editprofile", "--profile", "work"}
			},
		},
		{
			name: "edit key --add-alias",
			setup: func(t *testing.T, g guardEnv) []string {
				g.mkGuardMarkedDir(t, "personal")
				g.runGuardSetup(t, "new", "key", "case_editalias", "--no-passphrase")
				return []string{"edit", "key", "case_editalias", "--add-alias", "personal/case_editalias"}
			},
		},
		{
			name: "edit key --remove-alias",
			setup: func(t *testing.T, g guardEnv) []string {
				g.mkGuardMarkedDir(t, "personal")
				g.runGuardSetup(t, "new", "key", "case_editremovealias", "--no-passphrase")
				g.runGuardSetup(t, "edit", "key", "case_editremovealias", "--add-alias", "personal/case_editremovealias")
				return []string{"edit", "key", "case_editremovealias", "--remove-alias", "personal/case_editremovealias"}
			},
		},
		{
			name: "edit key --replace-material",
			setup: func(t *testing.T, g guardEnv) []string {
				g.runGuardSetup(t, "new", "key", "case_editreplace", "--no-passphrase")

				replacementDir := t.TempDir()
				root := NewRootCmd()
				root.SetArgs([]string{"new", "key", "case_editdonor", "--no-passphrase", "--yes", "--key-dir", replacementDir})
				root.SetOut(&bytes.Buffer{})
				root.SetErr(&bytes.Buffer{})
				if err := root.Execute(); err != nil {
					t.Fatalf("guard setup (replacement donor): %v", err)
				}

				return []string{"edit", "key", "case_editreplace", "--replace-material", filepath.Join(replacementDir, "case_editdonor")}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := newGuardEnv(t)
			args := tc.setup(t, g)

			// Snapshot the whole sandboxed $HOME, not just --key-dir — matching every sibling
			// write-guard test's convention (writeguard_adopt_test.go, writeguard_edit_test.go,
			// writeguard_newkey_test.go all call snapshotTree(t, home)) and tdd.md §12's own
			// guard-test discipline: the invariant under test is "hasp writes nothing outside the
			// key directory," which a key-dir-only snapshot can't actually prove — it can only
			// prove "nothing changed inside the one place we already trust." A prior version of
			// this test snapshotted only g.keyDir, which a review caught as narrower than both its
			// own doc comment's claim and the sibling tests it was meant to generalize.
			before := snapshotTreeForGuard(t, g.home)

			orig := isStdinTTY
			isStdinTTY = func() bool { return false }
			t.Cleanup(func() { isStdinTTY = orig })

			root := NewRootCmd()
			root.SetArgs(append(append([]string{}, args...), "--key-dir", g.keyDir))
			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			root.SetOut(out)
			root.SetErr(errOut)

			err := root.Execute()
			if err == nil {
				t.Fatalf("hasp %v (no TTY, no --yes): want an error, got nil", args)
			}
			if !errors.Is(err, app.ErrUsage) {
				t.Errorf("hasp %v error = %v, want app.ErrUsage (tdd.md §11: no TTY, no --yes -> fail closed)", args, err)
			}
			if got := ExitCode(err); got != 2 {
				t.Errorf("hasp %v: ExitCode(%v) = %d, want 2", args, err, got)
			}
			if !strings.Contains(err.Error(), "stdin is not a terminal") {
				t.Errorf("hasp %v error = %q, want it to name the TTY gate (confirmApply), not some earlier validation failure", args, err)
			}

			// The preview render call happens whether or not the write proceeds (write.go's own doc
			// comment on runWritePlan, step 1) — confirm it actually reached stdout even though the
			// write itself was refused.
			if out.Len() == 0 {
				t.Errorf("hasp %v: no preview was rendered before the no-TTY refusal", args)
			}

			after := snapshotTreeForGuard(t, g.home)
			assertTreeUnchangedForGuard(t, before, after)

			// Exit criterion 6 (roadmap.md §4): hasp reads settings and never writes them. A
			// refused write must not be the exception — check it explicitly, matching every
			// sibling write-guard test's own settingsPath assertion.
			settingsPath := filepath.Join(g.home, ".config", "hasp", "settings.toml")
			if _, err := os.Stat(settingsPath); err == nil {
				t.Errorf("hasp %v: %s was created, but hasp never writes settings (D16)", args, settingsPath)
			}
		})
	}
}
