---
id: decision-T19
title: 'Alias location is the mechanism for cross-profile key membership — D2''s write path'
date: '2026-08-28'
status: 'Accepted'
---
### Context

[D2](decision-D2.md) states a key may belong to more than one profile, and calls a key shared
between `personal` and `work.foobarco` *"a real situation, not a modelling error."*
[D13](decision-D13.md) then made a profile a **directory** and membership a fact of location.
A file lives in exactly one directory. Nothing in the design says how a user is supposed to
*create* the state D2 declares legitimate.

`--add-alias` existed, but framed purely as a naming convenience for tools that hardcode `id_rsa`
([D5](decision-D5.md), §5.2) — never as the mechanism that realizes D2.

### Decision

**An alias symlink placed inside a second profile's directory is what makes a key a member of
that profile.** Membership derivation unions over **all** of a key's locations — every path at
which its material is reachable, canonical or symlinked — not only the canonical one.

`edit key --add-alias=<path>` therefore does double duty: it is both D5's naming escape hatch and
D2's write path, depending on where the alias is placed. §5 and §9 say so explicitly rather than
leaving the user to infer it.

### Rationale

This needed no new mechanism, which is the point. [P9](../../docs/design.md#4-principles) says taxonomy rides
on structures that already exist, and the symlink was already ratified as the materialization of a
name (D5). Reading membership from *all* locations rather than one is a one-line change to the
derivation and keeps [D12](decision-D12.md) literally true — nothing is declared, and the
second membership is as visible to `ls` as the first.

The alternative was a metadata field recording extra profiles, which [D7](decision-D7.md)
would have permitted. It was rejected on P9's ladder: the filesystem can carry this fact, so the
metadata channel must not.

### Consequence

- A key's `Locations` is a set, and its `Profiles` is the union over that set. Any code path that
  reasons from "the" location of a key is wrong by construction.
- `release` on one profile must not remove material still reachable from another. The safe form
  is removing the alias in the released profile, never the canonical file —
  [D4](decision-D4.md) is unaffected because no operation removes the last copy.
- [J8](../../docs/design.md#7-journeys) inherits D2's hard case exactly as [D2](decision-D2.md) predicted:
  offboarding `work.foobarco` must report a key also reachable from `personal` as *kept, not
  revoked*, and `check` must be able to tell the two apart.
