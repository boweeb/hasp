---
id: decision-T56
title: 'Backlog.md adopted: the D-log and T-log become one file per entry under backlog/decisions, and task-level tracking now exists'
date: '2026-11-25'
status: 'Accepted · Amends: roadmap.md §8'
---
### Context

Two things in this repository had outgrown their formats. The decision logs were single, very long files, each carrying a hand-maintained Index table that duplicated every entry's title and status and had to be edited in the same commit as every append. [`docs/README.md`](../../docs/README.md)'s own rules say what that costs: prose restating a log goes stale, and [`tools/docscheck`](../../tools/docscheck/main.go) was written partly to police the Index-row count against the anchor count. And [`docs/roadmap.md`](../../docs/roadmap.md) was doing two jobs at once: sequencing and scope narrative (which belong in a document) and per-milestone work breakdowns, chunk tables, and exit-criteria lists (which are task tracking, kept honest only by a commit-anchored `Status` column and the rule from [`roadmap.md`](../../docs/roadmap.md) §8 that this must never become a backlog). Meanwhile concrete open work (the pre-v1.0.0 UAT findings, the gated M4 breakdown) had no home at all outside a provisional document kept outside the repository.

[Backlog.md](https://github.com/MrLesk/Backlog.md) (`backlog` CLI) stores tasks, drafts, milestones, and decisions as plain markdown files under `backlog/`, and its `decision` records accept free-form IDs and statuses.

It serves the same end as [T34](decision-T34.md) (documentation is generated or mechanically checked wherever it can drift) and [T32](decision-T32.md) (the `Docs` Mage target is part of the build contract), and it keeps [D8](decision-D8.md)'s gate on M4 visible as drafts instead of prose. No design principle (P1 through P10) or journey is touched: this is repository process, not product behavior, and the entry says so rather than citing one it does not serve.

### Decision

**1. Backlog.md is adopted, in CLI integration mode, rooted at `backlog/`.** `backlog/config.yml` carries the project configuration. The root `CLAUDE.md` and `AGENTS.md` carry Backlog.md's own short instruction nudge plus a hasp-specific note.

**2. The D-log and the T-log move, one file per entry, with their IDs intact.** `D3` is now `backlog/decisions/decision-D3.md` and `T27` is `backlog/decisions/decision-T27.md`. The front matter holds `id`, `title`, `date`, and `status`; the body is the entry, moved verbatim, with only link targets rewritten. `docs/decision-log.md` and `docs/tech-decision-log.md` remain as short documents holding the log rules and the living "Still open" roll-up, and point at the directory. Their Index tables are dropped: `backlog decision list` is the same projection with no second copy to maintain.

**3. `D` and `T` remain two separate ID namespaces, and entries stay append-only with permanent IDs.** None of [`docs/README.md`](../../docs/README.md)'s log rules change: a mandatory Consequence, a citation of what the entry serves, and amendment by a later entry rather than edit.

**4. New entries are hand-authored files, never `backlog decision create`.** The CLI mints `decision-1`, `decision-2`, and so on: a third namespace that belongs to neither log, and numbers that would collide in meaning with `D1` and `T1`. A new entry is a new file named `decision-D<next>.md` or `decision-T<next>.md` using the next unused number in its namespace, with the same front matter as the files around it. `backlog decision edit <id> --status ...` is the sanctioned way to change an existing entry's status field.

**5. Task-level tracking now lives in `backlog/`, which amends [`roadmap.md`](../../docs/roadmap.md) §8.** §8's wording was that task-level tracking, if it ever exists, lives outside the documentation set. It now exists, and it lives outside `docs/` exactly as that sentence required. The roadmap keeps its sequencing rationale, scope narratives, risks, and §8; milestones become `backlog milestone`s, and work breakdowns and exit criteria become `backlog task`s referencing the roadmap section they came from. The roadmap's headings are unchanged because they are link targets.

**6. `tools/docscheck` is re-pointed.** Citation resolution (Check B) and the per-log invariants (Check C) read `backlog/decisions/decision-{D,T}<n>.md`: the filename matches the front-matter `id`, numbers are contiguous from 1 per namespace, `date` and `status` are present, and every entry has a `### Consequence`-prefixed heading. The index-row check is gone with the Index. Link checking (Check A) covers `docs/`, the decision files, the backlog task, draft, milestone, doc, and completed folders, and the root `*.md` files, resolving links relative to the file they appear in.

### Rationale

Per-entry files make an append a one-file change, make `git log -- backlog/decisions/decision-T27.md` the history of exactly one decision, and remove the Index table that was the main source of a log's drift. Keeping IDs intact keeps every code comment, commit message, and conversation that says `D3` or `T27` true. Hand-authoring new entries costs one file copy and avoids a namespace collision that no later rename could undo, because IDs are permanent. The tracked work needed a real tracker so that the roadmap could return to being the sequencing document it says it is, rather than carrying a work breakdown that [`docs/README.md`](../../docs/README.md) had to defend as a projection and not a backlog.

### Consequence

Costs, stated plainly:

- GitHub no longer shows a browsable Index of decisions. Listing them needs the `backlog` CLI (`backlog decision list`) or a directory listing of `backlog/decisions/`.
- Agents now depend on the `backlog` CLI. The root `CLAUDE.md` tells them to run `backlog instructions overview` at the start of every conversation.
- The third-namespace hazard (`backlog decision create` minting `decision-1`) is a standing risk. `docscheck` now enforces it by rejecting any file in `backlog/decisions` that is not `decision-D<n>.md` or `decision-T<n>.md`.
- `docscheck` Check D does not verify quotations in backlog tasks, drafts, or milestones. The verbatim UAT text quoted in tasks is unverified by design: its source lives outside the repository.
- `CLAUDE.md` and `AGENTS.md` carve a hand-authored-decisions exception out of Backlog.md's rule not to edit its markdown files directly.
- Merge hashes the old roadmap recorded for M0 through M3.5 (for example `41a7a1a`, `d009473`, `4577e17`, `9243f1a`, `d7699c6`, `957ce9b`) do not exist in this repository's history. They were replaced with tag-derived and first-parent-merge hashes, so a reader should not conclude history was lost.

Effects:

- Every link of the form `decision-log.md#dN` or `tech-decision-log.md#tN` across the documentation set, `SECURITY.md`, and the Go comments that carry a path is rewritten to the decision's own file. A bare `D3` or `T27` citation in a Go comment or commit message is unchanged.
- `docscheck`'s `Docs` Mage target still guards the logs, now against files rather than anchors. The Index-row-count check is gone with the Index.
- A decision's heading is now the file's `title`; the entry body keeps its `###` headings so that the Consequence invariant reads the same text it always did.
- Backlog milestone files carry no status, so they are exempt from the YAML-front-matter rule that applies to project documents.
- Completed milestones are backfilled as Done tasks so that `backlog milestone list` reflects history; open v1.0.0 work, UAT findings, and the gated M4 breakdown (as drafts) are tasks too.
