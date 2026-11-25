---
id: decision-T43
title: '`adopt key`''s alias-preserving move is one atomic `Change`, not an ordered two-`Change` pair'
date: '2026-09-08'
status: 'Accepted · Amends: T20'
---
### Context

[T20](decision-T20.md) is correct about the ordering *rule* — a `Plan`'s `Changes` are ordered so any prefix
leaves a working state — but its own worked example has drifted from what shipped. T20 describes
`adopt key` inverting `[MoveFile, CreateSymlink]` into `[CreateSymlink, MoveFile]`, two ordered
`Change`s. That is not the mechanism in the codebase.

What actually ships: `AdoptKeyUseCase.Plan` (`internal/app/adoptkey.go`) builds a single
`ReplaceWithSymlink{From, To}` `Change` per file — one for the key, and a second, independent one
for its `.pub` sidecar when present, because the two paths never interact and there is no ordering
question between them. `ReplaceWithSymlink.Apply` (`internal/app/change_replacewithsymlink.go`)
delegates to `WriteFS.ReplaceWithSymlink` (`internal/app/plan.go`), whose documented contract is
copy the source's bytes to the destination, verify byte-for-byte, then atomically `os.Rename` a
freshly created symlink onto the source — one rename, one primitive, never a bare unlink of the
source.

### Decision

T20's ordering rule is unchanged and still governs any future `Change` kind that genuinely needs
two ordered steps. What is corrected here is the record of `adopt key`'s own mechanism: it is one
atomic `Change`, not two `Change`s kept in a careful order. There is no reachable intermediate
state at all — not even the "dangling symlink is inert" state T20's worked example relied on —
which is a strictly stronger guarantee than an ordered two-`Change` pair provides.

### Rationale

The two-`Change` shape was not merely reordered, it was rejected outright, in either direction.
`[MoveFile, CreateSymlink]` leaves the key briefly undiscoverable if the symlink step fails.
`[CreateSymlink, MoveFile]` — T20's own described order — is not equivalently safe, either:
`os.Rename`'s replace-on-conflict semantics mean a move landing on an existing path can destroy
the very file the symlink was standing in for, which is worse than the original problem, not a
fix for it. `docs/tdd.md` §4 ("Change ordering") and §11 (`Write mechanics`, point 4) already
carry this exact reasoning against the single atomic primitive that shipped instead. T20's worked
example simply predates that primitive landing; the append-only convention means the fix is a new
entry naming T20, not an edit to it.

### Consequence

- `docs/tdd.md` §4 and §11 already state the current mechanism correctly, so no `tdd.md` change
  and no code change accompanies this entry — it corrects `tech-decision-log.md` alone.
- `release`'s mirror operation, `WriteFS.ReplaceSymlinkWithFile` (`internal/app/plan.go`), follows
  the identical one-`Change` shape for the identical reason: a single atomic replace has no
  ordering problem to guard against in the first place.
- The general lesson for future write use cases: collapse a move-plus-alias operation into one
  atomic `Change` wherever the underlying filesystem primitive supports it, and reach for T20's
  ordered-multi-`Change` case only when it genuinely does not.
