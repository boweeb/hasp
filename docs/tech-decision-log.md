---
Status: APPROVED
DateCreated: 2026-08-28
DateApproved: 2026-08-29
DateLastReviewed: 2026-11-25
Related:
  - "[`docs/README.md`](README.md)"
  - "[`docs/design.md`](design.md)"
  - "[`docs/decision-log.md`](decision-log.md)"
  - "[`docs/tdd.md`](tdd.md)"
  - "[`docs/roadmap.md`](roadmap.md)"
---

# hasp — Technical Decision Log

**Purpose.** The permanent record of *technical* design questions — language, storage
mechanics, parsing strategy, dependencies, distribution — settled downstream of
[`docs/design.md`](design.md). That document is upstream and says nothing about implementation
by design; this log is where implementation questions get settled, the same way
[`docs/decision-log.md`](decision-log.md) settles domain questions. [`docs/tdd.md`](tdd.md) is
the *current* statement of the technical design; the log is *why* it says that, and what was
rejected.

**Where the entries live.** One file per entry, `T1` through the latest, at
`backlog/decisions/decision-T<n>.md` — for example
[`decision-T1.md`](../backlog/decisions/decision-T1.md). Each file's front matter carries the
`id`, `title`, `date`, and `status`; the body is the entry. Browse them with
`backlog decision list --plain` (or `backlog decision list`); that listing replaces the Index
table this file used to carry, and it is the authoritative list of which status relation clauses
are actually in use. This file keeps the rules and the living roll-up below, and nothing else.

**How to use the log.**

- Entries are append-only. A decision is never edited to say something different — it is
  superseded or amended by a **later** entry that names it.
- IDs are permanent and stable. `T3` means `T3` forever. `T` is a separate namespace from `D` —
  no `T`-ID collides with a `D`-ID, and neither log amends the other's numbering.
- New entries are **hand-authored** as `backlog/decisions/decision-T<next>.md`, with the next
  unused number. Never run `backlog decision create`: it mints `decision-1`, a third ID namespace
  that belongs to neither log ([T56](../backlog/decisions/decision-T56.md)).
- `docs/tdd.md` is the current statement of the technical design and always reflects every
  accepted entry here. When the two disagree, this log is the history and the TDD is the truth.
- An entry needs a **Consequence** to be complete (a `### Consequence` heading in the file). A
  decision whose cost nobody wrote down is a decision nobody actually made — the rule
  `docs/decision-log.md` established, kept here without modification.
- Every entry cites the principle (P1–P10), decision (a `D` entry), or journey (J1–J10) it serves.
  This project settles arguments by appeal to `docs/design.md`; a decision with no citation is
  a decision that gets re-litigated.

**Status values:** `Accepted` · `Open` · `Rejected` — optionally followed by one or more
relation clauses naming another entry: `amends`/`amended by`, `extends`/`extended by`,
`refines`/`refined by`, `supersedes`/`superseded by`, `ratified upstream as` (a `D`-log entry),
`staged by`, `gap closed by`. The three base values are closed; relation clauses are open-ended
and grow as entries need them.

---

## Still open

**Nothing.** Every entry is accepted.

`T27` through `T30` came out of the second review pass rather than the first drafting of
[`docs/tdd.md`](tdd.md), and that is worth recording as a fact about the process rather than a
defect in it. The first pass ran with `docs/design.md` frozen, so gaps it found could only be
**named**, not closed. Naming them was correct. Closing them required the freeze to lift — which
is also how [D16](../backlog/decisions/decision-D16.md), [D17](../backlog/decisions/decision-D17.md) and
[D18](../backlog/decisions/decision-D18.md) came to exist.

**`T35` through `T39` are the same story a second time**, for the investigation capability rather
than the settings file or the passphrase prompt: [D19](../backlog/decisions/decision-D19.md)–[D22](../backlog/decisions/decision-D22.md)
lifted the freeze `docs/design.md` had been under since the first reassessment, and these five
entries are what a technical design does with room newly opened upstream.

**One item in `tdd.md` §15 remains genuinely open, and it is open by choice rather than
omission**, as of the reassessment recorded in [T31](../backlog/decisions/decision-T31.md)–[T40](../backlog/decisions/decision-T40.md):

- **Multi-directory / non-default key locations**, deferred by `docs/design.md` §10 and not
  designed here. `--key-dir` is threaded explicitly through every layer rather than defaulted
  anywhere inside `internal/app` or `internal/domain`, so supporting more than one is additive
  whenever a case for it actually arrives.

**Two items that stood here alongside it are now closed, not merely narrowed.**

**The public-origin / module-path question** ([T31](../backlog/decisions/decision-T31.md), [T33](../backlog/decisions/decision-T33.md)) is closed by
[T40](../backlog/decisions/decision-T40.md): the author moved `origin` to `git@github.com:boweeb/hasp.git`, which settles
`go.mod`'s path before v1.0.0 freezes it and unblocks every deferred distribution channel at
once — exactly as [T33](../backlog/decisions/decision-T33.md) predicted a single answer would.

**SSH key inspection detail** — what `list key` and `show key` should report beyond the fact set
`tdd.md` §9's grid already named — was open because it awaited the user's own specification of
which candidate additions mattered. The investigation capability answered it from an unexpected
direction: rather than adding fields to the plain read, [T35](../backlog/decisions/decision-T35.md)–[T39](../backlog/decisions/decision-T39.md) add a
confidence-graded `--investigate` mode that reports fingerprint scheme, origin, and agent- or
passphrase-derived facts — a superset of every candidate the input-needed block in
[`roadmap.md` §5.5](roadmap.md#55-m35--hardening) once named — while leaving `list key`'s plain
output untouched, satisfying P7 exactly as the closed item's own bound required.
