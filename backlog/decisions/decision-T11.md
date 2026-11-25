---
id: decision-T11
title: 'Host-group composition: one `Include` line per group, hasp orders them'
date: '2026-08-28'
status: 'Accepted'
---
### Context

[D9](decision-D9.md) makes host groups real files (`~/.ssh/<group>.sshconfig`), but
`design.md` §10 leaves *"the composition mechanism behind host groups (§5.5)"* explicitly
deferred. OpenSSH's own `ssh_config(5)` semantics are **first-obtained-value-wins** per
parameter — the first matching value for a given keyword wins, later ones are ignored — which
makes the *order* group files are pulled into `~/.ssh/config` a fact with real behavioral
consequences, not a cosmetic one.

### Decision

hasp writes **one `Include` directive per host group it manages**, inside its own marked region
of `~/.ssh/config` — never a single glob (`Include ~/.ssh/*.sshconfig`) — so that **the position
of each `Include` line in the marked region is the precedence order**, controlled by hasp
directly. This is not a defense against nondeterminism: `ssh_config(5)` states that `Include`'s
wildcards *"will be expanded and processed in lexical order,"* which is deterministic. It is a
defense against the wrong *deterministic* order — lexical filename order is not necessarily the
order the user wants, and a glob would force naming discipline on group files (`0-personal`,
`1-work`) purely to win a precedence fight that explicit ordering avoids entirely.

```text
# >>> hasp:managed >>>
Include ~/.ssh/work.sshconfig
Include ~/.ssh/personal.sshconfig
# <<< hasp:managed <<<
```

### Rationale

D9's own rationale — *"SSH configuration is already multi-file in practice; the format supports
composition natively"* — is only fully honored if the composition is legible and hasp-controlled;
a glob makes the *effective* precedence order a function of how the group files happen to be
named, which is a fact the user would have to manage by naming discipline rather than one hasp
states directly (P1 — hasp should report the order, not make the user reverse-engineer it from
filenames). Because this block sits inside hasp's own marked region (D7), reordering host groups
is an ordinary `Plan`/`Change` ([T4](decision-T4.md)) — never a hand edit outside markers, keeping P2 intact
for the block's neighbors.

### Consequence

- Adding or removing a managed host group is always a `WriteRegion` change to the *same* marked
  region (append or remove one `Include` line) — never a separate edit outside markers.
- [T2](decision-T2.md)'s CST must treat `Include` as **semantically understood**, not opaque trivia, when it
  appears inside a marked region — it is part of the natively-managed directive set precisely
  because hasp needs to reason about its argument and its position, not merely preserve it.
  Outside a marked region, a human-written `Include` line round-trips like any other unrecognized
  construct — hasp reads it (to know what it must not shadow) but never rewrites it.
- `check` gains a finding category from first-obtained-value-wins directly: a later `Include`d
  group defining a `Host` pattern already fully shadowed by an earlier group or stanza is dead
  configuration — reachable by nothing — and is exactly the kind of silent bite this entry exists
  to name and catch rather than let a human discover the hard way.
