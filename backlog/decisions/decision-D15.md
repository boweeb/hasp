---
id: decision-D15
title: 'hasp reads the marker''s presence, never its contents'
date: '2026-08-28'
status: 'Accepted · Amends: D13, D14 · Extended by: D18'
---
### Context

[D13](decision-D13.md) required the `.hasp` profile marker to stay **empty**, on the reasoning that a file
with content becomes a manifest — option (iii) readmitted through the back door, undermining
[D12](decision-D12.md).

That rule was stated at the wrong altitude. It conflated the invariant with a crude proxy for
it. Bytes sitting in a file are not declared state; bytes a *program reads* are. "Empty" banned
the wrong noun, and in doing so it foreclosed something worth keeping: the ability to leave a
note for a human, and room for configuration to grow naturally at that scope if a case ever
genuinely arises that observation cannot cover.

### Decision

**The invariant is restated:**

> **hasp reads the marker's *presence*, never its *contents*.**

Precise, testable, and it protects [D12](decision-D12.md) exactly as well as "empty" did — while permitting
what "empty" needlessly forbade.

**1. The file's contents belong to the user.** They may write whatever they like in it. hasp
never parses, validates, or reports on it.

**2. Commentary is prefixed `#`.** One syntax, not two.

**3. `adopt` creates the file with a `#` header** stating what the file is, that its *presence*
is the signal, and that `#` lines are ignored.

**4. `release` must show the file's contents in its preview**, so that removing a marker the
user has written in is never a silent loss.

### Why `#`, and only `#`

**P9 decides it.** The neighbouring file in that very directory — `ssh_config` — uses `#`, as do
shell, `gitconfig`, TOML, and YAML. `//` matches nothing in this ecosystem. The convention that
already exists is the one to ride.

Blessing two syntaxes would be a parsing decision made years early, handing a future reader an
ambiguity for no benefit. And if data ever does land in this file, TOML is its likely shape
given the project's history — `#` is already TOML's comment character, so that door opens
cleanly rather than needing a migration.

### Why elaboration 3 is the load-bearing one

`#`-prefixing only protects a future parser if commentary is *actually* prefixed — and nothing
enforces that, precisely because hasp never reads the file. A user who writes bare prose today
hands a someday-parser something indistinguishable from data.

A `#` header written at creation time converts the convention from folklore into something
visible the first time anyone opens the file. Files written today are then entirely `#` lines,
which a future reader parses as zero data. That is what makes backward compatibility real rather
than hoped for.

### Why not rename the file instead

The alternative considered was renaming the marker to convey intent — `.hasp-owned` or similar.
Rejected: a self-documenting header does the naming's job from inside the file, so the name can
stay short. `.hasp` also ages better. When [J9](decision-D11.md) arrives the same marker plausibly scopes a
Git repository, where "owned" is the wrong word for what it marks.

### The gate for "someday"

**The day hasp reads this file's contents, it reopens §5.7 of the design document.**

That section is kept named-and-empty as a standing test: anything hasp must be *told*, rather
than able to look up, requires its own decision with [P9](decision-D13.md) as the bar. Content in `.hasp`
that hasp *reads* is precisely that. The flexibility is preserved and the tripwire already
exists; D15 simply points the two at each other.

This matters because the file has the same *shape* as the original failure — the TOML state file
began as "just a debug aid for visibility" and became the state model. The difference now is
that the door is explicitly a door, with a decision required to walk through it, rather than
something that drifts open one convenient field at a time.

### Consequence

- **[D13](decision-D13.md)'s "stays empty" is superseded** by the presence/contents rule. The protection is
  unchanged; the prohibition is narrower and aimed at the right thing.
- **§5.7 stays closed today.** Commentary in the marker is human-to-human, not human-to-hasp, so
  nothing hasp must be *told* has been reintroduced.
- **`adopt` gains a requirement:** write the `#` header when creating the marker.
- **`release` gains a requirement:** preview shows the marker's contents, not just its removal.
- Forward compatibility is structural — every file hasp creates is self-describing, and every
  file written under the convention parses as zero data to any future reader.
