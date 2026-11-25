---
id: decision-T33
title: 'Distribution is staged: self-hosted now, public channels blocked on one missing fact'
date: '2026-09-02'
status: 'Accepted · Amends: T9'
---
### Context

[T9](decision-T9.md) enumerated the full GoReleaser surface — `ko`, `nfpms`, `aur`, `homebrew_casks`,
`sboms`, `signs` — as though every section were equally reachable from where the project actually
stands. It is not: `.goreleaser.yaml`'s own header comment still opens "M0 scope only" and still
closes "Not releasing anything yet" — true when written, false since Gitea publishing was
configured. [T9](decision-T9.md) also states that generated shell completions and man pages are "packaged
into the release archives" — nothing in the repository generates either today, and
`.goreleaser.yaml` has no handling for them. `design.md` §3.1 scopes hasp to one laptop, one
human; the near-term distribution reality is a `localhost` Gitea remote, not a public one.

### Decision

**M3.5 ships**: `tar.gz` archives for all four build targets, **generated shell completions and
man pages** — closing the gap [T9](decision-T9.md) stated but never built, which cobra already provides via
`GenManTree` and the `completion` subcommand ([T3](decision-T3.md)) — plus `sboms`, published to the Gitea
release. `aur`, `homebrew_casks`, `ko`, and `signs` stay deferred.

**The synthesis that makes this one decision instead of four separate chores**, and the most
useful thing this entry records: **all four deferred channels are blocked on the same missing
fact — hasp has no publicly reachable origin.** AUR needs a fetchable source URL; Homebrew needs a
tap repository; `ko` needs a registry to push an image to; keyless signing needs a public
OIDC-issuing CI provider. A `localhost` Gitea satisfies none of the four. It is also, precisely,
**the same question** [T31](decision-T31.md) already names as the module-path blocker: one answer to "does
hasp get a publicly reachable origin" unlocks all five items — the module path and the four
deferred channels — and no answer blocks all five. No amount of additional GoReleaser
configuration substitutes for answering it.

### Rationale

Treating four TODOs as four independent chores invites solving each partway — a tap repo pointed
at a private remote, a signing setup with no public issuer to trust — none of which actually work.
Naming the single shared blocker converts four speculative pieces of GoReleaser configuration into
one tracked decision with a clear unblocking condition.

### Consequence

The deferral is now a single named blocker — recorded as an input-needed item in
[`roadmap.md` §5.5](../../docs/roadmap.md#55-m35--hardening) — rather than four independent "later" items
that each look individually actionable and are not. `.goreleaser.yaml`'s stale "M0 scope only"
header becomes a tracked M3.5 doc-hygiene item rather than a comment nobody owns.
