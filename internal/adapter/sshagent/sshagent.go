package sshagent

import (
	"bytes"
	"io"
	"net"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/boweeb/hasp/internal/domain"
)

// Fact is one of the agent's currently loaded keys, reduced to what T38's derivation source
// actually needs: a public key to cross-reference against the scanned key set, and the comment
// the agent protocol exposes for it — the one fact an OpenSSH-format encrypted key's own,
// unencrypted public half cannot supply on its own.
//
// Source is always domain.FactSourceAgent for a value this package produces — set explicitly on
// every Fact, rather than left for a caller to remember to attach after the fact, so the label
// travels with the value from the moment it is produced (domain.FactSource's own doc comment;
// T38's "labelled, never merged into the plain derived bucket").
type Fact struct {
	Public  ssh.PublicKey
	Comment string
	Source  domain.FactSource
}

// List connects to the agent listening on sockPath (a caller normally passes
// os.Getenv("SSH_AUTH_SOCK") — this package never reads the environment itself, the same
// Request-only framing tdd.md §3 and internal/cli/passphrase.go's package comment already apply
// to a passphrase mode) and returns every key it currently has loaded, as Facts.
//
// Declared failure stance (T38, tdd.md §11): fail-open, unconditionally. sockPath == "" (the
// unset-env-var case), a dead or nonexistent socket, a connection refused, and an agent-protocol
// error mid-List are all indistinguishable to a caller — every one of them degrades silently to
// a nil slice, never an error. This is a deliberate, load-bearing part of the contract, not an
// oversight: a read must stay safe (P5) whether or not the machine happens to have an agent
// running, and there is no case in which failing the whole --investigate run over an absent or
// misbehaving agent would be the right call (tdd.md §11's own fail-open row for exactly this).
//
// An individual key the agent reports that hasp cannot even parse as a public key is skipped,
// not fatal to the rest — the same per-item fail-open discipline §11 applies to a bad file
// mid-scan, applied here to a bad agent entry mid-list.
func List(sockPath string) []Fact {
	if sockPath == "" {
		return nil
	}
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return nil
	}
	defer func() { _ = conn.Close() }()
	return listFrom(conn)
}

// listFrom is List's dial-independent core: everything past "here is a live connection to an
// agent." Split out so tests can exercise the actual agent-protocol exchange over a net.Pipe
// instead of a real filesystem socket — avoiding a real socket's ~108-byte path-length cap on
// Linux (t.TempDir() nests deep enough to blow past it) and the flakiness that comes with racing
// a real listener's accept loop — while List itself stays the one place that ever dials a real
// path.
func listFrom(conn io.ReadWriter) []Fact {
	client := agent.NewClient(conn)
	keys, err := client.List()
	if err != nil {
		return nil
	}

	facts := make([]Fact, 0, len(keys))
	for _, k := range keys {
		pub, pErr := ssh.ParsePublicKey(k.Blob)
		if pErr != nil {
			// A key blob this agent reports but hasp cannot parse as a valid SSH public
			// key — skip it (fail-open, per this function's own doc comment) rather than
			// discard every other key the agent successfully reported.
			continue
		}
		facts = append(facts, Fact{Public: pub, Comment: k.Comment, Source: domain.FactSourceAgent})
	}
	return facts
}

// MatchComment cross-references facts against pub "by public key" — T38's own words — and
// returns the agent's comment for whichever Fact carries the identical SSH wire-format public key,
// if any. This is the chunk M3.6.4 call site's one and only agent lookup for a given candidate
// key; golang.org/x/crypto/ssh stays confined to this adapter package for exactly this comparison
// (tdd.md §18's "T35's registry needs x/crypto/ssh" boundary extended to the one other place this
// tree ever compares two ssh.PublicKey values).
//
// pub == nil always returns ok == false — a byPath-identified key (T1's undecidable row: legacy
// PEM, encrypted, no .pub sidecar) has no derivable public key at all, so there is no signal left
// on hasp's side to compare the agent's own listing against. tdd.md §18's own "known limitation,
// not yet worth its own T-log entry" names this exact case: T38 scopes cross-referencing to "by
// public key" and is silent on the byPath case, so such a key's comment stays unknown even with
// the right agent running.
func MatchComment(facts []Fact, pub ssh.PublicKey) (comment string, ok bool) {
	if pub == nil {
		return "", false
	}
	want := pub.Marshal()
	for _, f := range facts {
		if f.Public == nil {
			continue
		}
		if bytes.Equal(f.Public.Marshal(), want) {
			return f.Comment, true
		}
	}
	return "", false
}
