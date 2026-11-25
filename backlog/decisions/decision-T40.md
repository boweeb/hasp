---
id: decision-T40
title: 'The public origin is GitHub; T33''s shared blocker resolves and distribution restages'
date: '2026-09-04'
status: 'Accepted · Amends: T31, T32, T33'
---
### Context

[T33](decision-T33.md) named one missing fact behind four deferred distribution channels — `aur`,
`homebrew_casks`, `ko`, and `signs` — and predicted that a single answer would unlock all four at
once. [T31](decision-T31.md) already made the same fact load-bearing for the Go module path, because v1.0.0
freezes it before the tag rather than after. [T32](decision-T32.md) took a local Gitea as the near-term CI
target, naming GitHub or GitLab only as a long-term possibility "if [T31](decision-T31.md)'s public-origin
question resolves that way." It has resolved: the author dropped Gitea — a temporary local service
that had become more trouble to run than it was worth — and moved `origin` to
`git@github.com:boweeb/hasp.git`. (The reason is recorded as the author's, given when the change
was made; it is deliberately not set in quotation marks, because a quotation this log cannot
resolve against a file in the repository would be a permanent false positive for the
verbatim-quotation check [T32](decision-T32.md)'s `Docs` target specifies.)

`main` is the repository's default branch, per the author. The Python predecessor's history is
temporarily co-located at the same remote on `master`, bound for cold storage and deletion.
`git merge-base main origin/master` reports no common ancestor, so the two histories are unrelated
— which keeps `docs/README.md`'s and `design.md`'s existing statements that the predecessor's
`README.rst` is not carried into this repository literally true of this history. Branch layout on
the remote beyond that single fact is not verified here: SSH access to GitHub is unavailable from
this session.

### Decision

Three consequences follow, stated plainly:

1. **The module path is now honest.** `github.com/boweeb/hasp` resolves;
   `go install github.com/boweeb/hasp/cmd/hasp@latest` works; [T31](decision-T31.md)'s freeze-before-tag
   requirement is met before v1.0.0 is tagged, not discovered after.
2. **GitHub Actions is the CI platform.** The `.gitea/workflows/` shim [T32](decision-T32.md) planned as a
   deliverable is not built — Mage remains the contract; only the platform invoking it changed.
3. **Distribution restages.** `signs` (cosign keyless signing via GitHub Actions' OIDC issuer) and
   `ko` (publishing a container image to `ghcr.io`) join M3.5's shipping set alongside archives,
   generated completions, man pages, and `sboms`. `aur` and `homebrew_casks` remain deferred.

### Rationale

Two points are worth making, and the second is the more useful one.

**[T32](decision-T32.md)'s thin-shim rule is not validated by this migration, and claiming otherwise would be
the wrong lesson to draw.** An earlier draft of this entry said the rule "paid off"; it did not,
because there was nothing yet for it to pay off *on*. The honest state of the repository is that
`magefiles/` and `mage.go` **do not exist**, the only workflow — `.github/workflows/ci.yml`, from
M0 — hardcodes `go build`, `go vet`, `golangci-lint@latest` and `go test` directly in YAML, which
is precisely the shape T32's rule exists to prevent, and there is no release workflow at all. The
migration cost exactly one thing: a `release.gitea` → `release.github` change in
`.goreleaser.yaml`.

What this *does* establish is narrower and still worth acting on: **the platform moved before any
platform-specific build logic was written, which is the cheapest moment such a move can happen and
the strongest argument for adopting T32's rule now rather than after the release pipeline is
built.** The rule remains untested. T32's own Context already names this gap, and nothing here
closes it — `magefiles/`, the zero-install `mage.go` bootstrap, a GitHub Actions shim that actually
invokes a Mage target, and a tag-triggered release workflow are all still outstanding M3.5 work.

**The two remaining deferrals are qualitatively different from the four [T33](decision-T33.md) named, and
collapsing them would be a mistake.** Origin was *a fact hasp lacked*; a Homebrew tap and an AUR
package repository are *work the author must choose to take on* — each a separate repository the
author must create and maintain. The first kind of deferral ends when someone answers a question;
the second ends when someone commits to maintenance. Naming the difference keeps `aur` and
`homebrew_casks` honest deferrals rather than ones inherited unexamined from [T33](decision-T33.md).

This entry serves `design.md` §3.1's one-laptop scope and [P6](../../docs/design.md#4-principles): nothing
hasp knows may be trapped inside it, and a publicly resolvable module path plus a real release is
what makes that promise reachable by someone who is not the author.

### Consequence

- [`roadmap.md` §5.5](../../docs/roadmap.md#55-m35--hardening) loses its last `INPUT NEEDED` block — M3.5 now
  has no open questions.
- [`tdd.md` §15](../../docs/tdd.md#15-open-questions) drops from two open items to **one** (multi-directory
  key locations), and the T-log's own `## Still open` roll-up below is updated in the same pass —
  a count that has drifted before.
- `.goreleaser.yaml`'s `gitea_urls` and `release.gitea` became dead config pointing at `localhost`
  the moment origin moved; corrected in this pass to `release.github`.
- The predecessor's Python history is temporarily co-located at the same remote on `master`, bound
  for cold storage and deletion. `git merge-base` confirms the two histories share no ancestor, so
  the documentation set's existing claim that the predecessor's `README.rst` is not carried into
  this repository stays literally true.
