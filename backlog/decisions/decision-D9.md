---
id: decision-D9
title: 'Host groups exist, and a host group is an SSH config file'
date: '2026-08-26'
status: 'Accepted · Amends: D1'
---
### Context

[D1](decision-D1.md)'s original consequence retired "host group" along with "key group" and "profile
group," on the reasoning that profiles are the one grouping concept. On review this was wrong —
not because profiles are the wrong organizing concept, but because "host group" was being read
as *taxonomy* when its natural meaning here is *container*.

### Decision

A **host group is a physical container for host entries: an SSH config file.**

- The default host group is `~/.ssh/config`.
- A custom host group named `foo` is `~/.ssh/foo.sshconfig` — the file is named literally after
  the group.
- The default group is the sole exception to that naming rule.

### Rationale

SSH configuration is already multi-file in practice; the format supports composition natively.
Modelling that reality directly gives hasp a materialization strategy that is legible from
outside the tool: you can see the grouping by listing a directory, and any other tool — or a
human with an editor — reads exactly the same structure hasp does. It requires no invented
metadata, and it keeps hasp's records a projection of the world rather than a claim about it.

It also isolates blast radius. A group hasp authored is a file hasp authored, which is a much
easier thing to reason about than a region inside a file someone else wrote.

### Relationship to profiles

Host groups and profiles are **different axes and must not be conflated**:

- A **profile** is *who you are being* — persona, organizing taxonomy, spans keys and hosts and
  eventually Git/GPG identity.
- A **host group** is *where a host entry physically lives* — a file on disk.

They will often correlate, and a sensible default may be to materialize a profile's hosts into
a group file of the same name. They remain distinct concepts, and nothing requires them to
align.

### Consequence

- [D1](decision-D1.md)'s consequence is amended: "host group" returns to the vocabulary with this specific
  meaning. "Key group" and "profile group" stay retired.
- hasp must model *where a host lives* in addition to *what it is* and *who it belongs to*.
- Materializing a group means creating and owning a file; hasp's write model must account for
  files it fully authored as a distinct case from files it co-owns. This bears directly on
  [D7](decision-D7.md), which should be settled with this in hand.
- The default group is a file hasp did not create and probably should not claim, which is the
  hardest case and the one D7 must answer.
