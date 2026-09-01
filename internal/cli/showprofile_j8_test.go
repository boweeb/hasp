package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestShowProfile_OffboardingAnswersKeysAndHosts is J8's actual test (roadmap.md §5 exit
// criterion 5, T21): "if I show this profile, do I see everything that would break if it went
// away." ShowProfile's host-aggregation (profile.go) has no coverage anywhere in the repo that
// drives a host through it at all — every existing show-profile test (profile_usecase_test.go,
// profile-show-recurse.txtar) exercises keys only. This closes that gap with a realistic
// offboarding fixture: a managed profile ("work") that owns a key by location (D13), and a host
// bound to that key living in a custom host-group file of the kind `new host --group=<name>`
// creates (D9/T11) — not a synthetic domain.Host value. Driven through the real `hasp` CLI tree
// end to end: `new host --group` writes the fixture, `show profile` reads it back.
func TestShowProfile_OffboardingAnswersKeysAndHosts(t *testing.T) {
	home := t.TempDir()
	keyDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	// The "work" profile owns a real key by location (D13): a managed directory (.hasp marker)
	// holding the key file directly.
	fixturesDir := repoFixturesDir(t)
	copyFixtureInto(t, fixturesDir, "ed25519-openssh-plain-pub", filepath.Join(keyDir, "work", "id_ed25519_work"))
	copyFixtureInto(t, fixturesDir, "ed25519-openssh-plain-pub.pub", filepath.Join(keyDir, "work", "id_ed25519_work.pub"))
	writeFile(t, filepath.Join(keyDir, "work", ".hasp"), "# profile marker\n")

	run := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		root := cli.NewRootCmd()
		root.SetArgs(append(args, "--key-dir", keyDir))
		root.SetOut(&out)
		root.SetErr(&out)
		if err := root.Execute(); err != nil {
			t.Fatalf("hasp %v: %v\n%s", args, err, out.String())
		}
		return out.String()
	}

	// `new host --group=work` is the exact mechanism `new host --group=<name>` uses to create a
	// wholly hasp-owned custom host-group file (D7/D9/T11) — not a hand-built domain.Host.
	run("new", "host", "foobarco-prod",
		"--group", "work",
		"--key", "id_ed25519_work",
		"--hostname", "prod.foobarco.internal",
		"--yes",
	)

	groupFile := filepath.Join(keyDir, "work.sshconfig")
	if _, err := os.Stat(groupFile); err != nil {
		t.Fatalf("work.sshconfig was not created by new host --group=work: %v", err)
	}

	// J8's actual question: show me everything that would break if "work" went away.
	showOut := run("show", "profile", "work")
	if !strings.Contains(showOut, "id_ed25519_work") {
		t.Errorf("show profile work missing its key:\n%s", showOut)
	}
	if !strings.Contains(showOut, "foobarco-prod") {
		t.Errorf("show profile work missing its host (bound via a custom --group file):\n%s", showOut)
	}
	// Attribution (T21): both rows must be attributed to the "work" profile they actually came
	// from, not merely present somewhere in the output.
	for _, line := range strings.Split(showOut, "\n") {
		if strings.Contains(line, "id_ed25519_work") || strings.Contains(line, "foobarco-prod") {
			if !strings.Contains(line, "(from work)") {
				t.Errorf("row not attributed to work: %q", line)
			}
		}
	}

	// Same assertion through --json, where a machine consumer reads fromProfile directly (T21).
	jsonOut := run("show", "profile", "work", "--json")
	if !strings.Contains(jsonOut, `"name": "id_ed25519_work"`) && !strings.Contains(jsonOut, `"id_ed25519_work"`) {
		t.Errorf("--json output missing the key:\n%s", jsonOut)
	}
	if !strings.Contains(jsonOut, "foobarco-prod") {
		t.Errorf("--json output missing the host:\n%s", jsonOut)
	}
	if strings.Count(jsonOut, `"fromProfile": "work"`) < 2 {
		t.Errorf("--json output must attribute both the key and the host to \"work\":\n%s", jsonOut)
	}
}
