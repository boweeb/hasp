---
id: decision-T46
title: 'Container base image: `cgr.dev/chainguard/static` → `gcr.io/distroless/static:nonroot`'
date: '2026-09-09'
status: 'Accepted'
---
### Context

Re-cutting `v0.6.0` after making the repository public ([T40](decision-T40.md)) failed at the `ko` publish
step: `fetching base image: GET https://cgr.dev/v2/chainguard/static/manifests/latest:
MANIFEST_UNKNOWN`. Querying `cgr.dev` directly (outside CI) confirmed this isn't a missing-tag
fluke — anonymous pulls now return `401 Unauthorized` outright, and Chainguard's free tier
requires signing up with a business email, a bar a personal open-source project the author
maintains alone cannot honestly clear. `.goreleaser.yaml`'s `kos:` block had used
`cgr.dev/chainguard/static:latest` since M3.5's shipping-set, deliberately left floating rather
than pinned to a digest — the comment there contrasted this against [T32](decision-T32.md)'s dev-tool pins,
so a distroless base would keep picking up security patches on every release.

### Decision

Replace `cgr.dev/chainguard/static:latest` with `gcr.io/distroless/static:nonroot` in
`.goreleaser.yaml`'s `kos:` block. The floating-tag choice itself is unchanged — `:nonroot` still
tracks upstream patches on every release rather than freezing a digest — only the registry and
image family change, to one that still permits anonymous pulls.

`:nonroot`, not the bare `:latest` tag, is the deliberate part of this decision: Chainguard's
`static` image runs as a non-root user by default, while Google's plain
`gcr.io/distroless/static:latest` runs as **root** unless the `:nonroot` tag is used. Swapping to
the bare tag would have silently regressed the container's UID posture — exactly the sharp edge
`tdd.md` §13's "Container caveat" already calls out by name ("a mismatched UID inside the
container silently produces a `~/.ssh` the host user cannot read"). `:nonroot` keeps the same
UID-safety behavior the Chainguard image provided.

### Rationale

**Distroless, not Alpine or a full base.** [T32](decision-T32.md)'s floating-vs-pinned distinction depended on
the image being minimal and low-churn to begin with, not on it being Chainguard's specifically.
`gcr.io/distroless/static` is the same shape — no shell, no package manager, matching `tdd.md`
§13's "no `Dockerfile`, `FROM scratch`-capable" description of the container target — from a
registry with no anonymous-access gate.

**Why not authenticate to Chainguard instead of switching.** A free Chainguard account is real,
but gating it behind a business email is a signal, not an accident: personal open-source
maintainers are not the audience it is priced for. Wiring a login step and a token secret into
`release.yml` to work around that would trade a one-line `base_image:` swap for an ongoing
account-maintenance dependency this project has no organizational backing to sustain.

**This entry serves [P6](../../docs/design.md#4-principles) directly — the same clause [T40](decision-T40.md) already
invoked for the module path.** T40's own rationale is that a publicly resolvable module path is
what makes P6's "nothing hasp knows may be trapped inside it" promise reachable by someone who is
not the author; a container image gated behind a business-email signup is exactly the same kind
of barrier applied to a different artifact, `ko`'s published image rather than `go install`'s
module path. An image nobody but a signed-up account can pull is, in practice, a released artifact
only the author can use — the same failure mode P6 already names for anything hasp would keep
only to itself, applied here to distribution rather than to data.

### Consequence

- `.goreleaser.yaml`'s `kos:` block now points at `gcr.io/distroless/static:nonroot`; its comment
  still names Chainguard once, specifically to explain why `:nonroot` — not the bare tag — was
  chosen; the base image itself is no longer Chainguard's.
- No `tdd.md` change: §13 already described the container target generically, without naming a
  specific registry or vendor, so nothing there was inaccurate to begin with.
- `v0.6.0` is re-cut a second time against this fix, expected to clear the `ko` publish step and
  reach [T45](decision-T45.md)'s clean-room job on the now-public repository.
