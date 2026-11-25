---
id: decision-T50
title: '`find`''s comparison folds case and separator punctuation at compare time; `fpscheme.Normalize` stays exactly as it is'
date: '2026-09-10'
status: 'Accepted'
---
### Context

[J2](../../docs/design.md#7-journeys) promises *"Punctuation and case shouldn't matter,"* and
[D20](decision-D20.md) widens that promise to span every registered fingerprint scheme rather
than only spellings of one. [T36](decision-T36.md) specifies `fpscheme.Normalize`'s own transformations —
strip colons and internal whitespace, lowercase hex digits, strip base64 `=` padding, tolerate a
`SHA256:`/`MD5:` prefix — and `internal/adapter/fpscheme/normalize.go` implements exactly that,
deliberately **not** lowercasing base64: base64 is case-sensitive, so folding its case would
silently change which bytes a SHA-256 fingerprint names, not merely how it is spelled. That
implementation is correct on its own terms — a normalized value is also the value compared and
displayed, and corrupting it would be a real, not cosmetic, defect.

But J2's promise and base64's case-sensitivity are individually correct and jointly
unsatisfiable as a single-function contract: `hasp find key sha256:T4RN-6Go9-uzGH` (this
project's own committed `testdata/script/key-find.txtar` case, predating this chunk) must match
the key whose real fingerprint is `SHA256:t4rn6Go9uzGHCyff1lwYWtX+swf+EQhuqmE2dHSqkFE` — different
case throughout, and separator hyphens (`internal/cli/fullinventory_test.go`'s own case,
`T4RN-6go9`, is the same shape) that never appear as real content in any value a registered scheme
actually computes. `fpscheme.Normalize`, applied faithfully to [T36](decision-T36.md)'s spec, preserves both
the case difference and the hyphens verbatim — neither is a colon, whitespace, padding character,
or scheme prefix — so a direct comparison against `Normalize`'s own output silently fails on both
committed cases at once, a regression introduced when `internal/app.FindKeys` (M3.6.3's WIP) moved
from an older, single-scheme normalizer that folded both away unconditionally.

### Decision

Leave `fpscheme.Normalize` exactly as [T36](decision-T36.md) specifies it — its case- and shape-preserving
behavior remains correct for what it is actually for: [T36](decision-T36.md)'s own `CandidateSchemes` shape
routing needs an exact, uncorrupted length and alphabet to route a full-length clue to the right
candidate scheme, and a scheme's own `Compute` output must not be silently rewritten either.

`find`'s own comparison step folds **both** operands — a candidate scheme's computed value and the
normalized clue — down to lowercase letters and digits only, immediately before the substring
check, in a helper local to `internal/app.FindKeys` (`compareFold`) rather than inside
`fpscheme.Normalize` itself. This folds away case, exactly as J2 promises, and — necessarily
broader than the case tension alone — separator punctuation such as a hyphen a human inserts
purely for readability, since none of [T35](decision-T35.md)'s four registered schemes ever emits `-` or `_`
as real content: the two colon-hex schemes use only hex digits and colons, and the SSH-native
scheme emits standard, not URL-safe, base64 (`+`, `/`).

### Rationale

The residual risk this fold accepts is a theoretical false positive: two distinct base64
fingerprints, or two distinct values otherwise, differing only in a fold this comparison forgives.
Across a 43-character base64 digest or a 32/40-digit hex digest that is effectively impossible, and
even if it occurred, `find` returns a list the user disambiguates rather than acting unattended —
whereas a false *negative* is exactly the silent miss [D20](decision-D20.md) exists to
eliminate. The asymmetry favours folding, the same judgment [P10](../../docs/design.md#4-principles)'s
confidence vocabulary already makes structurally: `find`'s match evidence is reported as
`confirmed` or `possible`, never asserted beyond what the evidence supports, so a comparison that
occasionally over-matches is caught by the very origin-evidence distinction [T36](decision-T36.md) already
requires the renderer to preserve, not silently trusted as certain.

Folding at the comparison step, not inside `Normalize`, keeps [T36](decision-T36.md)'s shape-routing
contract — and every existing test asserting `Normalize`'s and `CandidateSchemes`' exact output —
intact and unchanged; this is an additive change to how `find` compares two already-normalized
values, not a redefinition of what "normalized" means.

### Consequence

- `internal/app.FindKeys`'s match comparison uses `compareFold` on both the candidate scheme's
  computed value and the normalized clue, in place of a direct substring check against
  `fpscheme.Normalize`'s own output.
- `tdd.md` §18's "Multi-scheme `find`" subsection gains a sentence stating that `find`'s own
  comparison step folds case and separator punctuation, distinct from `Normalize`'s own,
  narrower, shape-preserving transformations.
- `internal/cli/testdata/script/key-find.txtar`'s and `internal/cli/fullinventory_test.go`'s
  existing mangled-clue assertions — both predating this chunk — pass unmodified; neither needed a
  golden-value change, only this fix to what compares them.
