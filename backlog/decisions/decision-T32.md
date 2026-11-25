---
id: decision-T32
title: 'Mage is the build/CI contract; platform workflows are thin shims'
date: '2026-09-02'
status: 'Accepted'
---
### Context

CI today is GitHub Actions YAML running against a Gitea remote — the near-term target — with GitHub
or GitLab as a long-term possibility if [T31](decision-T31.md)'s public-origin question resolves that way. As
it stands: no release workflow exists at all, so nothing triggers GoReleaser; the linter runs
unpinned (`go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest`); `.golangci.yml`
configures no linters, only a timeout; darwin is never compiled in CI, though GoReleaser targets
`darwin/arm64` among four platforms; and `mise.toml` is `[tools] go = "latest"`, which
`roadmap.md` §2 already calls "pinning the toolchain" and `tdd.md` §2 already calls "installed via
`mise` (`go = \"latest\"`)" — neither description is true of `latest`. Encoding the actual build
logic in any one platform's YAML makes a future move a rewrite instead of a shim.

### Decision

Every build/test/lint/release action becomes a **Mage target** in `magefiles/`
(`github.com/magefile/mage`, latest **v1.17.2**, verified against the module proxy this session);
a platform workflow file's only job is checkout → set up Go → invoke one target. The target set:
`Build`, `Test`, `Vet`, `Lint`, `Cross` (`GOOS=darwin` compile), `Fuzz`, `Fixtures`, `Docs`,
`Release`, and `CI` as the `mg.Deps` aggregate that runs the others.

**Zero-install bootstrap**, so CI needs no mage binary of its own: a `mage.go` carrying
`//go:build ignore` that calls `mage.Main()`, invoked as `go run mage.go <target>`. This bootstrap
has a caveat worth stating verbatim rather than discovering it against a red build later, from
magefile.org: *"because of the peculiarities of `go run`, if you run this way, go run will only
ever exit with an error code of 0 or 1."* A project with a four-code exit contract ([T14](decision-T14.md))
must not build an exit-code-sensitive pipeline — `CI`'s own exit status — on a bootstrap that
flattens every failure to `1`; the workflow shims (below) treat any non-zero exit from
`go run mage.go ci` as failure and never branch on its specific value.

**Dependency budget: a new row.** Mage is the first build-time-only dependency this project has
taken on. `tdd.md` §2's "Test-only, never shipped" table gets a sibling, "Build-time only, never
shipped," carrying `github.com/magefile/mage`. The guard is mechanical, in this project's own
style: [T13](decision-T13.md)'s existing `go list -deps` layering guard extends to assert
`github.com/magefile/mage` never appears in `cmd/hasp`'s dependency graph — a magefile importing
it is fine; `cmd/hasp` importing it is a regression. `//go:build mage` tags stay on every file
under `magefiles/`, so `go build ./...` and `go vet ./...` ignore them exactly as they do today.

**Also decided here**, because each is a direct consequence of moving CI into Mage rather than an
unrelated bundle of chores:

- `golangci-lint` is pinned to an exact version inside the `Lint` target — an unpinned `@latest`
  makes lint non-reproducible and turns an unrelated upstream release into a red build on a commit
  that changed nothing.
- `.golangci.yml` adopts a real linter set — noting that golangci-lint v2 requires `version: "2"`
  in the config file, which the current file does not declare.
- `mise.toml` is pinned to the exact Go toolchain in use, which resolves the contradiction named
  in Context.

**Gitea specifics**, named so they are not discovered the hard way: Gitea Actions consumes
GitHub-Actions-compatible YAML but needs a registered `act_runner`, and resolves a step's `uses:`
against its own `DEFAULT_ACTIONS_URL` rather than implicitly against github.com; the release shim
needs a `GITEA_TOKEN`. Both `.github/workflows/` and `.gitea/workflows/` shims are kept — each is
a handful of lines precisely because the logic lives in Mage, not in either file.

**The `Docs` target is the one that protects this project's own culture**, and is worth its own
paragraph rather than a line item. It commits the doc checkers this repository already relies on
by hand — broken links and anchors, every `Dn`/`Tn` citation resolving to a real `<a id>`, the
per-log invariants (index row count equals anchor count, IDs contiguous, every entry carrying a
`### Consequence`), and verbatim-quotation checking — and runs them in CI as part of `CI`'s
`mg.Deps` set. They are written in Go, as a Mage target, rather than adding a Python dependency:
the toolchain is already here, and a doc-verification script that needs a second language
installed is a script nobody runs locally. Two gotchas are recorded here because they have already
cost real time once and would again: (a) GitHub anchor slugs give each space its own hyphen and do
not collapse them, so an em-dash heading yields a *double* hyphen and a naive `\s+`→`-` regex
reports a dozen false breaks; (b) a quotation can wrap across source lines, so a single-line
`grep -F` produces false misses — normalize whitespace and strip `**`, `*`, and backticks before
comparing.

### Rationale

`mg.Deps` runs its dependencies concurrently and `mg.SerialDeps` runs them in sequence, and either
way Mage guarantees each dependency is "guaranteed to run exactly once in a single execution of
mage" — precisely the primitive a `CI` aggregate target needs to compose `Vet`, `Lint`, `Test`,
`Cross`, and `Docs` without hand-rolling its own dependency bookkeeping. `mage -compile <path>` can
also emit a standalone static binary later, if the zero-install `go run` path ever proves too slow
for local iteration — an escape hatch that costs nothing to have and nothing to use today.

### Consequence

- Moving to GitHub or GitLab later — the outcome [T31](decision-T31.md)'s public-origin question may
  produce — is a new shim file, not a rewrite: the workflow file changes, `magefiles/` does not.
- The doc-verification pass becomes mechanical, run on every push, rather than something someone
  has to remember to run before merging — which is exactly the discipline this repository's
  culture already claims to have and, until this entry, did not actually enforce.
- `mage.go`'s exit-code caveat means the workflow shims must treat "CI failed" as a single boolean,
  never inspect a specific non-zero code from the bootstrap path — a distinction any future
  workflow author needs to know before reaching for it.
