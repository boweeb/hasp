---
id: decision-T12
title: 'Key identity: fingerprint when derivable, else canonical path'
date: '2026-08-28'
status: 'Accepted'
---
### Context

[`design.md` §5.1](../../docs/design.md#51-key) states a key's *"intrinsic identity is its fingerprint...
two files with the same fingerprint are the same key,"* but does not say what identity **means**
in the derivation-gap case ([T1](decision-T1.md)'s third row), where no fingerprint can be computed at all.
Code has to answer this: what makes two files "the same key" when that cannot be proven?

### Decision

```go
type KeyIdentity interface{ identityKey() string }

type byFingerprint Fingerprint          // Fingerprint = ssh.FingerprintSHA256 output
func (f byFingerprint) identityKey() string { return "fp:" + string(f) }

type byPath string                      // resolved (symlink-followed), absolute path
func (p byPath) identityKey() string    { return "path:" + string(p) }
```

**Identity is the fingerprint when derivable; otherwise it is the resolved canonical path.**
Two files with the same fingerprint are one key with two locations, exactly as design.md states.
A key identified by path (the undecidable case) can **never** be deduplicated against another
path-identified key, even one that a human strongly suspects is a copy of the same secret — hasp
has no way to prove it, and P1 forbids reporting a fact hasp cannot verify.

### Rationale

[T1](decision-T1.md)'s derivation-gap table row 3 (legacy PEM, encrypted, no `.pub`) is the concrete case
this answers: seven of the thirteen keys in the design doc's own motivating inventory
([`design.md` §2](../../docs/design.md#2-the-problem)) are PEM format. [D12](decision-D12.md) requires
this be a pure function of what is readable **this run** — no cross-scan memory of "I previously
decided these two paths are the same key" — so path-based identity being recomputed fresh every
scan, and sensitive to `mv`/symlink changes exactly as expected, is not a limitation of this rule
but a direct consequence of P1 correctly applied.

### Consequence

- `check` reports two same-format, same-size, undecidable files as a distinct finding category —
  *"possible duplicate, cannot confirm"* — never silently merged and never silently treated as
  unrelated. This sits beside, but is not the same finding as, a *confirmed* duplicate (two
  fingerprint-identified files sharing a fingerprint).
- Renaming an undecidable key's file changes its path-based identity by definition — this is a
  sharp edge worth flagging explicitly against J6's promise (*"rename a key without touching its
  material"*): rename must be implemented as a `MoveFile` change addressed by the key's stable
  `KeyName` handle, never by re-deriving identity after the move and hoping it still matches.
  `KeyName` is the stable handle across a rename; `KeyIdentity` is expected, correctly, to change
  for the `byPath` case as a direct consequence of the move — not a bug to guard against.
