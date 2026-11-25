---
id: decision-T42
title: 'Signing the release covers both the checksum and the `kos`-built container image, via two GoReleaser sections'
date: '2026-09-05'
status: 'Accepted · Amends: T9'
---
### Context

[T9](decision-T9.md) named `signs` as one of the GoReleaser v2 idioms this project embraces;
[`tdd.md` §13](../../docs/tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought)
repeats it without saying which artifact gets signed. Implementing the `signs:` block for M3.5.5
([`roadmap.md` §5.5](../../docs/roadmap.md#55-m35--hardening)) first assumed GoReleaser had no signing hook
for `kos:`-built images, since the top-level signing documentation lists archives, installers,
packages, checksums, and `dockers:`-built manifests, with no direct mention of `kos:`. That
assumption was wrong: GoReleaser's own `kos:` documentation states that a ko-built manifest is
added to the same artifact list `dockers:`-built images use, and it is `docker_signs:` — not
`signs:` — that signs artifacts from that list via `artifacts: manifests`.

### Decision

Both artifact classes carry a keyless cosign signature, via two separate `.goreleaser.yaml`
sections because each artifact lands in a different GoReleaser artifact list: `signs:` signs the
release checksum (`sign-blob --bundle`, `artifacts: checksum`); `docker_signs:` signs the
`ghcr.io/boweeb/hasp` image manifest `kos:` pushes (`sign`, `artifacts: manifests`). Both use the
same OIDC/Fulcio/Rekor keyless flow — no private key, no separate credential to manage.

### Rationale

The checksum file covers every archive and SBOM this release produces, but it says nothing about
the container image's integrity — a signed checksum and an unsigned image would have left exactly
the gap `signs`'s stated purpose (supply-chain attestation, [`tdd.md` §13](../../docs/tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought))
exists to close. Once GoReleaser's actual capability was confirmed rather than assumed, shipping
both signatures cost one more config block, not a deferred chunk.

### Consequence

- [`tdd.md` §13](../../docs/tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought)'s
  staging table now says `signs` ships covering both the checksum and the container image, citing
  this entry.
- A future reader reaching for `signs:` to cover a `kos:`-built artifact should reach for
  `docker_signs:` instead — GoReleaser's own section split, not a hasp-specific quirk, but easy to
  miss since `kos:` and `docker_signs:` don't sit next to each other in the config file.
