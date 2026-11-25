---
id: decision-D22
title: 'Confidence is part of the fact; P10 is added'
date: '2026-09-03'
status: 'Accepted · Adds: P10 · Generalizes: §5.1''s `unknown`'
---
### Context

§5.1 already established that hasp reports an epistemic status rather than only a value —
`unknown` is a first-class answer, because asserting an unverifiable fingerprint is worse than
admitting ignorance. Investigation needs the same honesty for the *middle* of that range: an
origin inferred from a matched console fingerprint is certain for RSA, an origin guessed from the
key alone is a hint, and an ED25519 origin can never be more than a hint — AWS's own fingerprint
scheme for ED25519 carries no origin signal at all, because created and imported keys produce an
identical fingerprint.

### Decision

Add **P10 — Confidence is part of the fact.** Every reported fact carries a status from a closed
set: `derived` (read directly from the artifact), `confirmed` (established by matching external
evidence the user supplied), `possible` (consistent with the evidence, not established), `unknown`
(cannot be determined). Weak assertions are permitted and useful, but must be marked as such and
must never be rendered in a way that reads as established.

### Rationale

This one feature needs all four statuses, which is the evidence the taxonomy is right-sized: a
matched RSA console fingerprint is `confirmed`; an ED25519 guess from the key alone is `possible`;
a value read straight off an artifact is `derived`; an underivable value stays `unknown`. Three of
the four words cost nothing new — they are already this project's vocabulary. [T12](decision-T12.md)'s
duplicate detection already splits findings into a *confirmed* duplicate and one it reports as
*possible*, "cannot confirm"; `unknown` is §5.1's own word for the derivation gap. Only `derived`
is genuinely new, because nothing before this needed to distinguish "read from the artifact" from
"matched against something the user supplied."

**Alternative rejected:** recording an explicit origin as metadata when hasp creates a key. Three
reasons, the third decisive. §5.8's metadata channel exists only inside marked regions of
configuration files, and §5.9 states plainly that *"there is no per-key marker, because writing
one would mean rewriting a private key file."* Origin-at-creation is also **history**, which §5.7
already places out of scope: *"'This key replaced that one' is history — genuinely underivable,
and out of scope."* And even setting both of those aside, it would help only hasp-created keys —
exactly the population that never needs investigating, because their origin is already known at
the moment of creation.

### Consequence

The confidence vocabulary becomes a **compatibility surface**: a consumer filtering on `derived`
is making a safety decision, so unlike [T29](decision-T29.md)'s deliberately re-tunable
`severity`, this vocabulary is closed and permanent. Downstream work must carry it into
[T31](decision-T31.md)'s contract table.
