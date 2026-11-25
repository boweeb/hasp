---
id: decision-T49
title: 'Narrowing §18/T39''s comment-recovery claim: a passphrase prompt cannot recover an OpenSSH-format key''s comment; only `ssh-agent` can'
date: '2026-09-10'
status: 'Accepted — amends T39'
---
### Context

`tdd.md` §18's "Passphrase-gated derivation" subsection and [T39](decision-T39.md) both state that, when
[T38](decision-T38.md)'s agent path finds nothing loaded, a passphrase prompt is one of two facts
passphrase-gated derivation exists to recover — the other being [T35](decision-T35.md)'s created-RSA scheme —
implying an OpenSSH-format encrypted key's comment is recoverable either way: from the agent, or
by prompting and decrypting.

That second path does not exist. Verified this session against
`$(go env GOMODCACHE)/golang.org/x/crypto@v0.56.0/ssh/keys.go` (the exact version this module
pins, `go.mod`): `parseOpenSSHPrivateKey` parses an OpenSSH-format key's on-wire comment into one
of `openSSHRSAPrivateKey`, `openSSHEd25519PrivateKey`, or `openSSHECDSAPrivateKey` — each of those
three structs carries its own `Comment string` field — but `parseOpenSSHPrivateKey` itself returns
only `(crypto.PrivateKey, error)`, and its caller, the exported
`ParseRawPrivateKeyWithPassphrase`, returns only `(interface{}, error)`. Neither return type
carries the comment anywhere a caller of the public decrypt API can reach — it is parsed, used
internally, and then discarded before the function returns. `internal/adapter/keyfile.OpenMaterial`
calls exactly this function ([T35](decision-T35.md)'s own read path), so no amount of decrypting through
hasp's own code, with a correct passphrase or otherwise, ever recovers this comment. `ssh-agent`
recovers it by a completely different mechanism ([T38](decision-T38.md)): the agent protocol exposes a
loaded key's comment directly, with no decryption performed by hasp at all, which is why that path
works when this one cannot.

### Decision

The passphrase-gated path's scope is **`aws-created-rsa` alone**. An OpenSSH-format encrypted
key's comment is **agent-only** ([T38](decision-T38.md)): if `ssh-agent` has nothing loaded for that key,
the comment stays `unknown`, full stop — a passphrase prompt is not a fallback for it, because
there is no code path by which a correct passphrase would ever produce it.

### Rationale

This is a narrowing forced by a fact about a dependency's own API shape
([P3](../../docs/design.md#4-principles), [D19](decision-D19.md)): D19's consent gate licenses *reading*
private key material the user explicitly asked for, but licensing the read does not manufacture a
return value the library used to perform it never exposes. A subprocess wrapping `ssh-keygen`
directly could recover the comment (it parses the same on-wire structure independently), but
[T1](decision-T1.md) forbids subprocessing for exactly the reasons that rule exists — pure-Go derivation,
no shelling out — so that route is not available either. [J10](../../docs/design.md#7-journeys)'s promise is
to surface everything hasp *can* determine, clearly marked with how sure hasp is; stating a
capability the code cannot actually deliver would be the opposite of that promise, not a detail in
service of it.

### Consequence

- `tdd.md` §18's "Passphrase-gated derivation" subsection is corrected: "two facts need
  [passphrase-gated derivation]" becomes one (`aws-created-rsa`), and the comment gap is described
  as agent-only, closing over the incorrect claim rather than leaving it as current truth.
- `tdd.md` §18's `ssh-agent` subsection gains a note that a `byPath`-identified key ([T1](decision-T1.md)'s
  undecidable row) has no derivable public key at all, so even a loaded agent has no signal on
  hasp's side left to cross-reference against — a related, previously unrecorded limitation this
  same reassessment surfaced.
- Chunk [M3.6.4](../../docs/roadmap.md#56-m36--investigation), which wires `PassphraseGate` and the agent
  source together at a shared call site, implements exactly one passphrase-recoverable fact
  (`aws-created-rsa`) rather than two — a smaller implementation surface than §18 previously
  described, not a smaller one than [T39](decision-T39.md)'s own mechanics (one prompt per invocation, degrade
  without a TTY) required regardless.
