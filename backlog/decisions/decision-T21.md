---
id: decision-T21
title: '`show profile` aggregates descendants by default, `--no-recurse` escape'
date: '2026-08-28'
status: 'Accepted'
---
### Context

[D1](decision-D1.md) makes profile names hierarchical paths — `work.foobarco` is a profile and
`work` is its parent and also a profile — and explicitly retires "profile group" in favour of "a
profile with children." `show profile` originally listed child profiles as one more field
alongside keys and hosts, which reads as enumeration rather than aggregation.

That is too weak for [J8](../../docs/design.md#7-journeys), whose whole framing is *"show me everything
scoped to that profile — every key, every host that depends on it — so I know what to revoke and
what will break."* Offboarding `work` while `work.foobarco` and `work.acme` hold the actual keys
would answer with an almost-empty screen, and the user would have to walk each child and union the
results by hand — precisely what [P7](../../docs/design.md#4-principles) says the tool exists to prevent.

### Decision

`show profile <name>` **aggregates the full subtree by default**: keys and hosts from the named
profile and every descendant, with each row attributed to the profile it actually came from.
`--no-recurse` restricts the view to the named profile alone.

### Rationale

The default follows P7 directly. The question a human asks about `work` on their last day is
about everything under `work`, and a tool that answers a narrower question than the one asked has
failed at comprehension even if every fact it prints is true.

Attribution-per-row is what keeps this honest under [D2](decision-D2.md) and
[T19](decision-T19.md): a key reachable from both `work.foobarco` and `personal` appears in the `work` subtree
*and* is visibly also `personal`, which is the distinction that decides whether it gets revoked.
Aggregating without attribution would produce a correct list that supports an incorrect decision.

### Consequence

- `--no-recurse` exists because "what is *directly* in this profile" is a real question when
  deciding whether a parent is a profile in its own right or merely a container
  ([D13](decision-D13.md)'s `.hasp` distinction).
- The JSON envelope carries the owning profile on every aggregated row, so a machine consumer can
  regroup without re-querying.
- A container directory with no `.hasp` marker is not a profile and cannot be the *subject* of
  `show profile`, but its descendants still aggregate under the nearest marked ancestor.
