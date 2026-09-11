package app

import (
	"github.com/boweeb/hasp/internal/adapter/fpscheme"
	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/domain"
)

// PassphraseFunc is D19/T39's injected passphrase source. internal/cli decides whether one
// exists at all — a TTY is attached, so it is safe to interrupt the user — and, if so, supplies
// x/term.ReadPassword's prompt (internal/cli/passphrase.go's existing promptPassphrase); nil
// means "nobody is present to ask," T39's declared degrade case, never "ask and get told no" (a
// wrong/rejected passphrase is a different case entirely — see PassphraseGate.Derive below).
//
// internal/app never constructs this itself and never checks an env var, a TTY, or a settings
// file to decide — the same Request-only framing tdd.md §3 and internal/cli/passphrase.go's own
// package comment already establish for `new key`'s passphrase mode, extended here to reads.
type PassphraseFunc func() ([]byte, error)

// PassphraseGate implements T39's four load-bearing rules for D19-gated key material derivation
// under --investigate:
//
//  1. Try the agent first (T38) — a key already loaded needs no prompt at all. This gate does
//     not talk to an agent itself (internal/adapter/sshagent, chunk M3.6.2's other half, is a
//     wholly separate source with its own contribution — the comment); "tried first" is
//     satisfied by ordering at the call site, not inside this type. Concretely: T38's agent path
//     can only ever supply a comment (agent.Key.Comment), and this file's own worked-out finding
//     is that x/crypto/ssh's public decrypt API (ParseRawPrivateKeyWithPassphrase) discards the
//     OpenSSH comment field even after a correct passphrase — verified against
//     golang.org/x/crypto/ssh's own source this session — so a prompt-then-decrypt can *never*
//     recover a comment this gate's own decrypt path would supply. The agent is therefore not
//     merely tried "first" for the comment fact; it is the *only* source for it. This gate's own
//     sole trigger is aws-created-rsa (see 2 below), which no agent can ever answer either — an
//     agent holds a decrypted key only to sign challenges with it, and its protocol never
//     exposes raw private key bytes to a caller (T38's own context section: "no passphrase and
//     no decryption performed by hasp at all"). The two sources are therefore not actually in
//     competition for anything: nothing this gate does would ever make an agent lookup
//     redundant, and nothing an agent lookup does would ever make this gate's prompt redundant.
//  2. Prompt only where a passphrase would unlock something otherwise completely unknown —
//     fpscheme.CreatedRSAPromptWorthy answers this from material already at hand (T39's
//     Decision text: "the created-RSA scheme is the case that matters"; an OpenSSH key's comment
//     alone was never a sufficient trigger even before the finding in (1) above made it a moot
//     one).
//  3. One passphrase, tried across every candidate key in the invocation — Derive prompts at
//     most once per PassphraseGate value (obtain below), regardless of how many times it is
//     called.
//  4. Held in memory and zeroed after — Close, called once by the caller after every candidate
//     key has been through Derive.
//
// A PassphraseGate is built once per --investigate invocation (chunk M3.6.4 wires it up; this
// chunk builds and tests the policy in isolation) and reused across every candidate key that
// invocation examines.
type PassphraseGate struct {
	// Open reads and, if a passphrase is supplied, decrypts a candidate key file. Defaults to
	// keyfile.OpenMaterial when nil; overridable for tests.
	Open func(path string, passphrase []byte) (keyfile.Material, error)

	// Passphrase is internal/cli's injected callback. nil means nobody to ask (T39's degrade
	// stance); see PassphraseFunc's own doc comment.
	Passphrase PassphraseFunc

	prompted   bool
	passphrase []byte
	promptErr  error
}

// DerivedMaterial is one candidate key's outcome from PassphraseGate.Derive: either unlocked
// material (Unlocked, carrying the decrypted Material.Private a scheme needs), or an honest "why
// not" (Reason) — never an error, because none of the outcomes this type can represent is a
// failure (§11's fail-open/degrade rows for exactly this case).
type DerivedMaterial struct {
	Material keyfile.Material
	Unlocked bool
	Reason   domain.ReasonToken
}

// Derive decides, for one candidate key at path, whether unlocking it is worth asking for and —
// if so — asks (at most once per PassphraseGate, rule 3 above) and attempts the decrypt.
//
// known is the material already derivable without any passphrase — ordinarily
// keyfile.OpenMaterial(path, nil). The caller supplies it rather than Derive re-deriving it, so
// a caller that already merged an agent-sourced comment (internal/adapter/sshagent) into known's
// Comment field before calling Derive gets that fact preserved through an unrelated call — on
// every path, including the successful-unlock one: Derive never touches or clears
// Material.Comment itself, and on a successful decrypt it carries known.Comment forward onto the
// freshly-opened Material returned from openFn whenever that fresh material has no comment of its
// own (mat.Comment == ""), rather than silently dropping it.
//
// That carry-forward is not merely tidy — it is the only way a caller ever gets an agent-sourced
// comment back at all when a correct passphrase is also supplied. T49 found that
// golang.org/x/crypto/ssh's passphrase-decrypt API (ParseRawPrivateKeyWithPassphrase, wrapped by
// keyfile.OpenMaterial) discards the OpenSSH comment field even after a correct passphrase — the
// freshly-opened mat this function receives from openFn has Comment == "" whenever there is no
// `.pub` sidecar to repopulate it from (keyfile.OpenMaterial's own doc comment), which is exactly
// the shape an OpenSSH-format encrypted key with no sidecar takes. ssh-agent (T38) is therefore
// the *only* source for that comment, and the caller's merge into known happened before this call
// precisely so the fact would survive it (tdd.md §18's "ssh-agent" subsection, roadmap.md §5.6's
// M3.6.4 chunk row). A prior version of this function returned openFn's mat verbatim on this
// path, silently erasing a caller-merged comment on the one branch where the caller had done the
// most work (a correct passphrase) to get here — never overwrite a genuine, non-empty
// mat.Comment though: a real `.pub` sidecar comment is its own legitimate plain-read fact, and
// the caller's agent-sourced merge must not clobber it.
func (g *PassphraseGate) Derive(path string, known keyfile.Material) DerivedMaterial {
	if known.Private != nil {
		// Already unencrypted, or already decrypted by an earlier call this invocation
		// somehow supplied — nothing to unlock, nothing to ask for.
		return DerivedMaterial{Material: known, Unlocked: true}
	}

	worthPrompting, _ := fpscheme.CreatedRSAPromptWorthy(known)
	if !worthPrompting {
		// T39 rule 2: a passphrase here would not unlock anything currently unknown — never
		// prompt just to reconfirm what is already derivable, and never decrypt a key this
		// gate already knows is not an aws-created-rsa candidate.
		return DerivedMaterial{Material: known, Unlocked: false, Reason: domain.ReasonSchemeNotApplicable}
	}

	passphrase, err := g.obtain()
	if err != nil || passphrase == nil {
		// No TTY (nil, no error — PassphraseFunc is nil) and a failed prompt (err != nil,
		// e.g. term.ReadPassword erroring against a non-interactive fd) degrade identically:
		// T39's declared stance is "report what's derivable, mark the rest unknown," never
		// an error surfaced to this gate's own caller for either cause (tdd.md §11's
		// fail-open row for this exact case, roadmap.md §5.6 exit criterion 5).
		return DerivedMaterial{Material: known, Unlocked: false, Reason: domain.ReasonPassphraseRequiredNoTTY}
	}

	openFn := g.Open
	if openFn == nil {
		openFn = keyfile.OpenMaterial
	}
	mat, oErr := openFn(path, passphrase)
	if oErr != nil {
		// path itself unreadable/unrecognizable — the same shape keyfile.OpenMaterial's own
		// error contract already uses (its doc comment: "reserved for path itself being
		// unreadable or not a recognizable private key at all"); not this gate's job to
		// interpret any further.
		return DerivedMaterial{Material: known, Unlocked: false, Reason: domain.ReasonPrivateKeyUnavailable}
	}
	if mat.Private == nil {
		// A wrong/rejected passphrase (keyfile.OpenMaterial's own doc comment: "reported as
		// Material.Private == nil, never as an error"). An honest degrade, not this gate's
		// failure — the same passphrase may still be correct for a different candidate key
		// later in this invocation, so the gate keeps it and keeps going (rule 3).
		return DerivedMaterial{Material: mat, Unlocked: false, Reason: domain.ReasonPrivateKeyUnavailable}
	}
	if mat.Comment == "" {
		// The successful-unlock path: carry a caller-merged comment (this function's own doc
		// comment, T49) forward onto the freshly-opened mat, since openFn's own decrypt can
		// never have supplied one itself (T49) and mat.Comment == "" here means no `.pub`
		// sidecar repopulated it either. A non-empty mat.Comment is left untouched — a real
		// sidecar comment is its own plain-read fact and must never be overwritten by a merge
		// performed for an unrelated (agent) reason.
		mat.Comment = known.Comment
	}
	return DerivedMaterial{Material: mat, Unlocked: true}
}

// obtain returns this invocation's one passphrase, prompting at most once (rule 3) and
// memoizing the result — including a nil callback or a prompt error — so a second Derive call
// for a second candidate key never prompts again, even when the first prompt failed or nobody
// was there to answer it.
func (g *PassphraseGate) obtain() ([]byte, error) {
	if g.prompted {
		return g.passphrase, g.promptErr
	}
	g.prompted = true
	if g.Passphrase == nil {
		return nil, nil // nobody to ask — T39's degrade stance, never an error.
	}
	g.passphrase, g.promptErr = g.Passphrase()
	return g.passphrase, g.promptErr
}

// Close zeroes the held passphrase (rule 4, D19's "zeroed after" constraint). Callers should
// `defer gate.Close()` immediately at construction — the same pattern internal/cli/new.go:61
// already uses for `new key`'s own passphrase buffer (`defer zeroBytes(secret)`) — rather than
// calling it by hand after the last Derive, so a return added later on an error path can never
// accidentally skip it. A PassphraseGate must not be reused after Close.
// (Guarded by TestPassphraseGate_Close_ZeroesThePassphraseBuffer, tdd.md §12.)
func (g *PassphraseGate) Close() {
	for i := range g.passphrase {
		g.passphrase[i] = 0
	}
	g.passphrase = nil
}
