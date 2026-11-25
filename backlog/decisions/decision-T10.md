---
id: decision-T10
title: 'Metadata format: sentinel-prefixed TOML fragments as comments'
date: '2026-08-28'
status: 'Accepted — amended by T25'
---
### Context

[D7](decision-D7.md) elaboration 3 gives hasp a general mechanism: *"anything hasp needs to
record that the format cannot express natively is written into its own region in comment form...
metadata may record what the machine cannot tell you; it may never cache what the machine can."*
This closes [`design.md` §10](../../docs/design.md#10-explicitly-deferred)'s "the metadata format" item —
load-bearing, and newly created by D7 — but the format itself was never specified.

### Decision

**Sentinel-prefixed TOML.** Inside a hasp-owned marked region, a metadata line has the exact
shape:

```text
#:hasp <key> = <toml-value>
```

Example, inside a marked region hasp owns wholesale (an explicit host group,
[D9](decision-D9.md)):

```text
# >>> hasp:managed >>>
Host foobarco-prod
    #:hasp created_by = "new host"
    HostName prod.foobarco.internal
    User jesse
    IdentityFile ~/.ssh/work/foobarco/id_ed25519
# <<< hasp:managed <<<
```

Stripping the `#:hasp` sentinel from a metadata line yields a syntactically valid single-line
TOML fragment (`key = value`) — a metadata reader concatenates every stripped line in a region
and feeds the result to an ordinary TOML decoder ([T7](decision-T7.md)'s dependency, reused, not
duplicated). Non-sentinel `#` comments inside a managed region (section banners hasp itself
writes, for instance) are ordinary trivia and are never parsed as data — **sentinel presence,
not comment syntax, is what marks a line as metadata**, mirroring D15's presence-not-contents
discipline for the `.hasp` marker exactly, rather than inventing a second convention beside it.

### Rationale

The D7 line is enforced by construction, not by review discipline: a fingerprint is derivable
([T1](decision-T1.md)) and therefore may never appear on the right-hand side of a `#:hasp` line — writing
one would be a D12 violation the same way it would be in the marker file D15 governs. A profile
assignment, by contrast, would be legitimate content for this channel in principle — except that
[T5](decision-T5.md) already derives host-profile membership from key location, which **removes the single
largest anticipated consumer** of this mechanism before it ever needed a value.

### Consequence

At the milestone scope this document covers (through M3), **hasp writes no field through this
channel.** That is deliberate, not an oversight: the mechanism exists because D7 requires it be
available, and its keyset is empty because T5 closed the one gap it was invented to fill. Any
future proposal to populate a field here must independently clear the same P9 bar every taxonomy
proposal must — *"show that the existing structure genuinely cannot carry the fact"* — the same
test [`design.md` §5.7](../../docs/design.md#57-intent--the-category-that-closed) already stands ready to
apply.

Because the grammar is "TOML after stripping a fixed string prefix," no bespoke parser is needed
beyond a string match plus the existing TOML decoder — no new dependency, and no new grammar for
[T2](decision-T2.md)'s fuzz test to cover beyond the surrounding region structure it already fuzzes. A
future field costs one more `#:hasp key = value` line; no schema migration and no version field
are needed while the keyset is empty, and `#:hasp _schema = N` remains available, at zero present
cost, if a real one is ever needed.

**[T25](decision-T25.md) gives this line its own place in the CST** — a `MetadataLine` node, recognized by
the sentinel at parse time rather than left inside an opaque `Line` for a metadata reader to
re-scan. The keyset staying empty through M3 is unaffected; this is a representation fix, not a
new field.
