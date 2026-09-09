package app

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh/agent"

	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/adapter/sshagent"
	"github.com/boweeb/hasp/internal/domain"
)

// testPassphrase mirrors tools/genfixtures/main.go's own constant of the same name and value —
// a fixed, publicly-known passphrase used only to make an "encrypted" fixture parseable by
// tests, never a secret. Duplicated per-package, as internal/adapter/keyfile's and
// internal/adapter/sshagent's own test suites already do, because tools/genfixtures is a `main`
// package and cannot be imported.
const testPassphrase = "hasp-test-fixture-passphrase"

func openKnownMaterial(t *testing.T, fixtureName string) (path string, known keyfile.Material) {
	t.Helper()
	path = filepath.Join(testdataKeysDir(t), fixtureName)
	mat, err := keyfile.OpenMaterial(path, nil)
	if err != nil {
		t.Fatalf("OpenMaterial(%s, nil): %v", fixtureName, err)
	}
	return path, mat
}

// TestPassphraseGate_NoTTY_Degrades is roadmap.md §5.6 exit criterion 5's own case and T39's
// declared stance (tdd.md §11): with no callback at all — internal/cli's stand-in for "no TTY
// attached" — a prompt-worthy key degrades to unknown with the exact reason token
// domain.ReasonPassphraseRequiredNoTTY, never an error.
func TestPassphraseGate_NoTTY_Degrades(t *testing.T) {
	path, known := openKnownMaterial(t, "rsa-pem-encrypted-nopub") // legacy PEM, no .pub: ambiguous, always worth prompting

	gate := &PassphraseGate{Passphrase: nil}
	got := gate.Derive(path, known)

	if got.Unlocked {
		t.Fatal("Unlocked = true with no passphrase source at all")
	}
	if got.Reason != domain.ReasonPassphraseRequiredNoTTY {
		t.Errorf("Reason = %q, want %q", got.Reason, domain.ReasonPassphraseRequiredNoTTY)
	}
}

// TestPassphraseGate_NotPromptWorthy_NeverPrompts proves T39 rule 2 directly: a key that is
// already known not to be RSA (ed25519, OpenSSH format — the algorithm is derivable from its
// own embedded public half with no passphrase at all, fpscheme.CreatedRSAPromptWorthy) must
// never trigger the callback, even though one is available and would happily answer.
func TestPassphraseGate_NotPromptWorthy_NeverPrompts(t *testing.T) {
	path, known := openKnownMaterial(t, "ed25519-openssh-encrypted-nopub")

	calls := 0
	gate := &PassphraseGate{Passphrase: func() ([]byte, error) {
		calls++
		return []byte(testPassphrase), nil
	}}
	got := gate.Derive(path, known)

	if calls != 0 {
		t.Errorf("Passphrase callback invoked %d times, want 0 (known-non-RSA key must never prompt)", calls)
	}
	if got.Unlocked {
		t.Error("Unlocked = true for a key this gate never attempted to decrypt")
	}
	if got.Reason != domain.ReasonSchemeNotApplicable {
		t.Errorf("Reason = %q, want %q", got.Reason, domain.ReasonSchemeNotApplicable)
	}
}

// TestPassphraseGate_OnePromptPerInvocation is roadmap.md §5.6 exit criterion 5 / T39 rule 3:
// the same PassphraseGate value, asked to Derive material for two separate RSA-candidate keys,
// must invoke the callback exactly once and reuse the same passphrase for the second key.
func TestPassphraseGate_OnePromptPerInvocation(t *testing.T) {
	path1, known1 := openKnownMaterial(t, "rsa-pem-encrypted-nopub")
	path2, known2 := openKnownMaterial(t, "rsa-openssh-encrypted-nopub")

	calls := 0
	gate := &PassphraseGate{Passphrase: func() ([]byte, error) {
		calls++
		return []byte(testPassphrase), nil
	}}

	got1 := gate.Derive(path1, known1)
	got2 := gate.Derive(path2, known2)
	gate.Close()

	if calls != 1 {
		t.Fatalf("Passphrase callback invoked %d times across two candidate keys, want exactly 1", calls)
	}
	if !got1.Unlocked || got1.Material.Private == nil {
		t.Errorf("key 1: Unlocked = %v, Private = %v; want unlocked with the correct passphrase", got1.Unlocked, got1.Material.Private)
	}
	if !got2.Unlocked || got2.Material.Private == nil {
		t.Errorf("key 2: Unlocked = %v, Private = %v; want unlocked with the correct passphrase", got2.Unlocked, got2.Material.Private)
	}
}

// TestPassphraseGate_WrongPassphrase_Degrades proves a rejected passphrase degrades exactly like
// "nobody was there to answer" — an honest unknown, never an error — matching
// keyfile.OpenMaterial's own documented contract for a wrong passphrase.
func TestPassphraseGate_WrongPassphrase_Degrades(t *testing.T) {
	path, known := openKnownMaterial(t, "rsa-pem-encrypted-nopub")

	gate := &PassphraseGate{Passphrase: func() ([]byte, error) {
		return []byte("definitely-the-wrong-passphrase"), nil
	}}
	got := gate.Derive(path, known)

	if got.Unlocked {
		t.Fatal("Unlocked = true with a wrong passphrase")
	}
	if got.Reason != domain.ReasonPrivateKeyUnavailable {
		t.Errorf("Reason = %q, want %q", got.Reason, domain.ReasonPrivateKeyUnavailable)
	}
}

// TestPassphraseGate_PromptError_DegradesAndMemoizes proves a callback that itself errors (e.g.
// internal/cli's promptPassphrase failing against a broken fd) degrades the same way a nil
// callback does — never surfaced as this gate's own error — and that the error is memoized too,
// so a second candidate key does not retry a callback that already failed once this invocation.
func TestPassphraseGate_PromptError_DegradesAndMemoizes(t *testing.T) {
	path1, known1 := openKnownMaterial(t, "rsa-pem-encrypted-nopub")
	path2, known2 := openKnownMaterial(t, "rsa-openssh-encrypted-nopub")

	calls := 0
	promptErr := errors.New("read passphrase: not a terminal")
	gate := &PassphraseGate{Passphrase: func() ([]byte, error) {
		calls++
		return nil, promptErr
	}}

	got1 := gate.Derive(path1, known1)
	got2 := gate.Derive(path2, known2)

	if calls != 1 {
		t.Fatalf("Passphrase callback invoked %d times, want exactly 1 (memoized even on error)", calls)
	}
	for i, got := range []DerivedMaterial{got1, got2} {
		if got.Unlocked {
			t.Errorf("key %d: Unlocked = true after a failed prompt", i+1)
		}
		if got.Reason != domain.ReasonPassphraseRequiredNoTTY {
			t.Errorf("key %d: Reason = %q, want %q", i+1, got.Reason, domain.ReasonPassphraseRequiredNoTTY)
		}
	}
}

// TestPassphraseGate_AlreadyUnlocked_NeverPrompts covers the fast path: material that already
// carries a decrypted private key (an unencrypted key, or material a caller decrypted some other
// way) needs no passphrase and must never trigger the callback.
func TestPassphraseGate_AlreadyUnlocked_NeverPrompts(t *testing.T) {
	path, known := openKnownMaterial(t, "rsa-pem-plain-pub") // unencrypted: Private already populated

	calls := 0
	gate := &PassphraseGate{Passphrase: func() ([]byte, error) {
		calls++
		return []byte(testPassphrase), nil
	}}
	got := gate.Derive(path, known)

	if calls != 0 {
		t.Errorf("Passphrase callback invoked %d times, want 0 (already-unlocked key must never prompt)", calls)
	}
	if !got.Unlocked {
		t.Error("Unlocked = false for an already-decrypted key")
	}
}

// TestPassphraseGate_AgentTriedFirst_NeverPrompts is the T38-consequence integration test roadmap.md
// §5.6 exit criterion 5 asks for: "a key present in the agent must not trigger the callback at
// all." It exercises the real internal/adapter/sshagent source end to end — a fake agent with the
// ed25519-openssh-encrypted-nopub fixture loaded and a non-empty comment (the committed fixture
// itself carries an empty one, tools/genfixtures used ssh-keygen -C "") — merges the agent's
// comment into the material a caller would pass to Derive (simulating chunk M3.6.4's own
// "agent tried first" wiring at the call site, per PassphraseGate's own doc comment), and asserts
// the callback never fires and the agent-sourced comment survives the call unchanged.
func TestPassphraseGate_AgentTriedFirst_NeverPrompts(t *testing.T) {
	path, known := openKnownMaterial(t, "ed25519-openssh-encrypted-nopub")

	priv, err := keyfile.OpenMaterial(path, []byte(testPassphrase))
	if err != nil {
		t.Fatalf("OpenMaterial with the correct passphrase (to load into the fake agent): %v", err)
	}
	if priv.Private == nil {
		t.Fatal("fixture did not decrypt with testPassphrase")
	}

	keyring := agent.NewKeyring()
	const wantComment = "hasp-test-agent-comment"
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv.Private, Comment: wantComment}); err != nil {
		t.Fatalf("keyring.Add: %v", err)
	}

	sockFile, err := os.CreateTemp("", "hasp-passphrasegate-*.sock")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	sockPath := sockFile.Name()
	if err := sockFile.Close(); err != nil {
		t.Fatalf("Close placeholder socket file: %v", err)
	}
	if err := os.Remove(sockPath); err != nil {
		t.Fatalf("Remove placeholder socket file: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(sockPath) })

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("net.Listen(unix, %s): %v", sockPath, err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		_ = agent.ServeAgent(keyring, conn)
	}()

	facts := sshagent.List(sockPath)
	if len(facts) != 1 {
		t.Fatalf("sshagent.List returned %d facts, want 1: %+v", len(facts), facts)
	}
	if facts[0].Source != domain.FactSourceAgent {
		t.Fatalf("Source = %q, want %q", facts[0].Source, domain.FactSourceAgent)
	}

	// "Agent tried first" (T38's own consequence bullet): the caller merges the agent-sourced
	// comment into known before ever calling Derive.
	known.Comment = facts[0].Comment

	calls := 0
	gate := &PassphraseGate{Passphrase: func() ([]byte, error) {
		calls++
		return []byte(testPassphrase), nil
	}}
	got := gate.Derive(path, known)

	if calls != 0 {
		t.Errorf("Passphrase callback invoked %d times, want 0 — a key present in the agent must never trigger a prompt", calls)
	}
	if got.Material.Comment != wantComment {
		t.Errorf("Comment = %q, want %q (agent-sourced comment must survive Derive unchanged)", got.Material.Comment, wantComment)
	}
}
