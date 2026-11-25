---
id: decision-D7
title: 'hasp owns only the regions of a file it marks as its own'
date: '2026-08-26'
status: 'Accepted (2026-08-26)'
---
**Amends:** P2 · **Reshapes:** [D13](decision-D13.md)

### Context

The single hardest and most consequential question in the design, and the one that determines
whether "hand edits are sacred" is a real guarantee or an aspiration. Configuration files are
co-owned by hasp and the human.

### Proposal (not yet ratified)

**hasp owns marked regions; everything else is human territory, preserved exactly.** hasp does
not take custody of a file it did not fully author. Content it does not understand is not an
error and is never rewritten — it is simply not hasp's.

### Bearing of D9 on this question

[D9](decision-D9.md) materially changes the shape of this problem and should be resolved *before* D7 is
ratified. If host groups are separate files, then hasp files it created wholesale
(`~/.ssh/<group>.sshconfig`) may be fully hasp-owned, and marked-region co-ownership may be
needed only for the default `~/.ssh/config` — the one file the human has already written in.
That would narrow the hard problem considerably. Whether it narrows it *enough* to change the
answer is exactly what needs deciding.

### Open questions

- Does the default group file get marked-region treatment, or is it read-mostly with hasp
  writing only an include?
- What happens when a human edits inside a hasp-marked region?
- Are hasp-authored group files still preserved byte-for-byte outside their known constructs?

### Consequence if accepted

hasp must be able to work with a file it only partially comprehends, forever. This forecloses
any approach requiring a complete parse of a config file in order to write to it at all, and
is the primary constraint on the deferred parsing investigation.

### Decision

**Marked regions, accepted** — with elaborations that make the mechanism stronger than the
original proposal, and that answer all three open questions above.

**1. Inside its own markers, hasp is authoritative.** hasp-owned regions are *rigidly managed*:
hasp may normalize, reorder, and reformat them freely, because it wrote every byte.

**2. Manual edits inside a hasp region are intentionally lost.** This is the trade that buys
rigid management, and it is deliberate rather than regrettable. **It amends P2** — hand edits
are sacred *outside* hasp's markers; inside them they are forfeit. The marker is both the
boundary and the warning, and P2's guarantee is narrowed to match rather than quietly violated.

**3. hasp-owned regions may carry structured metadata as comments.** Anything hasp needs to
record that the format cannot express natively is written into its own region in comment form.
The metadata structure must be expressive enough to capture whatever the human intended, so
that a construct hasp does not model natively can still be carried rather than dropped.

**4. The default file is supported; explicit host groups are the nudge.** `~/.ssh/config` gets
marked-region treatment like any other file. But the smoothest experience is explicit host
group files ([D9](decision-D9.md)), and hasp should steer users there — the fully-owned file is the easy
case, and the co-owned one is where the sharp edges live.

### Why elaboration 3 matters

It **substantially dissolves the exhaustive-parsing problem**, which was the project's hardest
identified technical risk. hasp no longer needs a complete native model of the configuration
format in order to be safe with it: what it understands it manages natively, and what it does
not it carries as metadata. The parsing investigation shrinks from "model the whole format" to
"model what we manage, carry the rest."

### Is comment-metadata a violation of D12?

**No, and the distinction is worth stating explicitly**, because this is precisely the kind of
mechanism that could reintroduce persisted state through a side door.

[D12](decision-D12.md) prohibits *derived* state — a copy of facts the machine can already answer, which
drifts because the original changes without telling you. Comment-carried metadata is
**declared**: it is input, authored deliberately, mirroring nothing. It lives in the world
rather than beside it, is legible without hasp (P6), and is version-controllable with the file
that carries it.

The line to hold: **metadata may record what the machine cannot tell you. It may never cache
what the machine can.** A fingerprint written into a comment would be a D12 violation. A profile
assignment would not.

### Consequence

- **P2 is narrowed** to "sacred outside hasp's markers." The design document states the
  amendment where P2 is defined.
- hasp gains a general mechanism for recording declared intent inside files it owns, which
  **reshapes [D13](decision-D13.md)** — profile membership now has a candidate home that costs no new file.
- Rigid management means a hasp region is fully regenerable, so a corrupted or hand-mangled
  region is repairable by rewriting it rather than by parsing what someone did to it.
- The metadata format becomes load-bearing and needs its own design: it must round-trip unknown
  constructs, survive reformatting, and stay readable to a human with an editor.
