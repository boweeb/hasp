---
id: decision-T18
title: 'CST marker defects are represented as data, not parse errors'
date: '2026-08-28'
status: 'Accepted — amends T2'
---
### Context

[T2](decision-T2.md) states that parsing never fails — every byte becomes some node, which is what makes
`Render(Parse(b)) == b` total. §11's safety table then posits *"a marker or region is malformed"*
as a fail-closed guard for writes. Both cannot be true as originally written: if parsing never
errors, nothing carries the signal the guard keys off.

### Decision

Marker defects are **data on the parsed file**, not errors from parsing. `File.MarkerDefects
[]MarkerDefect` is populated during the parse, enumerating the detectable conditions: an
unmatched begin, an unmatched end, a duplicate begin for the same region id, an end preceding its
begin, and a marker nested inside a `HostBlock`.

A non-empty `MarkerDefects` makes writes **to that file** fail closed. Reads are unaffected and
report what parsed, per §11's fail-open rule for reads and [P5](../../docs/design.md#4-principles)'s
guarantee that reads are always safe.

### Rationale

Representing defects as data preserves T2's totality — the contract that a file hasp does not
understand still round-trips is what [P2](../../docs/design.md#4-principles) rests on, and an error return
would break it for exactly the damaged files most in need of being left alone.

It also puts the two stances in the right places. A damaged marker means hasp cannot know where
its own territory ends, so writing would risk destroying human bytes — fail closed. But refusing
to *report* on the file would break [J1](../../docs/design.md#7-journeys), whose promise is that surveying
any machine works. The asymmetry is deliberate and follows the shape §11 already uses everywhere.

### Consequence

- `check` gains a finding for each `MarkerDefect` kind — a damaged region is exactly the sort of
  untidiness [J5](../../docs/design.md#7-journeys) exists to surface, and the user repairs it with an editor
  ([P6](../../docs/design.md#4-principles)), since hasp will not write to that file until it is sound.
- A hand-mangled region is recoverable rather than fatal: because hasp owns the region wholesale
  ([D7](decision-D7.md)), the repair is to rewrite it from scratch once the markers are
  balanced again.
