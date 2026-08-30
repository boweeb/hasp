package sshconfig

import "strings"

// Node is one element of a parsed File: a Line, a Directive, a HostBlock, or a MarkedRegion.
// Every byte of the input lands in some Node — Parse never errors (T2).
type Node interface{ node() }

// Line is an unrecognized or purely human line: reconstructed byte-exact from its parts. This
// covers blank lines, comment-only lines, and any keyword line whose keyword is not in the
// natively-managed set (managedKeywords) — an unrecognized directive is carried opaquely and
// passed through verbatim, never an error (P6, T2).
type Line struct {
	Raw        []byte // full line content, not including the terminator
	Terminator []byte // "\n", "\r\n", or nil for a final line with no trailing newline
}

func (*Line) node() {}

func (l *Line) render() []byte {
	if l == nil {
		return nil
	}
	out := make([]byte, 0, len(l.Raw)+len(l.Terminator))
	out = append(out, l.Raw...)
	out = append(out, l.Terminator...)
	return out
}

// Directive is a recognized-keyword line (managedKeywords), preserving every raw byte needed to
// reconstruct it exactly (T17): ssh_config(5) allows "Port 22", "Port = 22", and "Port=22" as
// byte-different, semantically identical spellings, and a node holding only a parsed Keyword and
// Args has nowhere to record which was written.
type Directive struct {
	// LeadingTrivia is this line's own leading whitespace/indentation before Keyword. This is a
	// documented, deliberate narrowing of tdd.md §3's sketch, whose doc comment describes
	// LeadingTrivia as also covering "blank lines/comments immediately preceding this line" —
	// here, those preceding whole lines remain independent Line nodes in the containing Nodes
	// slice instead of being folded into this field. Render is unaffected either way (node order
	// is preserved and each node re-emits its own bytes exactly); this narrower scope keeps
	// parsing a single-line-at-a-time operation with no lookback bookkeeping, which is what T17's
	// "leading indentation" requirement actually needs.
	LeadingTrivia []byte
	Keyword       string // as written; ssh_config keywords are case-insensitive and hasp does
	// not normalize a human's casing outside a region it owns
	RawValue []byte // everything between the keyword and the trailing comment/terminator —
	// separator ('=' or whitespace), quoting, internal spacing, untouched
	// by Parse. Synthesized fresh only when hasp authors the line itself,
	// inside a region it owns (D7 elaboration 1) — out of scope for M1,
	// which never writes.
	Trivia []byte // a trailing inline comment, if any, including its leading whitespace;
	// a '#' inside a double-quoted RawValue span is never treated as this (T17)
	Terminator []byte // "\n", "\r\n", or nil
}

func (*Directive) node() {}

func (d *Directive) render() []byte {
	if d == nil {
		return nil
	}
	out := make([]byte, 0, len(d.LeadingTrivia)+len(d.Keyword)+len(d.RawValue)+len(d.Trivia)+len(d.Terminator))
	out = append(out, d.LeadingTrivia...)
	out = append(out, d.Keyword...)
	out = append(out, d.RawValue...)
	out = append(out, d.Trivia...)
	out = append(out, d.Terminator...)
	return out
}

// Args parses RawValue into tokens on demand, honoring ssh_config(5)'s quoting rule: a
// double-quoted span may contain whitespace and is not split on it. Used by hasp's semantic
// logic (binding resolution, check findings) — never consulted by Render, which only ever
// replays RawValue's bytes.
func (d *Directive) Args() []string {
	return tokenizeArgs(d.RawValue)
}

// managedKeywords is the natively-managed directive set (tdd.md §6): the 21 directives salvaged
// from the predecessor's README.rst, plus Include, which hasp itself writes inside its own
// regions (M2+) and must therefore understand semantically there. Keyed lowercase; comparison is
// case-insensitive per ssh_config(5), but the Keyword field always preserves the bytes as
// written.
var managedKeywords = func() map[string]struct{} {
	names := []string{
		"AddKeysToAgent", "Ciphers", "ControlMaster", "ControlPath", "ControlPersist",
		"ForwardAgent", "HashKnownHosts", "HostKeyAlgorithms", "HostName", "IdentitiesOnly",
		"IdentityFile", "Include", "KexAlgorithms", "LogLevel", "MACs", "PasswordAuthentication",
		"Port", "ProxyCommand", "PubkeyAuthentication", "StrictHostKeyChecking", "User",
		"UserKnownHostsFile",
	}
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[strings.ToLower(n)] = struct{}{}
	}
	return set
}()

func isManagedKeyword(keyword string) bool {
	_, ok := managedKeywords[strings.ToLower(keyword)]
	return ok
}

// isHostOrMatchKeyword reports whether keyword introduces a block (Host or Match). Neither is in
// managedKeywords — they are structural, handled by HostBlock grouping, not treated as an
// ordinary semantic directive.
func isHostOrMatchKeyword(keyword string) bool {
	lower := strings.ToLower(keyword)
	return lower == "host" || lower == "match"
}
