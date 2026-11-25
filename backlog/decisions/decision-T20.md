---
id: decision-T20
title: 'Plan ordering: any prefix leaves a working state; `adopt`''s alias-preserving move'
date: '2026-08-28'
status: 'Accepted — amends T4, T15; amended by T43'
---
### Context

[T13](decision-T13.md) accepts that a `Plan` touching several files has no cross-file atomicity, because
POSIX offers no way to rename two files as one operation, and routes the resulting inconsistency
into a `check` finding. That is honest, but incomplete: `Applier` walks `Changes` in slice order
and nothing said how the slice should be **built**.

Ordering is not cosmetic here. `adopt key` naturally plans `[MoveFile, CreateSymlink]`. If the
move succeeds and the symlink does not, the key sits in its profile directory with no top-level
alias — and `ssh` stops finding it. That is [D13](decision-D13.md)'s stated consequence
arriving as a silent failure: the user's next push fails with nothing pointing at hasp.

### Decision

**A `Plan`'s `Changes` are ordered so that any prefix of them leaves the machine in a working
state.** The least recoverable change goes last.

Applied to `adopt key`, this inverts the natural order to `[CreateSymlink, MoveFile]`: the symlink
is created first, pointing at a destination that does not yet exist. A dangling symlink is inert —
`ssh` simply does not find it, which is the pre-adopt status quo — and the subsequent move makes
it valid.

### Rationale

This converts an unavoidable weakness into a bounded one. Cross-file atomicity is not available
(T13), so the remaining lever is *which* half-applied states are reachable. Ordering by
recoverability means every reachable partial state is either the status quo or an improvement on
it, and the worst outcome is a stray artifact `check` can name rather than a silently broken
authentication path.

It also composes with [P4](../../docs/design.md#4-principles) rather than duplicating it. Backups make a
partial application *recoverable*; ordering makes it *non-damaging in the first place*. The second
is worth more, because it does not require the user to notice anything went wrong.

### Consequence

- Ordering is a property of each use case's `Plan` construction, and it needs a test per
  write verb: inject a failure at each index and assert the machine still works.
  [T15](decision-T15.md)'s injected-failure guard test for `adopt` is the template.
- A dangling symlink is now a legitimate transient state, so `check`'s dangling-alias finding must
  distinguish "left over from a failed `adopt`" from "the target was deleted" — or at minimum
  report it in terms that make the remedy obvious.
- Any future `Change` kind arrives with an ordering obligation, not just an `Apply`. The rule is
  stated in §4 so it is visible where plans are built, not only here.
