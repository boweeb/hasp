---
id: decision-T41
title: '`CI`''s `mg.Deps` aggregate excludes `Fixtures`: non-deterministic fixtures race `Test` and break `TestFullInventory`'
date: '2026-09-04'
status: 'Accepted · Amends: T32'
---
### Context

M3.5.2 (`roadmap.md`'s CI-shim chunk) rewired `.github/workflows/ci.yml` to invoke exactly one
step, `go run mage.go ci`, in place of the hardcoded `go build`/`go vet`/`golangci-lint`/`go test`
steps [T40](decision-T40.md) found still in place — the first time anyone actually ran `go run mage.go ci`
against the real workflow. [T32](decision-T32.md)'s original target set for `CI`'s `mg.Deps` aggregate was
`Build, Vet, Lint, Test, Cross, Fixtures`. Running it surfaced two real bugs, not one:

1. **A race.** `mg.Deps` runs its dependencies concurrently ([T32](decision-T32.md)'s own Rationale cites
   this). `Fixtures` shells out to `go run ./tools/genfixtures`, which rewrites all 28 files under
   `testdata/keys/` using `crypto/rand`, while `Test` reads those same files — `Test` can start
   reading a fixture directory `Fixtures` is still mid-write, producing spurious failures like "got
   27 private-key fixtures, want 28".
2. **A correctness bug, worse than the race.** Serializing `Fixtures` before `Test` does not fix
   this: `tools/genfixtures` regenerates fresh, random key material on every run, and
   `internal/cli/fullinventory_test.go`'s `TestFullInventory` hardcodes a fingerprint-derived
   assertion against the *currently-committed* fixture content — `run("find", "key",
   "T4RN-6go9")` is a mangled fragment of the real SSH fingerprint of the committed
   `testdata/keys/ed25519-openssh-plain-nopub` file, asserted to resolve to `id_ed25519_foobarco`.
   Regenerating that file with a fresh random keypair changes its fingerprint and breaks the
   assertion deterministically, every time — not flakily, and not fixable by reordering.

Direct experiment during this chunk confirmed both: `go run mage.go ci` as [T32](decision-T32.md) specified it
fails with the race symptom and, after serializing, with the `TestFullInventory` fingerprint
mismatch; `go build ./... && go vet ./... && go test ./...` run without touching `Fixtures` at all
— i.e. against `testdata/keys/` exactly as committed — passes clean, including
`TestFullInventory`. `tools/genfixtures`'s own package doc already calls it "a development-time
convenience, run via `go run ./tools/genfixtures` from the repo root": a manual
regenerate-then-commit step, not something CI should re-run and discard on every invocation.

### Decision

`CI`'s `mg.Deps` aggregate drops `Fixtures`, leaving `Build, Vet, Lint, Test, Cross` — the same set
[T32](decision-T32.md) already excluded `Fuzz` from, for the same reason: `Fixtures` is a standalone,
manually-invoked dev-time target (`go run mage.go fixtures`), not part of the blocking set every
`CI` run executes. `Fixtures` itself is unchanged as a target; only its membership in `CI`'s
dependency list is removed. `Test` does **not** gain an `mg.Deps(Fixtures)` call either — that
ordering was tried and rejected for the same correctness reason: any run that touches
`testdata/keys/` invalidates `TestFullInventory`'s hardcoded fingerprint fragment regardless of
race conditions.

### Rationale

`testdata/keys/` as checked into git is hand-committed, checked-in content that other tests
already depend on by fingerprint value, not a build artifact `CI` is entitled to regenerate and
discard. Treating `Fixtures` as part of the blocking aggregate conflated two different kinds of
target: things that verify the tree as committed (`Build`, `Vet`, `Lint`, `Test`, `Cross`) and a
thing that mutates the tree (`Fixtures`). Mixing a mutator into a concurrent `mg.Deps` set with the
tests that read its output was the root mistake — `mg.Deps`'s concurrency guarantee ("exactly once
per execution," cited in [T32](decision-T32.md)'s Rationale) says nothing about ordering between
non-dependent targets, so `Fixtures` and `Test` were always racing once both were listed together.

### Consequence

- [`tdd.md` §17](../../docs/tdd.md#17-continuous-integration-and-release-automation)'s statement of `CI`'s
  actual `mg.Deps` set is corrected to `Build, Vet, Lint, Test, Cross` and now says the thin-shim
  rule is exercised by a real workflow, not still unproven.
- Anyone regenerating fixtures locally must still run `go run mage.go fixtures` by hand and commit
  the result — `CI` will never do this silently, and a future fixture regeneration that changes
  `TestFullInventory`'s fingerprint assumptions must update that test in the same commit.
- The general lesson generalizes beyond this one target: no future Mage target that mutates
  checked-in files under `testdata/` (or anywhere else `git status` would show as dirty) should be
  added to `CI`'s `mg.Deps` set without checking whether another target in that same concurrent set
  reads what it writes.
