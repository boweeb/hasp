---
id: decision-T45
title: 'The clean-room test: `tools/cleanroom`, a `CleanRoom` Mage target, a post-release CI job, and a shared `internal/snapshot` package'
date: '2026-09-08'
status: 'Accepted'
---
### Context

[`roadmap.md` §5.5](../../docs/roadmap.md#55-m35--hardening) exit criterion 6 states the milestone's own
mechanical proof: a clean-room install of `hasp` (README.md's one documented path, `go install
github.com/boweeb/hasp/cmd/hasp@latest`) must be able to run `list key` against a synthetic
`~/.ssh` fixture and get an accurate inventory back, then round-trip that fixture through `adopt`
and `release` byte-identical to where it started. Unlike every other M3.5 chunk, this criterion
had no assigned chunk and nothing in the repository beyond the roadmap's own prose implemented it
— confirmed by grep for "clean-room" and "clean room" turning up nothing else.

Two duplicated snapshot/diff implementations already existed by the time this entry was written:
`internal/app/adopt_release_roundtrip_test.go`'s `snapshotDir`/`assertSnapshotsEqual` (digest
includes each file's permission mode) and `internal/cli/writeguard_test.go`'s
`snapshotTree`/`assertTreeUnchanged` (digest omits it) — the first file's own doc comment already
named the reason for the duplication: `internal/cli` imports `internal/app`, so the dependency
cannot run the other way, and a `*testing.T`-coupled helper in one package cannot be called from
the other. A closer read for this entry found a **third**, `internal/cli/writeguard_alltree_test.go`'s
`snapshotTreeForGuard`/`assertTreeUnchangedForGuard` — needed because that file lives in package
`cli` itself (the only place `write.go`'s unexported `isStdinTTY` can be forced without a real
pty), and a package-`cli` file cannot see symbols defined in a package-`cli_test` file even though
both live under `internal/cli/`. `tools/cleanroom` was about to become a legitimate fourth caller
of the same logic.

### Decision

**A new package, `internal/snapshot`,** holds the digest/diff logic as two plain functions with no
`*testing.T` coupling — `Snapshot(root string) (map[string]string, error)` and
`Diff(before, after map[string]string) []string` (empty means identical) — importable from any of
`internal/app`, `internal/cli` (both packages), and `tools/cleanroom` without regard to which
already imports which. It keeps the stronger, mode-including digest `internal/app`'s version used;
the other two callers gain mode-change detection they didn't have before, not lose coverage they
did. All three prior call sites become thin wrappers: `snapshotDir`/`assertSnapshotsEqual`,
`snapshotTree`/`assertTreeUnchanged`, and `snapshotTreeForGuard`/`assertTreeUnchangedForGuard` each
keep their own name (for their own file's callers) and `t.Helper()`/`t.Errorf` shape, but their
bodies now call `internal/snapshot.Snapshot`/`.Diff` instead of reimplementing the walk. This
consolidates three existing near-duplicates into one, rather than adding a fourth.

**A new tool, `tools/cleanroom`,** matching the existing `tools/docscheck`/`tools/gendocs`/
`tools/genfixtures` convention (standalone `package main`, run via `go run ./tools/<name>`, never
imported by `cmd/hasp`). `tools/cleanroom/fixture.go` builds a small, realistic `~/.ssh`-shaped
fixture — one key already adopted into a managed profile (a `.hasp`-marked directory, D13) with
its top-level alias symlink, one still-unmanaged key, and a real ed25519 keypair via
`internal/adapter/keyfile.Generate` (the same pure-Go mechanism `new key` itself uses, T1) rather
than raw bytes, since `keyfile.Inspect` fails open and silently skips anything it cannot classify
as a key. `tools/cleanroom/main.go` execs a real `hasp` binary as a subprocess against that
fixture — `list key --json`, `adopt key ... --json --yes`, `release key ... --json --yes` — reusing
`internal/cli/render.Envelope` to unmarshal the outer `--json` envelope (its `Data` payload needs
its own narrow, unmarshal-only view, since `domain.Key`'s `Identity` field is a
`json.Marshaler`-only interface with no matching `UnmarshalJSON`). It snapshots the fixture before
and after (excluding `.hasp-backups/`, mirroring the existing round-trip test's own exclusion of
hasp's own backup store, T8/P4) via `internal/snapshot`, and reports every subprocess's own
non-zero exit and every snapshot mismatch as a hard failure — no partial-success state, per the
roadmap's own "deliberately a script rather than a judgement."

**A new, standalone Mage target, `CleanRoom`,** alongside `Fuzz`/`Fixtures`/`GenDocs`/`Release` —
excluded from `CI`'s `mg.Deps` for the same family of reasons those targets already are. Reads
`HASP_CLEANROOM_BIN`; if unset, builds the current tree to a temp binary first, so `go run mage.go
cleanroom` is also a self-contained local dev-loop smoke test.

**A new job in `.github/workflows/release.yml`, not `ci.yml`,** running after the existing
`release` job (`needs: release`) so the pushed tag and its published artifacts already exist. It
checks out the repository — unlike the hypothetical "no checkout at all" shape a simpler design
might reach for, `tools/cleanroom`'s own source lives in this repository, so the harness driving
the test legitimately needs it — but the `hasp` binary under test is never built from that
checkout. It comes from `go install github.com/boweeb/hasp/cmd/hasp@${{ github.ref_name }}`, the
tag just pushed, never `@latest`, wrapped in a bounded retry loop (six attempts, 20s apart) to
absorb the Go module proxy's own indexing lag immediately after a fresh tag push. No `container:`
directive is needed for this isolation: a GitHub Actions job is a fresh VM by default, and this
workflow does not share any `actions/cache` between the `release` job and this one, so `go install`
here already cannot resolve from the `release` job's own build/module cache — the actual risk this
criterion exists to rule out.

### Rationale

**No Dockerfile.** A container image would need to be built, published, and kept in sync with
`mise.toml`'s pinned Go version by hand — a second place that version could drift from the one
`docs/tdd.md` §17 already cites as the single source of truth. A plain GitHub Actions job already
gives the needed isolation (a fresh VM, no shared cache) for free.

**`release.yml`, not `ci.yml`.** [T41](decision-T41.md) already established the precedent this decision
extends: don't put a slow, network-dependent, and occasionally-flaky check into the aggregate that
blocks every commit and PR. `go install ...@<tag>` depends on the public Go module proxy actually
having indexed a tag that, from this workflow's own point of view, was pushed moments ago —
exactly the kind of external timing dependency `CI`'s `mg.Deps` set has no business carrying.
Gating it on `release` instead means it only runs when there is a real tag to prove, at the one
point in the release process a stranger's own `go install` would actually happen.

**`@<tag>`, not `@latest`.** `@latest` resolves to whatever the proxy currently considers the
newest version at request time — for a workflow run immediately following the tag that triggered
it, that should be the same tag, but nothing guarantees it stays that way if a second tag lands in
the same window, and it gives up the one guarantee this job actually needs: proving *this* tag's
artifact, not proximately-the-newest-one.

### Consequence

- `docs/tdd.md` §17's target-set sentence now lists `CleanRoom` alongside `Build`, `Test`, `Vet`,
  `Lint`, `Cross`, `Fuzz`, `Fixtures`, `Docs`, and `Release`, and gains a paragraph describing what
  it does and why it stays outside `CI`'s `mg.Deps`, matching how `Docs` and `Release` are already
  described there.
- `internal/app/adopt_release_roundtrip_test.go` and `internal/cli/writeguard_test.go` (and, found
  during this chunk, `internal/cli/writeguard_alltree_test.go`) all changed to call
  `internal/snapshot` instead of reimplementing it; none of their own test bodies or assertions
  changed, and the full existing suite (`go test ./...`) stays green.
- A future caller that needs "did this directory tree change" reaches for `internal/snapshot`
  first — a fourth reimplementation of the same ~20 lines would now be a regression in its own
  right, not just an opportunity missed.
- [`roadmap.md` §5.5](../../docs/roadmap.md#55-m35--hardening) exit criterion 6 now has a real, mechanical
  implementation behind its prose, runnable locally (`go run mage.go cleanroom`) and in CI
  (`.github/workflows/release.yml`'s `clean-room` job) exactly as the criterion itself demands.
