---
id: decision-T48
title: '`find key` never prompts for a passphrase; an inapplicable-for-that-reason scheme is an explicit warning, not a silent miss'
date: '2026-09-10'
status: 'Accepted'
---
### Context

[T36](decision-T36.md) supplies multi-scheme `find`'s matching algorithm — which schemes a clue's shape
admits, and that both MD5 candidates are computed rather than one guessed — but does not say
whether `find` may prompt for a passphrase to evaluate a candidate key against a scheme that needs
the decrypted private key. One registered scheme does: `aws-created-rsa`
([T35](decision-T35.md)) hashes the *decrypted* private key, so a 40-hex-digit clue routes to a scheme that,
for an encrypted candidate key with no already-derivable private material, cannot be evaluated at
all without exactly the passphrase [T39](decision-T39.md) scopes prompting to `--investigate` alone.
`tdd.md` §9's global-flag table already lists `--investigate`'s own cost as *"speed, and possibly
a passphrase prompt"* and states the flag is absent from every other verb×noun cell — `find`
included — but nothing said what `find` should report when it hits exactly the case that flag
exists to unlock.

### Decision

`find key` never prompts for a passphrase, under any circumstance. [T39](decision-T39.md)'s prompt is scoped
entirely to `--investigate`; `find` is not that flag, and does not borrow its consent. A key that
cannot be evaluated against a candidate scheme because the decrypted private key is unavailable —
concretely, [T35](decision-T35.md)'s `aws-created-rsa` scheme against an encrypted candidate key — is **never
silently absent from the result**: it is reported as an explicit warning, naming the scheme and
how many keys in the search could not be evaluated against it, distinct from an honest "this key's
fingerprint under this scheme does not match."

### Rationale

[D20](decision-D20.md)'s own rationale is that a false negative during `find` reads as "you
don't have this key," which is a confidently wrong answer, not a missing one. Treating "never
evaluated" identically to "evaluated and did not match" reintroduces that exact failure one layer
inside the search itself: a user holding the right key, encrypted, sees no signal distinguishing
"hasp checked and this isn't it" from "hasp never actually looked." An explicit warning keeps that
distinction visible without asking `find` to do what only `--investigate` is licensed to do — one
passphrase, held for one operation, that operation being `--investigate`, not `find`
([D19](decision-D19.md), [P3](../../docs/design.md#4-principles)). This also keeps `find` fast and
non-interactive, matching [J2](../../docs/design.md#7-journeys)'s framing of identification as an ordinary,
low-friction lookup — the opposite of [J10](../../docs/design.md#7-journeys)'s deliberately higher-friction
investigation.

### Consequence

- `internal/app.FindKeys` returns `(matches []FindMatch, warnings []string)` rather than a bare
  slice: `keyfile.OpenMaterial` is always called with a `nil` passphrase, and a scheme reporting
  `domain.ReasonPrivateKeyUnavailable` for a candidate key increments that scheme's own warning
  count rather than being dropped.
- Both renderers surface the warning: `render.JSON`'s existing `warnings` field
  ([T14](decision-T14.md)) needs no contract change, and the human renderer gains its own channel (stderr,
  after the results table — advisory, not part of the answer stdout carries, `tdd.md` §10).
- `tdd.md` §9's `find` cell for **key** and §18's "Multi-scheme `find`" subsection are amended to
  state this plainly.
