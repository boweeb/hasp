---
id: decision-T14
title: 'Output contract: 4 exit codes, a versioned JSON envelope, silent stdout'
date: '2026-08-28'
status: 'Accepted'
---
### Context

[`design.md` §6.3](../../docs/design.md#63-cross-cutting-requirements) requires *"every read has a
machine-readable form"* — `design.md` does not use the word "contract," but this is what a
requirement worded that way means in practice: `hasp list key --json | jq` must keep working
release over release, not merely work once. `check` ([§6.2](../../docs/design.md#62-the-verbs)) is
explicitly advisory and must be usable in a pipeline, which means "found issues" and "failed to
run" have to be distinguishable by exit code alone.

### Decision

Four exit codes, and only four:

| Code | Meaning | Who can return it |
| --- | --- | --- |
| `0` | Clean — no findings, or a write completed | Any command |
| `1` | Findings — `check` surfaced issues; not a hasp failure | `check`, exclusively |
| `2` | Usage — bad flags, bad arguments, or an unresolvable non-interactive precondition ([T6](decision-T6.md)'s fail-closed case) | Any command |
| `3` | Error — hasp could not complete the requested operation (I/O failure, a fail-closed guard tripped mid-write, backup failed) | Any command |

Cobra itself never calls `os.Exit`; `cmd/hasp`'s `main()` maps the error returned from
`rootCmd.Execute()` to one of the four codes via a small sentinel-error taxonomy
(`errors.Is(err, app.ErrFindings)`, `errors.Is(err, app.ErrUsage)`, else `3`).

`log/slog` writes to stderr, **off by default**, enabled only by `--verbose` — stdout carries
data and nothing else, which is what makes the pipeline promise literal rather than aspirational.

JSON output is wrapped in a versioned envelope:

```go
type Envelope struct {
    Version  int             `json:"version"`  // envelope schema version — independent of hasp's own release version
    Kind     string          `json:"kind"`      // "key.list", "host.show", "check.report", ...
    Data     json.RawMessage `json:"data"`
    Warnings []string        `json:"warnings,omitempty"`
}
```

### Rationale

P7 (*"answer the question in one screen"*) governs the human renderer; §6.3's machine-readable-form
requirement governs the JSON one — the two are explicitly allowed to diverge in **form** but must
never diverge in **content**, which [T13](decision-T13.md)'s single read-model aggregate guarantees by
construction: there is exactly one code path that computes an answer, and two that display it.

A versioned envelope from the first release, not retrofitted after the first breaking change, is
what makes the machine-readable form a contract in practice rather than an accident of the
current implementation — a consumer piping `hasp ... --json` through `jq` can branch on
`.version` whenever it needs to, rather than only after hasp breaks it without warning.

### Consequence

- `check` is the **only** use case permitted to resolve to exit code `1` — no other verb ever
  does, which keeps the meaning of "1" stable across the entire surface: a scripted
  `if hasp check ...; then` reads unambiguously, forever.
- `--verbose`'s diagnostic stream and `--json`'s data stream can never collide, because they are
  different file descriptors by construction, not by convention that a future change could erode.
