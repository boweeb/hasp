---
id: decision-T25
title: '`MetadataLine` is its own CST node type for `#:hasp` lines'
date: '2026-08-28'
status: 'Accepted — amends T2, T10'
---
### Context

[T10](decision-T10.md) defines the metadata channel as `#:hasp <key> = <toml-value>` comment lines inside a
hasp-owned region. The CST had no node for them, which implied reading them by re-scanning raw
`Line` bytes for a string prefix. That is workable for reading and wrong for writing: rigid
regeneration ([D7](decision-D7.md)) rebuilds a region from structure, and a metadata line that
exists only as an unrecognized `Line` has no place to be *put back* — no way to say which
`HostBlock` it belongs under, or that it must precede the directives it annotates.

### Decision

`MetadataLine{ Key string; Value string; Trivia []byte }` is a node type, recognized by the
`#:hasp` sentinel (including its trailing space), valid **only inside a hasp-owned region**. Outside one it is an ordinary
comment `Line` and is never interpreted — the sentinel confers no meaning on territory hasp does
not own.

### Rationale

Giving it a node type now, while [T10](decision-T10.md)'s keyset is still empty, costs one small type and
removes the need to retrofit the CST at the moment a real field first lands — which is exactly
when the pressure to cut a corner will be highest.

The inside-a-region-only restriction is [P2](../../docs/design.md#4-principles) applied literally: a `#:hasp`
line a human typed into their own config is their comment, and hasp reading it as data would be
hasp claiming territory by string match. The marker defines ownership
([D7](decision-D7.md)); the sentinel only labels a line within it.

### Consequence

- [T10](decision-T10.md)'s "the keyset is empty through M3" is unchanged. This entry gives the mechanism a
  place to live; it does not put anything in it, and the P9 bar for adding a field stands.
- Round-tripping is unaffected: a `MetadataLine` renders from its retained bytes like every other
  node ([T17](decision-T17.md)), so the fuzz contract holds whether or not the sentinel is present.
