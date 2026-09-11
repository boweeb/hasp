package sshagent

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/domain"
)

// testPassphrase mirrors tools/genfixtures/main.go's own constant of the same name and value —
// a fixed, publicly-known passphrase used only to make an "encrypted" fixture parseable by
// tests, never a secret. Duplicated per-package (as internal/adapter/keyfile's own test suite
// already does) because tools/genfixtures is a `main` package and cannot be imported.
const testPassphrase = "hasp-test-fixture-passphrase"

func fixturesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// this file lives at internal/adapter/sshagent/sshagent_test.go
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "testdata", "keys")
}

// loadedFixtureKey decrypts testdata/keys/ed25519-openssh-encrypted-nopub — committed with an
// empty comment (the fixtures were generated via `ssh-keygen -C ""`, tools/genfixtures/main.go)
// — and returns its decrypted private key, ready for agent.AddedKey. The non-empty comment this
// test needs is supplied separately, via AddedKey.Comment, not from the fixture itself.
func loadedFixtureKey(t *testing.T) any {
	t.Helper()
	path := filepath.Join(fixturesDir(t), "ed25519-openssh-encrypted-nopub")
	mat, err := keyfile.OpenMaterial(path, []byte(testPassphrase))
	if err != nil {
		t.Fatalf("OpenMaterial(%s): %v", path, err)
	}
	if mat.Private == nil {
		t.Fatalf("OpenMaterial(%s) with the correct passphrase: Private is nil, want populated", path)
	}
	return mat.Private
}

// pipedAgent starts agent.ServeAgent against keyring over one end of a net.Pipe and returns the
// other end for a client to dial against — sshagent.go's doc comment explains why a net.Pipe is
// preferred over a real filesystem socket for the common case (path-length and flakiness).
// The server goroutine's own error is not asserted: it always errors once the pipe closes
// (io.EOF or "io: read/write on closed pipe"), which is this helper's own teardown, not a test
// failure.
func pipedAgent(t *testing.T, keyring agent.Agent) net.Conn {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	go func() {
		_ = agent.ServeAgent(keyring, serverConn)
	}()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})
	return clientConn
}

// TestListFrom_FakeAgent is roadmap.md §5.6 exit criterion 4's own test, and tdd.md §12's named
// guard: a fake agent (agent.NewKeyring() + agent.ServeAgent), with the fixture key loaded under
// a non-empty comment supplied via agent.AddedKey (the committed fixtures themselves carry an
// empty comment — genfixtures used `ssh-keygen -C ""` — so the comment must come from the test,
// or it proves nothing). Asserts the comment is reported and labelled agent-sourced.
func TestListFrom_FakeAgent(t *testing.T) {
	priv := loadedFixtureKey(t)

	keyring := agent.NewKeyring()
	const wantComment = "hasp-test-agent-comment"
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv, Comment: wantComment}); err != nil {
		t.Fatalf("keyring.Add: %v", err)
	}

	conn := pipedAgent(t, keyring)
	facts := listFrom(conn)

	if len(facts) != 1 {
		t.Fatalf("listFrom returned %d facts, want 1: %+v", len(facts), facts)
	}
	fact := facts[0]
	if fact.Comment != wantComment {
		t.Errorf("Comment = %q, want %q", fact.Comment, wantComment)
	}
	if fact.Source != domain.FactSourceAgent {
		t.Errorf("Source = %q, want %q (T38: labelled, never merged into the plain derived bucket)", fact.Source, domain.FactSourceAgent)
	}
	if fact.Public == nil {
		t.Fatal("Public is nil, want the loaded key's public half")
	}

	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("ssh.NewSignerFromKey: %v", err)
	}
	if !bytes.Equal(fact.Public.Marshal(), signer.PublicKey().Marshal()) {
		t.Error("Public does not match the key actually loaded into the agent")
	}
}

// TestList_NoSocket proves T38's declared failure stance for the unset-SSH_AUTH_SOCK case: List
// degrades to zero Facts, never an error — there is no error return to check, which is itself
// part of the point (a caller cannot even observe a distinction between "no agent" and "agent
// misbehaved").
func TestList_NoSocket(t *testing.T) {
	facts := List("")
	if facts != nil {
		t.Errorf("List(\"\") = %v, want nil", facts)
	}
}

// TestList_DeadSocket proves the same degrade for a socket path that simply does not exist —
// the "a dead socket" case tdd.md §11's fail-open row names explicitly.
func TestList_DeadSocket(t *testing.T) {
	facts := List(filepath.Join(t.TempDir(), "does-not-exist.sock"))
	if facts != nil {
		t.Errorf("List(dead socket) = %v, want nil", facts)
	}
}

// TestList_RealSocket exercises List's own dial path (not just listFrom's protocol-only core)
// against a real, short-lived Unix socket. Deliberately placed in os.TempDir() rather than
// t.TempDir(): t.TempDir() nests under the test's own name and can produce a path near or past
// the ~108-byte AF_UNIX path cap on Linux, which is exactly the flakiness this test must not
// have — see this package's own doc comment and sshagent.go's listFrom comment.
func TestList_RealSocket(t *testing.T) {
	priv := loadedFixtureKey(t)
	keyring := agent.NewKeyring()
	const wantComment = "hasp-test-real-socket-comment"
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv, Comment: wantComment}); err != nil {
		t.Fatalf("keyring.Add: %v", err)
	}

	sockFile, err := os.CreateTemp("", "hasp-sshagent-*.sock")
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

	facts := List(sockPath)
	if len(facts) != 1 {
		t.Fatalf("List(%s) returned %d facts, want 1: %+v", sockPath, len(facts), facts)
	}
	if facts[0].Comment != wantComment {
		t.Errorf("Comment = %q, want %q", facts[0].Comment, wantComment)
	}
}
