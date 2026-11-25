---
id: decision-T4
title: 'The `Plan` type: change is represented as data before it is applied'
date: '2026-08-28'
status: 'Accepted — amended by T20, T22, T26'
---
### Context

[D6](decision-D6.md) makes preview the default for **every** write, with no
operation-by-operation judgment call about which changes are risky enough to deserve it. P5
states this explicitly as a modeling constraint, not an interface nicety: *"the ability to
preview is not a convenience flag bolted on later; it constrains how change is modeled from the
beginning."* A design where a use case performs I/O directly and merely *logs what it did*
cannot satisfy this — preview must exist **before** anything is touched, which means the thing
previewed must be a value, not a side effect already in flight.

### Decision

Every write-verb use case returns a `Plan` — an ordered list of typed `Change` values — instead
of performing I/O. A single `Applier` is the only thing in the codebase permitted to touch the
filesystem for a write.

```go
type Plan struct {
    Summary string
    Changes []Change
}

type Change interface {
    Describe() string        // one-line preview text, rendered by both the human and JSON renderers
    RequiresBackup() bool
    Apply(fsys WriteFS) error
}

type MoveFile struct{ From, To, Reason string }
type WriteRegion struct{ File string; Marker RegionID; Body []byte }
type CreateSymlink struct{ Path, Target string }
type CreateMarker struct{ Dir string; Header []byte }
type WriteKeyFile struct{ Path string; Contents []byte; Mode fs.FileMode } // the D14 exception

type Applier struct {
    FS      WriteFS
    Backups BackupStore
}

func (a *Applier) Apply(p Plan) (Result, error) {
    for _, c := range p.Changes {
        if c.RequiresBackup() {
            if err := a.Backups.Snapshot(c); err != nil {
                return Result{}, fmt.Errorf("backup failed, aborting before write: %w", err)
            }
        }
        if err := c.Apply(a.FS); err != nil {
            return Result{}, err
        }
    }
    return Result{Applied: p.Changes}, nil
}
```

A use case's method signature is `Plan(req Request) (Plan, error)` — it may perform *reads* to
compute the plan (P5: reads are always safe), but returns before anything is written.

**Three later entries refine this type sketch rather than replace it: [T20](decision-T20.md)** states the
general ordering rule for `Changes` and adds no new type; **[T22](decision-T22.md)** adds an
`AllowOverwrite` field to `WriteKeyFile` and a `Remove` change kind; **[T26](decision-T26.md)** replaces
`Describe() string` with `Preview() Preview`, a structured value carrying an optional diff,
because `Describe`'s single line could not represent what a `WriteRegion` rewrite actually
changes.

### Rationale

This is the architectural keystone the plan for this document names explicitly: D6 is only
structural, rather than a flag bolted onto each command, if change is data first and an
executable action second. It is the Go-appropriate stand-in for the Unit-of-Work pattern Cosmic
Python uses for a different purpose ([T13](decision-T13.md) states why UoW itself does not port).

**The resulting inversion is worth stating in exactly these terms: `--dry-run` does not exist,
because preview *is* the default.** The flag that exists is `--yes` — its absence is what
triggers the preview-only path, not its presence.

### Consequence

- Every use case in `internal/app` is testable without touching a filesystem: assert on the
  returned `Plan` value. This is the design that makes the guard test in
  [`tdd.md` §12](../../docs/tdd.md#12-testing-strategy--a-function-of-a-directory) ("hasp writes nothing
  outside the key directory") tractable to write per-use-case, not just end-to-end — the
  `testscript` `.txtar` cases in that same section cover the full end-to-end path.
- New `Change` kinds are additive and carry their own `RequiresBackup()` answer — nothing about
  the safety machinery depends on a call site remembering to ask for a backup; it is a property
  of the change, not of the caller's diligence.
- A `Plan` with zero `Changes` is a legitimate, renderable value ("nothing to do") — every
  write-verb use case can report "no change needed" through the same preview path used for a
  real change, so there is no separate no-op code path to keep in sync.
