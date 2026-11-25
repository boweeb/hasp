---
id: decision-D18
title: '`release` never deletes content, for hosts as for profiles'
date: '2026-08-29'
status: 'Accepted · Extends: D15'
---
**Opened by:** [`docs/tdd.md`](../../docs/tdd.md) §9

### Context

[D15](decision-D15.md) gave `release` a requirement: *"`release` must show the file's contents in its
preview, so that removing a marker the user has written in is never a silent loss."* It states
that for **profiles**, because the `.hasp` marker is what D15 was about.

`release host` has no such rule, and needs one more than profiles do. A profile marker holds a
note the user may have written. A host stanza holds *the user's working SSH configuration* — the
hostname, the port, the proxy command, the thing that makes a connection succeed. `adopt host`
wraps an existing hand-written stanza in hasp's markers ([D7](decision-D7.md), [D14](decision-D14.md)); if
`release host` simply removed what sits inside them, adoption would be a trap — hand hasp a
working stanza, get an empty region back.

The technical design proposed the obvious answer and then flagged that it had no authority to
make it: this document says the rule for profiles and not for hosts. That is a gap in the
ratified design, so it gets an entry rather than an extrapolation.

### Decision

**`release host` moves the stanza out of hasp's marked region and re-inserts it as plain text
immediately after that region. Content is never deleted, only unmanaged.**

Stated once, generally, so it covers the three nouns and any noun added later:

> **`release` is content-preserving.** It withdraws hasp's authority over a resource. It never
> destroys what the resource contains.

### Rationale

This is [D14](decision-D14.md)'s own argument for why `release` exists, followed to its end: *"`adopt` moves
files and writes markers. Without an inverse it is a **one-way door**, and a one-way door turns
"try hasp on my real `~/.ssh`" from an experiment into a commitment."* An inverse that hands back
an empty region instead of the stanza is not an inverse — it is a one-way door with an extra step.

It is also [D4](decision-D4.md)'s spirit at a different altitude. D4's letter is about key material, and a
host stanza is not irreplaceable the way a private key is. But *hasp has no operation that
destroys what the user cannot get back* is the reason D4 reads as it does, and a hand-written
`ProxyCommand` that took an afternoon to get right sits close enough to that line to be treated
the same way. Backups (P4) would technically recover it — but requiring someone to go digging
through a backup directory to undo an operation named `release` is not a safety property, it is
an apology.

### Consequence

- **`release` is now symmetric across all three nouns**, and the symmetry is worth naming,
  because it is what makes adoption reversible in practice rather than in principle:

  | Noun | `adopt` does | `release` does |
  | --- | --- | --- |
  | **key** | Moves the file into a managed profile directory, leaving a top-level alias | Moves the file back out |
  | **host** | Wraps the stanza in hasp's markers | Re-inserts the stanza as plain text outside them |
  | **profile** | Writes a `.hasp` marker with a `#` header | Removes the marker, showing its contents first |

- **The re-inserted stanza is unmanaged the instant it lands**, so [P2](decision-D7.md) protects it again:
  outside hasp's markers it is human territory, preserved byte for byte from that point on.
- **Rigid regeneration ([D7](decision-D7.md)) is untouched.** Content inside a marked region is still hasp's
  to rewrite freely. `release` is the act of moving content *out* of the region, not an exception
  to what happens while it is in there.
- A future first-class noun inherits the general rule above, and must say how it satisfies it.
