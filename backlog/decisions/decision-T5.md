---
id: decision-T5
title: 'A host''s profile membership is derived from its key bindings'
date: '2026-08-28'
status: 'Accepted — amended by T16'
---
### Context

[D13](decision-D13.md) settled how a **key's** profile membership is expressed — the
directory it lives in — but [`design.md` §5.4](../../docs/design.md#54-host) only asserts that a host has
*"two independent affiliations... who it belongs to (a profile) and where it physically lives (a
host group)"* without saying how the first one is computed. Left unresolved, this is exactly the
kind of gap D13's own preamble warns about: something hasp would have to be *told*, reopening
the closed §5.7 "Intent" category.

### Decision

**A host's profile set is the union of the profile sets of the keys its `IdentityFile` lines
resolve to**, via the binding resolution mechanism in
[`tdd.md` §5](../../docs/tdd.md#5-derivation-pipeline--scan-classify-resolve-project). No metadata is
declared for this. A host resolves to zero or more keys (D2 permits a key to belong to many
profiles; nothing prevents a host from listing more than one `IdentityFile`), and its profile
set is exactly the union across all of them.

Because a key's own profile membership already accounts for [§5.3](../../docs/design.md#53-profile)'s
`.hasp`-marker rule (a directory is a profile only if marked), this derivation needs no separate
marker check for hosts — it reuses the already-computed key-profile-set verbatim.

**[T16](decision-T16.md) adds a second binding source.** This entry, as written, resolved only *explicit*
`IdentityFile` lines. A stanza with none is not a stanza with zero bindings — `ssh` itself still
tries its own default identity files — and T16 extends "the keys its bindings resolve to" to
cover both.

### Rationale

This keeps [D12](decision-D12.md) literally true with **no new asterisk**: nothing is
declared for hosts that the filesystem plus the config file don't already carry. It works on
**unmanaged** hosts too, since the computation only needs key *location*, not host management
state — a stanza hasp has never wrapped in markers still gets a computed profile, which is what
lets M1's read-only survey (D14) report host-profile membership before `adopt` exists at all,
strengthening J1 and J3.

The rejected alternative — declaring host→profile membership as metadata inside a marked region,
which D7's mechanism ([T10](decision-T10.md)) makes possible — was rejected on P9's own ladder: it would
duplicate a fact fully computable from structure that already exists, and P9 requires showing
existing structure genuinely *cannot* carry the fact before falling back to declaration. It can.

### Consequence

- A host with no `IdentityFile` line belongs to no profile — the same legitimate state as a
  top-level key ([§5.1](../../docs/design.md#51-key)) — and it is a `check` finding, mirroring the language
  D13 already uses for the key case.
- A host bound to a key shared across profiles (D2) inherits **every one of them**. This is
  exactly what J8 (offboard) needs — *"show me everything scoped to that profile"* — walked
  backwards from key to host along the binding relation ([§5.6](../../docs/design.md#56-binding)), and is
  the concrete reason D11 called binding-as-a-relation load-bearing.
- If a resolved `IdentityFile` target sits in a directory *without* a `.hasp` marker, it
  contributes nothing to the host's profile set — a plain unmarked directory carries no
  taxonomy, by [§5.3](../../docs/design.md#53-profile)'s own rule, and the host derivation must not invent
  an exception for itself.
