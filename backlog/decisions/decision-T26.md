---
id: decision-T26
title: '`WriteRegion` previews carry a real diff; key material is never diffed'
date: '2026-08-28'
status: 'Accepted — amends T4'
---
### Context

[T4](decision-T4.md) gave `Change` a `Describe() string` returning *"one-line preview text."* That cannot
serve [D6](decision-D6.md) for the change kind that most needs it. `WriteRegion` regenerates
a whole region body — reordered `Include` lines, host stanzas, comments — and a one-line summary
("rewrite host group `personal.sshconfig`") tells the user nothing about whether a stanza moved or
vanished.

[P5](../../docs/design.md#4-principles) requires an operation to *"describe exactly what it would change"*,
and calls previewability a constraint on how change is modelled rather than a flag. A `Change`
that carries only the new `Body`, with no access to the current bytes, structurally cannot
produce that description.

### Decision

`Change` returns a `Preview{ Summary string; Diff []DiffLine }`. `WriteRegion` computes a real
line diff of the region's current bytes against its regenerated body. Kinds with nothing to
compare — `MoveFile`, `CreateSymlink`, `Remove` — return `Diff == nil` and render as a summary
line only.

**`WriteKeyFile` returns `Diff == nil` unconditionally**, even though it has bytes and they are
new. Private key material is never rendered, to any renderer, in any mode.

### Rationale

D6 buys its safety from the user *reading* the preview. A preview that summarizes rather than
shows moves the burden of noticing a mistake back onto the person with the least information at
the moment they confirm — the same reasoning D6 itself used to reject opt-in previewing.

The `WriteKeyFile` exception is not a gap in that argument but an application of a stronger rule.
[P3](../../docs/design.md#4-principles) keeps hasp out of custody of secrets; printing a freshly generated
private key to a terminal, a pipe, or a JSON consumer would put it into scrollback, logs, and
shell history in one step. The user does not need to see the bytes to judge the operation — the
summary names the path, algorithm and passphrase mode, which is the whole decision.

### Consequence

- `Diff` is a structured value, not preformatted text, so the human renderer can colorize it and
  the JSON renderer can marshal it — the same `Preview` serves both, and §10's envelope omits
  `diff` entirely rather than emitting an empty array when there is nothing to show.
- Computing the diff means `Plan()` reads the target file's current bytes. Reads are always safe
  ([P5](../../docs/design.md#4-principles)), so this costs nothing in guarantees — but it does mean a `Plan`
  is computed against a snapshot, and a file changed between preview and confirm is a race the
  `Applier` must detect rather than assume away.
- A guard test asserts no renderer path can reach `WriteKeyFile`'s contents: the field is not
  exported into `Preview` at all, so the boundary is enforced by absence, as in [T1](decision-T1.md).
