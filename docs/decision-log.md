---
Status: APPROVED
DateCreated: 2026-08-26
DateLastReviewed: 2026-11-25
Related:
  - "[`docs/design.md`](design.md)"
  - "[`docs/tdd.md`](tdd.md)"
  - "[`docs/tech-decision-log.md`](tech-decision-log.md)"
  - "[`docs/roadmap.md`](roadmap.md)"
---

# hasp — Decision Log

**Purpose.** The permanent record of design questions that have been *settled*, and why.
`docs/design.md` describes what hasp is. The log records how it came to say that, which
alternatives were rejected, and what each decision commits the project to.

**Where the entries live.** One file per entry, `D1` through the latest, at
`backlog/decisions/decision-D<n>.md` — for example
[`decision-D1.md`](../backlog/decisions/decision-D1.md). Each file's front matter carries the
`id`, `title`, `date`, and `status`; the body is the entry. Browse them with
`backlog decision list --plain` (or `backlog decision list`); that listing replaces the Index
table this file used to carry, so there is no second list to keep in step. This file keeps the
rules and the living roll-up below, and nothing else.

**How to use the log.**

- Entries are append-only. A decision is never edited to say something different — it is
  superseded or amended by a **later** entry that names it. The wrong turn stays visible.
- IDs are permanent and stable. `D3` means `D3` forever, including in code comments and
  commit messages.
- New entries are **hand-authored** as `backlog/decisions/decision-D<next>.md`, with the next
  unused number. Never run `backlog decision create`: it mints `decision-1`, a third ID namespace
  that collides with neither log and belongs to neither. An entry's status field may gain an
  "amended by" clause when a later entry amends it; that goes through
  `backlog decision edit <id> --status ...` ([T56](../backlog/decisions/decision-T56.md)).
- `docs/design.md` is the *current* statement of the design and always reflects every
  accepted decision here. When the two disagree, this log is the history and the design doc
  is the truth.
- An entry needs a **Consequence** to be complete (a `### Consequence` heading in the file). A
  decision whose cost nobody wrote down is a decision nobody actually made.

**Status values:** `Accepted` · `Open` · `Amended by Dn` · `Superseded by Dn` · `Rejected`

---

## Still open

**Nothing.** Every entry is ratified.

[D13](../backlog/decisions/decision-D13.md) closed the last of the original fifteen by putting profile membership in the
filesystem, which makes [D12](../backlog/decisions/decision-D12.md)'s "no persisted state" literally true with no asterisk: hasp
declares nothing and derives everything. [D14](../backlog/decisions/decision-D14.md) is what made that affordable, by
distinguishing resources hasp merely reports on from resources the user has explicitly handed it.

**D16 through D18 arrived from the other direction — upstream, out of the technical design — and
that is the process working as intended.** Deriving [`docs/tdd.md`](tdd.md) from this document
surfaced two places where an honest implementation needed something the design did not license
([D16](../backlog/decisions/decision-D16.md)'s settings file, [D17](../backlog/decisions/decision-D17.md)'s passphrase prompt) and one place where the design
stated a rule for one noun that plainly belonged to all three ([D18](../backlog/decisions/decision-D18.md)). None of the three
was resolved by reinterpreting a sentence already here. Each is an amendment, which is the only
way this document is allowed to change.

**Deliberately undecided is not the same as open**, and both remaining items are named where they
live: cross-machine synchronization (`docs/design.md` §3.3) and multi-directory key locations
(§10). Neither is a question waiting on an answer; both are doors held shut on purpose.

The technical questions this document deferred are settled downstream in
[`docs/tech-decision-log.md`](tech-decision-log.md) in that log — including the two
`docs/design.md` §10 called load-bearing: the **metadata format** ([D7](../backlog/decisions/decision-D7.md)'s comment channel,
`T10`) and the **configuration parsing approach** (`T2`).

**D19 through D22 arrived from a third direction — from use.** Answering a fingerprint held in
hand from a cloud console needed a hash computed over a *private* key ([D19](../backlog/decisions/decision-D19.md)), a `find` that
spans schemes rather than spellings ([D20](../backlog/decisions/decision-D20.md)), a name for the investigative work hasp was
actually born from ([D21](../backlog/decisions/decision-D21.md)), and a way to say how sure hasp is rather than only what it found
([D22](../backlog/decisions/decision-D22.md)). None of the four came from deriving a downstream document; all four came from a
question the tool could not yet answer. [D19](../backlog/decisions/decision-D19.md) is the sharpest of them, because it retires a
stance — P3's absolute prohibition on reading an existing key's material — that was a reasonable
default at liftoff and had since been mistaken for a requirement.
