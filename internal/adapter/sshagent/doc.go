// Package sshagent is T38's `ssh-agent` derivation source for `--investigate` (D21): connecting
// to a running agent, listing its loaded keys, and exposing each one's public key and comment so
// a caller (chunk M3.6.4's projection assembly) can cross-reference by public key against the
// scanned key set and report the one fact an OpenSSH-format encrypted key's own unencrypted
// public half cannot supply.
//
// golang.org/x/crypto/ssh/agent lives inside the already-required golang.org/x/crypto module
// (tdd.md §2, T1, T38) — this package adds no new module dependency; confirm with
// `go list -m golang.org/x/crypto`.
//
// Declared failure stance (T38, tdd.md §11's safety-stance table): fail-open, always, and
// implemented as such rather than left implicit — List never returns an error. No agent, an
// unset or empty socket path, a dead socket, a refused connection, or a protocol error all
// degrade silently to zero Facts, exactly as they would if the caller had simply never asked. A
// read must stay safe (P5) regardless of whether the environment happens to have an agent
// running; treating an absent agent as a failure would make `--investigate` unusable on a
// machine with no agent at all, which defeats the point of a mode meant to surface more, not
// less.
package sshagent
