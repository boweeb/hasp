
<!-- BACKLOG.MD GUIDELINES START -->
<!-- backlog.md-instructions-version: 1.55.3 -->
<CRITICAL_INSTRUCTION>

## Backlog.md Workflow

This project uses Backlog.md for task and project management.

**At the beginning of each conversation in this project, run `backlog instructions overview` before answering or taking action. Re-read it only if you have not read it yet in the current conversation.**

Use the overview to decide whether to search, read, create, or update Backlog tasks.

Before task lifecycle actions, read the matching detailed guide:
- `backlog instructions task-creation` before creating or splitting tasks
- `backlog instructions task-execution` before planning, changing status or assignee, adding a plan or implementation notes, or implementing task work
- `backlog instructions task-finalization` before checking acceptance criteria, writing final summaries, or moving tasks to terminal statuses

Use `backlog <command> --help` before running unfamiliar commands. Help shows options, fields, and examples.

Do not edit Backlog task, draft, document, decision, or milestone markdown files directly. Use the `backlog` CLI so metadata, relationships, and history stay consistent.

</CRITICAL_INSTRUCTION>
<!-- BACKLOG.MD GUIDELINES END -->

## hasp conventions on top of Backlog.md

These notes override the generic guidance above where they differ.

### Decisions are hand-authored files

- The decision logs are one file per entry: `backlog/decisions/decision-D<n>.md` (domain, `D`) and `backlog/decisions/decision-T<n>.md` (technical, `T`). `D` and `T` are separate ID namespaces ([T56](backlog/decisions/decision-T56.md)).
- **Never run `backlog decision create`.** It mints `decision-1`, `decision-2`, and so on, a third namespace that belongs to neither log. Hand-author the next file instead, copying the front matter shape (`id`, `title`, `date`, `status`) of its neighbors and using the next unused number. This is the one exception to "do not edit Backlog markdown files directly". `backlog decision edit <id> --status ...` is fine for an existing entry's status.
- The logs are append-only. Never edit an entry to say something different; amend or supersede it with a later entry that names it. IDs are permanent. Every entry has a `### Consequence` heading and cites the principle, decision, or journey it serves. The full rules are in [docs/decision-log.md](docs/decision-log.md) and [docs/tech-decision-log.md](docs/tech-decision-log.md).
- Quote `design.md` and `tdd.md` verbatim. A paraphrase presented as a quotation is treated as a real defect.

### Documentation checks

- After editing anything under `docs/` or `backlog/`, run `go run mage.go docs`. It runs `tools/docscheck` (links, `Dn`/`Tn` citations, decision-file invariants, verbatim quotations) and the generated-reference staleness check.
- `go run mage.go ci` is the build contract; CI invokes only that.
- `docs/roadmap.md` holds sequencing, scope, and risks. Milestones, work breakdowns, and exit criteria are backlog tasks. Never renumber the roadmap's sections: Go comments cite `roadmap.md §2` through `§5`.
- Project documents under `docs/` carry YAML front matter. Backlog milestone files have no status in their format and are exempt, as are the root `README.md`, `CLAUDE.md`, and `AGENTS.md`.

### Working with the backlog here

- Use the `backlog` CLI for tasks, drafts, and milestones. CLI writes add `updated_date` and rewrite inline YAML lists as block lists; that diff is expected.
- Work bigger than one task is an **effort**: a parent task labeled `effort`, with subtasks. Do not call it a "plan" in repo files; that word belongs to Claude Code's plan mode.
- The M4 (Identity) tasks are drafts and are gated behind v1.0.0 ([D8](backlog/decisions/decision-D8.md)). `backlog search` excludes drafts; use `backlog draft list`.
- Tasks labeled `uat` quote findings from a UAT log that lives outside this repository (`go-hasp.support/UAT.md`); finding IDs of the form `U<n>.F<m>` are not F-log references.
- Completed history (M0 through M3.6) is backfilled as Done tasks labeled `backfill`; leave them on the board.
- Never run `backlog completion install`.
