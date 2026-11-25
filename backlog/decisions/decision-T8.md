---
id: decision-T8
title: 'Backups live in `~/.ssh/.hasp-backups/`, timestamped, never pruned by hasp'
date: '2026-08-28'
status: 'Accepted'
---
### Context

P4: *"Before hasp modifies anything it did not create, the prior state is preserved... the user
should never need to have thought ahead."* D6's preview → confirm → back up → write cycle
requires a concrete location and mechanism, and [§5.10](../../docs/design.md#510-what-is-deliberately-absent)
is explicit that hasp is not to grow a state file or database in the process of providing it.

### Decision

Backups live at `~/.ssh/.hasp-backups/`, as timestamped copies —
`.hasp-backups/config.20260828T140501Z` — inheriting the key directory's own `0700` mode,
restorable with a plain `mv`, and excluded from the profile scanner via the same
dotfile/marker-exclusion rule that already skips `.hasp` markers themselves (this is a reused
exclusion, not a new one).

### Rationale

P4 requires the backup to be automatic on **every** qualifying write, not opt-in — a rolling
backup directory inside the same tree the write touches satisfies that with no separate
subsystem. It sits **inside** `~/.ssh`, deliberately, rather than beside settings
([T7](decision-T7.md))'s `os.UserConfigDir()` location: a backup is a copy of *this directory's own
content* and belongs next to it, restorable with the same tools — `ls`, `mv`, `cp` — a human
already has for everything else here (P6).

### Consequence

This precisely completes hasp's write-surface invariant, now stated exactly:
**hasp writes to exactly one directory tree — the key directory, including its
`.hasp-backups/` subtree — and reads at most one optional settings file it never writes.** This
is the literal assertion the guard test in
[`tdd.md` §12](../../docs/tdd.md#12-testing-strategy--a-function-of-a-directory) checks: no file is ever
created, modified, or deleted outside that one tree, for any operation, in any test run.

A backed-up key file is itself key material under D4's canon — hasp therefore never prunes,
rotates, or garbage-collects `.hasp-backups/` automatically. Retention is the user's own act,
with their own tools (`rm`), matching D4's philosophy exactly: hasp's job is to never be the
reason a copy of a secret disappears, not to decide when enough copies exist.
