---
id: decision-T13
title: 'DDD adapted to Go: no Unit of Work, no message bus, split aggregate boundary'
date: '2026-08-28'
status: 'Accepted'
---
### Context

The architectural style for this project is DDD. Its most available reference material —
Cosmic Python — is Python-flavored: its patterns (a Unit of Work wrapping a database session, a
Repository pattern backed by that session, a message bus dispatching domain events to multiple
handlers, ports collected in a separate `abstractions` module) assume there is state to
transact over and more than one process or consumer in the picture. [D12](decision-D12.md)
(no persisted state, hasp is a pure function of the machine) and P8 (one laptop, one process, one
invocation) both remove exactly the conditions those patterns exist to serve.

### Decision

Four layers, dependency direction strictly inward:

```text
internal/domain     Key, Host, Profile, HostGroup, Binding, value objects
                     (Fingerprint, ProfilePath, KeyName, HostPattern) — zero third-party imports
internal/app         one use case per verb×noun cell; owns the Plan type (T4);
                     declares ports as interfaces at the point of consumption
internal/adapter     scan, keyfile (T1), sshconfig (T2), settings (T7), backup (T8), fswrite (T15)
internal/cli         cobra wiring (T3), flags, human + JSON renderers (T14)
```

**Two deliberate deviations from Cosmic Python, each justified against a specific principle:**

1. **No Unit of Work, no repository-with-a-session.** Nothing is persisted (D12), so there is no
   transaction to own. `Applier` ([T4](decision-T4.md)) is not a UoW in disguise — it has no rollback across
   multiple aggregates on partial failure, only a per-`Change` backup-then-write guarantee.
2. **No message bus, no domain events.** P8: *"a single human operating interactively"* — there
   is no second consumer for an event to reach. A cross-cutting effect (adopting a key must also
   ensure a top-level alias exists, per D13) is an ordinary function call inside the use case
   building the `Plan`, not a published event with a handler registered elsewhere.

**Ports are declared at the point of consumption**, in `internal/app`, per-use-case — the Go
idiom of "accept interfaces, return structs" — rather than collected in a separate abstractions
module the way Cosmic Python centralizes them.

**The aggregate boundary differs by direction, deliberately:**

- **Reads use one read-model aggregate** — a single derived snapshot of "the machine," covering
  keys, hosts, profiles, and bindings together, computed once per invocation. Justified by P8
  (cheap at this scale) and *required* by P7: answering "which hosts use this key" in one screen
  needs the join computed once, not stitched together in the CLI layer from several independent
  repository calls.
- **Writes are file-scoped** — one aggregate boundary per file being changed. The file is hasp's
  actual unit of atomic replacement ([T15](decision-T15.md)) and of backup ([T8](decision-T8.md)); two files changed in
  one `Plan` are two independent `Change` entries, each independently backed up. There is no
  cross-file transaction, because POSIX offers no way to rename two files atomically as one
  operation — stated plainly here rather than implied away.

### Rationale

Every deviation above traces to a specific principle already in force (D12, P8, P7) rather than
to "DDD is inconvenient here" — the whole discipline of this project is that an argument gets
settled by citation, and an architectural deviation is exactly the kind of claim that needs one.

### Consequence

- A use case touching two files (for example, `edit host --group` moving a stanza between host
  groups) produces a `Plan` with two `Change` entries and **no atomicity guarantee stronger than
  "each individual file write is atomic"** ([T15](decision-T15.md)). If the second write fails after the
  first succeeds, the result is a `check`-detectable partial application — the stanza present in
  both groups, or in neither — and `check` must carry a finding for exactly this shape. This is
  documented here so a future contributor does not assume `Plan` grants all-or-nothing atomicity
  across files when it only grants it per file.
- A guard test enforces `internal/domain` has zero third-party imports — the domain layer is pure
  Go standard library plus its own types, checkable mechanically (`go list -deps` on the package,
  asserted in CI-equivalent tooling), not merely a convention someone has to remember.
