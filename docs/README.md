---
Status: APPROVED
DateCreated: 2026-08-29
DateLastReviewed: 2026-11-25
---

# hasp — documentation

hasp manages **identity**, not secrets: an accurate, queryable picture of the SSH keys, hosts and
personas on one laptop, and safe ways to rearrange them. Start with
[`design.md`](design.md) §1–§2 for what that means and why it exists.

This project is documentation-first. Nothing here describes code that has been written yet — the
Go implementation begins at [`roadmap.md`](roadmap.md) §2 (M0). That is deliberate: the
predecessor implementation died mid-refactor because its design decisions were never written down,
and the whole point of this set is that a settled argument stays settled.

---

## The documents

| Document | Is the truth for | Status |
| --- | --- | --- |
| [`design.md`](design.md) | **What hasp is and must be true of.** Principles (P1–P10), the domain model, capabilities, journeys (J1–J10), milestones (M1–M4, plus M3.5/M3.6). Says nothing about language, libraries, or mechanism, on purpose | Approved |
| [`decision-log.md`](decision-log.md) | **Why the design says that.** The rules of the `D` log; the entries themselves — the reasoning, the alternatives rejected, and what each decision commits the project to — are one file each under [`backlog/decisions/`](../backlog/decisions/) | Approved |
| [`tdd.md`](tdd.md) | **How a Go program satisfies all of it.** Stack, architecture, the `Plan` model, derivation pipeline, config parsing, command surface, safety stances, testing, versioning, CI/CD, and investigation — schemes, confidence, gated derivation (§18) | Approved |
| [`tech-decision-log.md`](tech-decision-log.md) | **Why the technical design says that.** The rules of the `T` log, same as the D-log's, separate namespace; the entries are one file each under [`backlog/decisions/`](../backlog/decisions/) | Approved |
| [`roadmap.md`](roadmap.md) | **What gets built in what order, and why.** Sequencing, scope boundaries, and risks; deliberately no dates or estimates. Milestones, work breakdowns, and exit criteria live in the backlog | Approved |
| [`project-assessment-2026-08.md`](project-assessment-2026-08.md) | **The Python predecessor, as found in August 2026.** Evidence, not a plan — its roadmap is superseded | Historical |
| [`user-guide.md`](user-guide.md) | **How to use hasp, by journey (J1–J8, J10).** Real commands and example invocations for each journey `design.md` §7 defines, pointing at the generated CLI reference for exhaustive flag syntax | New |
| [`on-disk.md`](on-disk.md) | **What hasp actually leaves on disk.** `.hasp` markers, in-file marked regions, the settings file, `~/.ssh/.hasp-backups/`, and the full withdrawal path | New |

## Reading order

**To understand the project:** `design.md`, then `roadmap.md`. That is the whole picture at the
level most questions live at.

**To implement it:** `design.md` → `tdd.md` → `roadmap.md` §2, and read the log entry behind
anything that looks arbitrary. It usually is not. `tdd.md` §16–§17 (versioning, CI/CD) read the
same way as every earlier section — implementation detail, not a plan — even though they arrived
later than the rest.

**A second audience now has documentation.** M3.5 ([`roadmap.md`
§5.5](roadmap.md#55-m35--hardening)) added the hand-written half of that split
([T34](../backlog/decisions/decision-T34.md)): a root `README.md` aimed at orientation and install, a
[`user-guide.md`](user-guide.md) organized around J1–J8, an [`on-disk.md`](on-disk.md) on what
hasp actually leaves behind, and a root `SECURITY.md` restating the project's existing custody and
access guarantees. M3.6 ([`roadmap.md` §5.6](roadmap.md#56-m36--investigation)) added a J10 section
to that same guide once `--investigate` gave the journey commands to document; J9 stays out of it,
still gated behind M4 ([D8](../backlog/decisions/decision-D8.md)). Both new documents under `docs/` are indexed in
the table above, alongside the generated reference under `docs/cli/` that this second audience also
relies on for exact flag syntax.

**To argue with it:** find the `Dn` or `Tn` that settled the question. If none did, the design has
a gap and it gets amended rather than reinterpreted — which is how `D16`, `D17` and `D18` came to
exist.

---

## How the two logs work

Both follow the same rules, and the rules are the reason they are worth keeping. The entries live one per file at `backlog/decisions/decision-D<n>.md` and `decision-T<n>.md` ([T56](../backlog/decisions/decision-T56.md)); list them with `backlog decision list --plain`. The two documents here, [`decision-log.md`](decision-log.md) and [`tech-decision-log.md`](tech-decision-log.md), keep the rules and the living "Still open" roll-up.

- **Append-only.** An entry is never edited to say something different. It is superseded or
  amended by a *later* entry that names it, so the wrong turn stays visible.
- **IDs are permanent.** `D3` means `D3` forever — in code comments, in commit messages, in
  conversation. `D` and `T` are separate namespaces; no ID collides, and neither log renumbers the
  other.
- **New entries are hand-authored files**, named for the next unused number in their namespace.
  Never `backlog decision create`: it mints a third ID namespace.
- **A **Consequence** section is mandatory.** A decision whose cost nobody wrote down is a
  decision nobody actually made.
- **Every entry cites what it serves** — a principle, a decision, or a journey. This project
  settles arguments by appeal to `design.md`, so an entry with no citation is one that gets
  re-litigated. A paraphrase presented as a quotation is treated as a real defect.
- **The doc is the truth; the log is the history.** When they disagree, `design.md` (or `tdd.md`)
  is what is currently true and the log records how it got there.

**ID order is not chronological.** [`T27`](../backlog/decisions/decision-T27.md) — the choice of Go — was
decided before `T1`, and numbered last, because it was written down late. An append-only log
numbers by when an entry was *recorded*, not by when the decision was *made*.

## Where project management lives

Two places, split by what each is for. **Sequencing, scope boundaries, and risks** live in
[`roadmap.md`](roadmap.md) and nowhere else in this documentation set. **Task-level tracking** —
milestones, work breakdowns, exit criteria, open findings — lives in [`backlog/`](../backlog/),
managed with the `backlog` CLI, and outside `docs/` by design ([T56](../backlog/decisions/decision-T56.md)).
`tdd.md` carries no estimates, no schedule, and no ordering-as-plan — its newest sections (§16
versioning, §17 CI/CD) follow the rule exactly as the oldest ones do; it references `M1`–`M4`
purely as scope anchors ("this applies from M2 onward" is a boundary of applicability, not a
commitment). Keeping the two apart is a standing constraint on this documentation set, not a
stylistic preference — a technical document that starts carrying deadlines stops being a technical
document.

After editing anything here or under `backlog/decisions/`, run `go run mage.go docs`. It checks
links and anchors, `Dn`/`Tn` citations, the decision-file invariants, and verbatim quotations
([`tdd.md` §17](tdd.md#17-continuous-integration-and-release-automation)).

## A note on this repository

A repository-root `README.md` exists — it arrived with M0 ([`roadmap.md`](roadmap.md) §2), along
with the first Go code, and was rewritten as an actual orientation document (install, a
sixty-second demo, license) as part of M3.5 ([`roadmap.md`
§5.5](roadmap.md#55-m35--hardening), [T34](../backlog/decisions/decision-T34.md)). It deliberately does not
restate milestone status as prose — that lives in the backlog (`backlog milestone list`) and in the
commit-anchored table in [`roadmap.md` §1](roadmap.md#1-how-this-document-sequences-work) — so a reader of the root
`README.md` is pointed there rather than handed a copy that can go stale on its own. A root
`SECURITY.md` arrived alongside it, restating (not inventing) the custody and access guarantees
`design.md` §3.2 and [D17](../backlog/decisions/decision-D17.md)/[D19](../backlog/decisions/decision-D19.md) already state. The
predecessor's `README.rst` is not carried into this repository — `design.md` supersedes its
"Primary Goals" and "Future Ideas" sections, and
[`project-assessment-2026-08.md`](project-assessment-2026-08.md) records what was there.
