package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/cli"
)

// TestFullInventory exercises roadmap.md §3's exit criteria together against a synthetic
// ~/.ssh matching design.md §2's own motivating inventory: mixed algorithms (RSA/ECDSA/
// Ed25519/DSA), mixed formats (OpenSSH/PEM), a symlink alias, and an encrypted legacy-PEM key
// with no public half (the derivation gap, T1/D12). Built from the real committed fixtures
// (testdata/keys/) rather than hand-typed crypto material — a hand-fabricated OpenSSH/PEM
// blob is not guaranteed to parse, and a first attempt at this exact test caught exactly that
// mistake (two keys silently failed open and vanished from the inventory).
func TestFullInventory(t *testing.T) {
	fixturesDir := repoFixturesDir(t)
	dir := t.TempDir()

	copyFixtureInto(t, fixturesDir, "ed25519-openssh-plain-pub", filepath.Join(dir, "id_ed25519"))
	copyFixtureInto(t, fixturesDir, "ed25519-openssh-plain-pub.pub", filepath.Join(dir, "id_ed25519.pub"))
	copyFixtureInto(t, fixturesDir, "rsa-openssh-plain-pub", filepath.Join(dir, "id_rsa"))
	copyFixtureInto(t, fixturesDir, "rsa-openssh-plain-pub.pub", filepath.Join(dir, "id_rsa.pub"))
	copyFixtureInto(t, fixturesDir, "ecdsa-openssh-plain-pub", filepath.Join(dir, "id_ecdsa"))
	copyFixtureInto(t, fixturesDir, "ecdsa-openssh-plain-pub.pub", filepath.Join(dir, "id_ecdsa.pub"))
	copyFixtureInto(t, fixturesDir, "rsa-pem-plain-pub", filepath.Join(dir, "id_rsa_legacy"))
	copyFixtureInto(t, fixturesDir, "rsa-pem-plain-pub.pub", filepath.Join(dir, "id_rsa_legacy.pub"))
	copyFixtureInto(t, fixturesDir, "dsa-pem-plain-pub", filepath.Join(dir, "id_dsa_ancient"))
	copyFixtureInto(t, fixturesDir, "dsa-pem-plain-pub.pub", filepath.Join(dir, "id_dsa_ancient.pub"))
	// The derivation gap (T1, D12): legacy PEM, encrypted, no public half — fingerprint must
	// report as unknown, never guessed.
	copyFixtureInto(t, fixturesDir, "rsa-pem-encrypted-nopub", filepath.Join(dir, "id_rsa_old"))

	foobarcoReal := filepath.Join(dir, "work", "foobarco", "id_ed25519_foobarco")
	copyFixtureInto(t, fixturesDir, "ed25519-openssh-plain-nopub", foobarcoReal)
	writeFile(t, filepath.Join(dir, "work", "foobarco", ".hasp"), "# profile marker\n")
	if err := os.Symlink(foobarcoReal, filepath.Join(dir, "id_ed25519_foobarco")); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(dir, "config"), ""+
		"Host foobarco-prod\n"+
		"    HostName prod.foobarco.internal\n"+
		"    IdentityFile "+filepath.Join(dir, "id_ed25519_foobarco")+"\n"+
		"\n"+
		"Host legacy\n"+
		"    HostName legacy.example.com\n", // no IdentityFile: implicit-default probing (T16)
	)

	before := snapshotTree(t, dir)

	run := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		root := cli.NewRootCmd()
		root.SetArgs(append(args, "--key-dir", dir))
		root.SetOut(&out)
		root.SetErr(&out)
		_ = root.Execute() // check-shaped exit codes are asserted separately where relevant
		return out.String()
	}

	// Exit criterion 1: hasp list key returns an accurate inventory, one screen (P7).
	listOut := run("list", "key")
	for _, want := range []string{"id_ed25519", "id_rsa", "id_ecdsa", "id_rsa_legacy", "id_dsa_ancient", "id_ed25519_foobarco", "unknown"} {
		if !strings.Contains(listOut, want) {
			t.Errorf("list key output missing %q:\n%s", want, listOut)
		}
	}
	for _, line := range strings.Split(strings.TrimRight(listOut, "\n"), "\n") {
		if len(line) > 120 {
			t.Errorf("list key row exceeds 120 columns (%d): %q", len(line), line)
		}
	}

	// Exit criterion 2: hasp find key identifies a key from a fingerprint fragment regardless
	// of punctuation and case (J2).
	findOut := run("find", "key", "T4RN-6go9")
	if !strings.Contains(findOut, "id_ed25519_foobarco") {
		t.Errorf("find key t4rn6go9 (mangled) did not match id_ed25519_foobarco:\n%s", findOut)
	}

	// Exit criterion 3: hasp show key <name> lists every host that binds it, explicit and
	// implicit-default bindings labelled distinctly (T16).
	showFoobarCo := run("show", "key", "id_ed25519_foobarco")
	if !strings.Contains(showFoobarCo, "foobarco-prod") || !strings.Contains(showFoobarCo, "(explicit)") {
		t.Errorf("show key id_ed25519_foobarco missing an explicit foobarco-prod binding:\n%s", showFoobarCo)
	}
	showRSA := run("show", "key", "id_rsa")
	if !strings.Contains(showRSA, "legacy") || !strings.Contains(showRSA, "(implicit-default)") {
		t.Errorf("show key id_rsa missing an implicit-default legacy binding:\n%s", showRSA)
	}

	// Exit criterion 7: every read has --json, and it carries version 1.
	for _, args := range [][]string{
		{"list", "key", "--json"}, {"show", "key", "id_ed25519", "--json"}, {"find", "key", "id_ed25519", "--json"},
		{"list", "host", "--json"}, {"show", "host", "legacy", "--json"}, {"find", "host", "legacy", "--json"},
		{"list", "profile", "--json"}, {"find", "profile", "foobarco", "--json"},
	} {
		out := run(args...)
		if !strings.Contains(out, `"version": 1`) {
			t.Errorf("%v: --json output missing version 1:\n%s", args, out)
		}
	}

	// Exit criterion 4: the write-nothing guard passes across the entire read suite — a
	// pre/post snapshot of the whole key-dir tree is byte-identical (§11, D14: M1 never writes
	// to ~/.ssh at all).
	for _, args := range [][]string{
		{"list", "host"}, {"show", "host", "legacy"}, {"find", "host", "foobarco-prod"},
		{"list", "profile"}, {"show", "profile", "work.foobarco"}, {"find", "profile", "foobarco"},
		{"check"}, {"check", "key"}, {"check", "host"}, {"check", "profile"},
	} {
		run(args...)
	}
	after := snapshotTree(t, dir)
	assertTreeUnchanged(t, before, after)
}

func repoFixturesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// this file lives at internal/cli/fullinventory_test.go
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "keys")
}

func copyFixtureInto(t *testing.T, fixturesDir, name, destPath string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixturesDir, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destPath, b, 0o600); err != nil {
		t.Fatal(err)
	}
}
