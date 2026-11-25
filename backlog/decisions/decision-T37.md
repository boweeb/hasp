---
id: decision-T37
title: 'Confidence is a closed, permanent vocabulary in the output contract'
date: '2026-09-03'
status: 'Accepted · Amends: T14, T29, T31'
---
### Context

[D22](decision-D22.md) ratified [P10](../../docs/design.md#4-principles): every reported fact carries a
status from a closed set — `derived`, `confirmed`, `possible`, `unknown`. [T14](decision-T14.md) already makes the `--json` envelope a
public contract, [T29](decision-T29.md) already makes a `check` finding's `id` permanent, and [T31](decision-T31.md)
already enumerates the compatibility surface those two entries imply. Confidence has to land in
exactly that surface, not beside it, or it is a promise nobody actually made.

### Decision

An `--investigate` response carries an `origins` array (and, more generally, any reported fact
carries the same shape wherever P10 applies):

```json
"origins": [
  {"id": "aws-ec2-created", "confidence": "possible",
   "because": ["algorithm=rsa", "format=pem", "no-console-fingerprint-supplied"]}
]
```

`because` carries **machine-readable evidence tokens, never prose** — a fixed vocabulary of short,
`key=value`-shaped strings a consumer can branch on without parsing English. Free text belongs in
the human renderer only, which is free to turn the same tokens into a sentence.

**Contrast with [T29](decision-T29.md)'s `severity` explicitly, because the two look alike and are not.**
`severity` is deliberately re-tunable — hasp may decide next year that a finding deserves a
different severity without breaking anything, because [T29](decision-T29.md) says so outright. `confidence`
is **closed and permanent**: a consumer filtering on `derived` to decide whether to *trust* a
value is making a safety decision, and a vocabulary a producer can silently expand out from under
that filter would be a contract violation dressed as a feature. Nothing about the confidence
vocabulary is subject to future re-tuning the way `severity` is.

[T31](decision-T31.md)'s compatibility-surface table gains a seventh row: **the confidence
vocabulary — `derived`, `confirmed`, `possible`, `unknown` — is closed; adding, removing, or
redefining a value is a breaking change**, exactly like [T29](decision-T29.md)'s finding `id`s (row 4) and
unlike its `severity` (explicitly named in the non-contract).

**A correction to [T31](decision-T31.md)'s own body, recorded here since this is the entry that amends it.**
T31's Decision section states *"v1.0.0 is tagged at the close of
[M3.5](../../docs/roadmap.md#55-m35--hardening)"* — true when T31 was written, on 2026-09-02, against the
milestone structure that existed then. [D21](decision-D21.md) and [D22](decision-D22.md)
split that milestone: `roadmap.md` §5.5 (M3.5 — Hardening) now closes without tagging anything, and
a new §5.6 (M3.6 — Investigation) closes with the tag instead, per `design.md` §9. T31's sentence
is **pre-amendment text** — the same treatment [T27](decision-T27.md) and [T29](decision-T29.md) give an earlier draft's
superseded claim — and the current truth is `roadmap.md` §5.6, not §5.5.

### Rationale

[D22](decision-D22.md)'s own rationale is the evidence this taxonomy is right-sized: this one
feature genuinely needs all four statuses at once — a matched RSA console fingerprint is
`confirmed` ([T36](decision-T36.md)); an ED25519 guess is `possible`; a value read straight off an artifact
(most of what `list key` already reports) is `derived`; an underivable value stays `unknown`,
exactly as §5.1 already uses the word. Landing it in the same three entries that already govern
the output contract, rather than inventing a fourth place for stability promises to live, is what
keeps [T31](decision-T31.md)'s enumeration actually complete.

### Consequence

- `Envelope` itself ([T14](decision-T14.md)) is unchanged in shape — `version`, `kind`, `data`, `warnings` —
  because confidence lives inside `data`, not the envelope wrapper. T14 is amended in the sense
  that its `data` payloads for investigation-bearing kinds now carry a new, contract-bound shape,
  not in the sense that the struct above changes.
- [T29](decision-T29.md)'s own text is unchanged and remains correct; this entry's contrast with it is worth
  reading alongside T29 for any future reader deciding whether a new field belongs in the
  `severity` camp (re-tunable) or the `confidence` camp (closed).
- `tdd.md` §16's compatibility-surface table gains the row described above, citing this entry.
