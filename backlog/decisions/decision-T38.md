---
id: decision-T38
title: '`ssh-agent` is a derivation source for public facts, and its contribution is labelled'
date: '2026-09-03'
status: 'Accepted'
---
### Context

§5.1's derivation gap is total for an OpenSSH-format encrypted key's **comment** — the public half
is embedded unencrypted in the private file, so the fingerprint and algorithm are derivable, but
the comment lives inside the encrypted blob and is not. A loaded `ssh-agent` already holds the
decrypted key in memory for exactly this reason — it has to, to sign challenges — and its protocol
exposes both the public key and the comment for anything currently loaded, with **no passphrase
and no decryption performed by hasp at all**.

`golang.org/x/crypto/ssh/agent` was confirmed this session to exist inside the already-required
`golang.org/x/crypto` module ([T1](decision-T1.md)), at the version already pinned (`v0.55.0`), and its
`agent.Key` type carries a `Comment` field. **This adds no new module dependency** — the same
module `go.mod` already requires for the entire key-parsing read path supplies the agent client
too.

### Decision

Under `--investigate` ([D21](decision-D21.md)), hasp connects to `SSH_AUTH_SOCK` if it is set,
lists the agent's loaded keys, and cross-references each by public key against the scanned key
set. For every match, the agent's comment is reported for that key, **labelled `agent-sourced`**
rather than merged into the ordinary `derived` bucket — the same treatment
[T16](decision-T16.md) gives implicit-default bindings: a real fact, but a distinct *kind* of fact, and
merging kinds that arrived differently is exactly the dishonesty T16 was written to avoid.

**Named honestly: an agent-sourced fact is real but transient.** Unlike a fact read from a file,
which persists until the file changes, an agent-sourced fact is only as durable as the agent's own
running state — it can vanish the moment the agent restarts, the key is removed from it, or the
process exits. This is a genuine wrinkle in [P1](../../docs/design.md#4-principles)'s "the world is the
truth" framing: the *agent* is part of the world at the moment hasp asks, and gone from it a
moment later, which is precisely why the label matters more here than anywhere else P1 applies.

**Declared failure stance: no agent, or `SSH_AUTH_SOCK` unset, degrades silently to what is
derivable without it — never an error.** A read must stay safe
([P5](../../docs/design.md#4-principles)) regardless of whether the environment happens to have an agent
running, and treating an absent agent as a failure would make `--investigate` unusable on a
machine with no agent at all, which defeats the point of a mode meant to surface more, not less.

### Rationale

This closes the comment gap for OpenSSH-format encrypted keys **without ever touching a
passphrase** — it is strictly cheaper and safer than [T39](decision-T39.md)'s prompt-based path, and is tried
first for exactly that reason. It serves [J10](../../docs/design.md#7-journeys) directly: an investigation
needs more than a plain read, and an agent a user already has running is data hasp was leaving on
the table. Zero new dependency keeps [T1](decision-T1.md)'s "no subprocess" boundary, and the Go-and-stdlib-
first preference [T27](decision-T27.md) already established, both intact.

### Consequence

- `tdd.md` §5 gains the agent as a named derivation source, feeding the same projection step
  [T35](decision-T35.md)'s scheme registry feeds.
- `tdd.md` §9's `show key --investigate` and `list key --investigate` cells surface agent-sourced
  comments, visibly labelled as such in both renderers.
- `tdd.md` §11 gains a fail-open row: no agent / `SSH_AUTH_SOCK` unset → degrade to what is
  derivable without it, never an error.
- [T39](decision-T39.md)'s passphrase-gated path tries the agent first, so a key already loaded never
  triggers a prompt for a fact this entry already supplies.
