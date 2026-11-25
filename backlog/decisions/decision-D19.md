---
id: decision-D19
title: 'P3 is split: custody stays absolute, access becomes consent-gated'
date: '2026-09-03'
status: 'Accepted · Narrows: P3 · Generalizes: D17'
---
### Context

P3 bundles three distinct jobs: a scope boundary (hasp is not a vault, agent, or passphrase
manager, §3.2), a data-flow rule (private key bytes never enter hasp's own records), and an
access rule (hasp never asks for a passphrase to open something that already exists). Only the
access rule has ever blocked anything, and it has now blocked twice: [D17](decision-D17.md) narrowed it once,
for `new key`. The AWS created-RSA fingerprint scheme — which hashes the *private* key, not the
public half — and the comment on an encrypted OpenSSH key both need it narrowed again, and neither
is a case D17 covers, because both are reads of a key that already exists. Worth stating plainly,
now that it has recurred: the access rule was a reasonable default chosen to limit complexity at
project liftoff, not a requirement anything else in the design actually rests on.

### Decision

Keep the scope boundary and the data-flow rule **absolute and unchanged.** Replace the access rule
with the consent rule — [D17](decision-D17.md)'s four constraints, promoted from a special case covering one
write path to the general rule covering every read: hasp may read private key material only when
the user has explicitly asked for an operation that requires it, only for that operation's
duration, held in memory and zeroed after use; never automatically, and never as a side effect of
a read that did not ask for it.

### Rationale

Retiring P3 outright was considered and rejected. It has 18 citations across `docs/` and 4 in the
Go source, and nearly all of them cite the scope boundary or the data-flow rule, both of which
remain true without qualification. Retiring the whole principle would orphan [D14](decision-D14.md)'s
reasoning for not annotating existing private keys, §5.1's derivation-gap rationale,
[D17](decision-D17.md)'s own narrowing, and [T6](decision-T6.md)'s structural audit — which states
that *"a reviewer reading a diff that adds passphrase-accepting code anywhere else in the tree is
reading a P3 violation on sight"* — every one of which is an argument about custody or data flow,
not about access. Splitting the principle preserves all of them; narrowing the whole thing again,
the way D17 did, would not.

### Consequence

The audit heuristic changes shape rather than disappearing. Passphrase-accepting code is no longer
forbidden outside `new key`, so what a reviewer checks at a call site is no longer "does this
exist" but whether it is **user-initiated, scoped to one operation, and zeroed** — the same three
tests D17 already applied to key generation, now the general review question for any read. §5.1's
derivation gap is also no longer permanent for every case it names: a fact that was `unknown`
because deriving it needed a passphrase may become derivable the moment the user explicitly asks.
