---
id: decision-D12
title: 'hasp has no persisted state; it is a pure function of the machine'
date: '2026-08-26'
status: 'Accepted · Reshapes: D6 · Opens: D13'
---
### Context

The original implementation kept a TOML state file. It was introduced for a modest reason — a
debug and development aid, for visibility into what the tool was seeing — and then quietly
became the state model. In September 2020 it began migrating to SQLite with an ORM, and that
migration is exactly where the project died: `sync` half-written, `# TODO: Left off here` in
the middle of the reconciliation logic, and two competing stores wired into different commands
for the next six years.

The question this entry settles is whether hasp needs persisted state at all.

### Decision

**It does not.** hasp derives everything it reports from the machine at the moment it is asked.
There is no index, no database, no state file. hasp is a pure function of the filesystem.

**Corollary — no cache either.** Not merely "no cache for now": a cache requires a *measured*
justification — an actual timing, on an actual `~/.ssh`, showing a delay a human notices. If
one is ever admitted it must be disposable, auto-invalidating, never consulted for correctness,
and deletable mid-run with no effect other than latency.

### Rationale

**`sync` dissolves.** The verb exists only to reconcile an index against reality. With no index
there is nothing to reconcile. The single most-worked and least-finished part of the abandoned
codebase — the part with the abandonment marker literally inside it — is not fixed by this
decision, it is *deleted* by it. That is the strongest available evidence the direction is right.

**The sin was persisting derivable facts, not persistence as such.** Derived state drifts,
because it is a copy of something that changes without telling you. Declared state does not,
because it is not mirroring anything. The old state file stored fingerprints, algorithms, bit
lengths, formats, and aliases — every one of them a question the machine answers in
milliseconds. That is what rotted.

**An audit found almost nothing that must be declared.** Fingerprint, algorithm, size, comment,
format: from the key artifact. Aliases: from symlinks. Hosts, host settings, bindings: from
config files. Host groups: from files in the key directory ([D9](decision-D9.md)). The only residue is
profile membership, which is [D13](decision-D13.md).

**A cache does not pay at this scale.** Tens of keys, a few milliseconds each (P8).
For a process that starts and exits, a cache only helps if it is on disk — which means
invalidation, staleness checks, and a second thing that can be wrong. It would reintroduce the
exact failure being eliminated in order to save time a human cannot perceive.

**And a cache is how this happened the first time.** The TOML file started as "just a
visibility aid." A cache admitted "just in case" is the same nose under the same tent, which is
why the bar above is deliberately set at measurement rather than judgement.

**Statelessness is more honest.** Where hasp cannot derive a fact it must say so. A stateful
hasp would confidently report a fingerprint it can no longer verify.

**Three properties come free.** There is no corruption mode, because there is no index to leave
inconsistent. Testing becomes tractable — hasp is a function of a directory, so a fixture tree
is the entire setup, which directly addresses why the original has zero tests. And abandonment
costs nothing, since nothing is trapped in a format nobody remembers.

### The derivation gap

One case was tested rather than assumed, and it is real:

| Case | Fingerprint derivable? |
| --- | --- |
| Public half present | Yes |
| OpenSSH format, encrypted, no public half | **Yes** — the public half is stored unencrypted inside the private file; only the comment is lost |
| Legacy PEM, encrypted, no public half | **No** — `ssh-keygen` reports `is not a key file` |

The last row matters here specifically: seven of the thirteen keys in the author's own recovered
inventory are PEM format.

**Resolution: report the fact as unknown.** hasp does not prompt for a passphrase (P3), and it
does not cache a value it cannot re-verify. "Unknown" is the correct answer and a stateful
design would answer it worse.

### Consequence

- **`sync` and `forget` are retired from the verb set**, which drops from eight verbs to six.
  `forget` meant "drop this from my records"; there are no records. [D10](decision-D10.md)'s grid is
  now 3 × 6.
- **P1 strengthens** from "the index is rebuildable" to "there is no index." **P6** — records
  legible without hasp — is satisfied maximally and vacuously, and is restated rather than kept
  as written.
- **[D6](decision-D6.md)'s two tiers collapse into one.** See that entry.
- **Storage leaves the deferred list.** There is nothing to defer.
- **Every read is a scan.** Correctness now depends on derivation being complete and cheap,
  which makes the derivation gap above a permanent design fact rather than an edge case.
- **[D13](decision-D13.md) is opened** by the one thing that could not be derived.
