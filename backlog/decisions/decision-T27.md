---
id: decision-T27
title: 'Go, and a fresh implementation rather than a repair of the predecessor'
date: '2026-08-29'
status: 'Accepted'
---
**Closes:** [`design.md` §10](../../docs/design.md#10-explicitly-deferred)'s *implementation stack* item

### Context

`design.md` §10 deferred the stack completely. Before this entry closed it, the item read:
*"**Implementation stack** — language, runtime, distribution, dependencies. Nothing in this
document assumes any of them."* [`docs/tdd.md`](../../docs/tdd.md) §2 then opens with **Go 1.27.1** as a
given.

No entry in this log recorded how it got there. That is a gap of exactly the kind this project's
culture exists to catch: the largest technical decision in the project — larger than any of
`T1`–`T26`, every one of which presupposes it — carried no citation at all. Two questions were
actually decided, and neither was written down: **whether to repair the Python implementation or
replace it**, and **what to write the replacement in**.

### The prior question: repair or replace

[`docs/project-assessment-2026-08.md`](../../docs/project-assessment-2026-08.md) §6 proposed a five-phase
repair roadmap for the existing Python codebase. It was not taken — and *not* because repair
looked hard. The assessment says the opposite: *"the blockers are small and well-localised"* and
*"A focused day gets you back to a working single-store tool."*

It was not taken because **[D12](decision-D12.md) deleted most of the work the repair
consisted of.** `project-assessment-2026-08.md` §6's Phase 1 is *"Decide the store, then converge
on it."* Phase 2 is
*"Finish `sync key`"*, labelled *"this is the MVP."* D12 answers Phase 1 with *neither store*,
and dissolves Phase 2 outright in its own words: *"**`sync` dissolves.** The verb exists only to
reconcile an index against reality. With no index there is nothing to reconcile."*

So the repair's two largest phases had no work left in them once the design was ratified. What
survived from the predecessor was its **ideas**, not its code, and the assessment names which:
the verb×noun matrix — *"the good architectural idea in this codebase"*, with the instruction
*"Keep this."* — and the eleven concrete scenarios in `keys.feature`.

What remained to write was: state handling (deleted by D12), the key read path ([T1](decision-T1.md)), the
config parser ([T2](decision-T2.md) — which never existed in the predecessor; the assessment's Phase 5 lists
it as an unstarted spike), and the command surface ([D10](decision-D10.md),
[T3](decision-T3.md)). That is the entire program. **A rewrite was not chosen over a repair — the repair had
almost nothing left in it.**

### The language

Given a fresh implementation, four constraints actually discriminated:

1. **The derivation gap must close with no subprocess and no cgo.** Every read recomputes every
   fact (D12), on every invocation. The predecessor shelled out to `ssh-keygen` and linked
   `libmagic`; the assessment §4 counts both as toolchain weight, and
   [D14](decision-D14.md) records `ssh-keygen -c` failing outright on encrypted keys.
   `x/crypto/ssh` reads the whole key matrix from raw bytes in pure Go ([T1](decision-T1.md)) — the specific
   capability that made `CGO_ENABLED=0` reachable ([T9](decision-T9.md)) and removed that failure class.
2. **Distribution as a single static binary.** GoReleaser is a stated constraint on this project
   ([T9](decision-T9.md)). A tool whose job is to inspect `~/.ssh` on a machine you have just sat down at
   should not first require a runtime, a virtualenv, or a package manager.
3. **A parser hasp owns either way.** [T2](decision-T2.md) found no `ssh_config` library in any ecosystem
   offering the round-trip contract P2 requires — and the assessment's Phase 5 had already reached
   the same conclusion for Python, listing `storm`, `advanced-ssh-config`, `paramiko.SSHConfig`
   and `sshconf` as *"a genuine build-vs-buy decision and deserves its own spike."* Since the
   parser is hand-rolled in either language, the ecosystem's parser inventory stopped being a
   reason to prefer one.
4. **Static typing over a domain of small value objects.** `tdd.md` §3's domain is almost entirely
   immutable values, and its one genuinely tricky rule — `KeyIdentity` being a fingerprint *or* a
   path ([T12](decision-T12.md)) — is expressible as a sealed interface the compiler checks rather than a
   convention a reviewer enforces.

### Decision

**Go, and a fresh implementation in a new repository.** The Python codebase is not migrated, not
vendored, and not referenced by the new code. It survives as the subject of
`docs/project-assessment-2026-08.md`: a record of what was learned, including the specific failure
this entire documentation set exists to avoid repeating.

Go 1.27.1 via `mise`; targets `linux/{amd64,arm64}` and `darwin/{amd64,arm64}` ([T9](decision-T9.md)).

### Rationale

The three points above are the case for Go. The honest counterweight is smaller than expected.

**What was given up.** The `keys.feature` scenarios were executable in principle — `pytest-bdd`
was a declared dependency — but the assessment records *"zero step definitions and zero Python
test files"*, so nothing executable was actually lost. The eleven scenarios are carried forward
as journeys (`design.md` §7) and as `testscript` cases (`tdd.md` §12). Python's ecosystem
advantage for this specific domain proved narrow: `paramiko` cannot round-trip a config, and
`python-magic` solved a problem `x/crypto/ssh` does not have.

**What was explicitly not a reason: performance.** P8 is unambiguous — *"any design justified by
performance, concurrency, or scale is almost certainly solving a problem hasp does not have"* —
and tens of keys parse in milliseconds in either language. Choosing Go for speed would have meant
choosing it for the one reason this project's own principles reject.

### Consequence

- **`design.md` §10's implementation-stack item is closed**, and §10 cites this entry.
- **`T1`–`T26` all rest on this entry, and it is numbered after them.** That is an artifact of an
  append-only log, not of the decision arriving late — the choice was made before the technical
  design was written and simply never recorded. **T-ID order carries no chronology and must not
  be read as any.**
- **The predecessor repository is a historical artifact.** Nothing here depends on it, and
  `docs/project-assessment-2026-08.md` now carries a banner saying so, so that its Phase 0–5
  roadmap is not mistaken for live work.
- Every design question the assessment §5 left open is now answered in
  [`docs/decision-log.md`](../../docs/decision-log.md); that mapping is recorded in the assessment itself
  rather than duplicated here.
