---
id: decision-T28
title: 'Relative `IdentityFile` resolves against the key directory, and says so'
date: '2026-08-29'
status: 'Accepted · Amends: T5, T16'
---
**Closes:** [`docs/tdd.md`](../../docs/tdd.md) §15's relative-path question

### Context

An `IdentityFile` value may be a bare relative path — no leading `~`, no leading `/`. Real `ssh`
resolves it against **the working directory of the `ssh` process at connection time.**

hasp is never that process. It has no connection, no connection time, and no working directory
bearing any relationship to the one `ssh` will have. The value is therefore not resolvable by hasp
the way `ssh` resolves it — not *difficult*, but **undefined**.

`tdd.md` §5 proposed resolving against the key directory and flagged, correctly, that this was
*"a judgment call, not a transcription of OpenSSH's own rule."*

### Options

**(i) Resolve against the key directory, silently.** Matches what such a path almost always means
in a hand-written config. *Cost:* hasp reports a binding as fact when `ssh` may resolve it
somewhere else — asserting what it cannot verify, which is the thing P1 exists to prevent.

**(ii) Refuse to resolve; report the binding as unresolvable.** The strictest P1 reading,
symmetric with an unresolvable `%` token. *Cost:* J3's one-screen answer acquires a hole where a
perfectly working binding should be, and `check` reports a problem the user does not have. hasp
would be technically honest and practically wrong.

**(iii) Resolve, and report the divergence.** Both.

### Decision

**Option (iii).** hasp resolves a bare relative `IdentityFile` against the key directory
(`--key-dir`, default `~/.ssh`) — **and** emits a `check` finding, `relative-identityfile`
([T29](decision-T29.md), `severity: warning`), naming the raw value, the path hasp resolved it to, and the
fact that `ssh` will resolve it against its own working directory instead.

### Rationale

The two principles this sits between stop conflicting once the answer is allowed two parts. P7
asks the tool to answer the question — an inventory with a hole where a working binding belongs is
a worse answer than a resolved one. P1 forbids asserting what cannot be confirmed. A resolved
value *accompanied by a stated caveat* asserts nothing false: it reports what hasp did, and it
reports that `ssh` may do otherwise.

Choosing between them, rather than doing both, is what forces the bad trade. This is the same
shape as `design.md` §5.1's derivation gap, which reports `unknown` rather than guessing or
failing: hasp says exactly what it knows, and exactly what it does not.

**Rarity cuts toward the finding, not against it.** Bare relative `IdentityFile` values are
uncommon in real configs, so the finding costs almost nothing in noise. A rule that would be
unbearable at high frequency is free at this one.

### Consequence

- **`design.md` is deliberately unaffected.** This is a fact about `ssh_config(5)` and hasp's
  reading of it, not about hasp's domain — no principle moves and no upstream decision is needed.
  It is recorded here because `tdd.md` §5's divergence must be traceable to something, not because
  the design had a gap.
- `check` gains exactly one finding id ([T29](decision-T29.md)).
- The divergence stays documented in `tdd.md` §5 as a *stated* divergence. A user who hits it
  should learn why from the technical design, not by reading source.
- **If a real config ever shows key-dir resolution to be the wrong guess, the fix is to change
  what hasp resolves to — not to remove the finding.** The finding is what makes the resolution
  rule safe to change later.
