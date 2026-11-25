---
id: decision-T9
title: 'GoReleaser v2 idioms: `ko`, `nfpms`, `aur`, `homebrew_casks`, `sboms`, `signs`'
date: '2026-08-28'
status: 'Accepted'
---
### Context

GoReleaser is a stated constraint on this project, to be embraced rather than worked around.
`design.md` §13 (via [`tdd.md`](../../docs/tdd.md)) needs: `CGO_ENABLED=0` (made unconditionally achievable
by [T1](decision-T1.md)), a container image path, and package-manager coverage that includes the Arch User
Repository, since the dev host is Arch Linux.

### Decision

Verified against the installed `2.14.0` this session, with docs current at v2.18:

| Section | Status | hasp uses it for |
| --- | --- | --- |
| `ko` | Current, recommended | The container image — no `Dockerfile`, `FROM scratch`-capable because of [T1](decision-T1.md) |
| `dockers` / `docker_manifests` | **Deprecated** | Not used |
| `dockers_v2` | Current (the Docker-specific route) | Not used — `ko` is preferred over hand-writing a Dockerfile for a statically linked binary |
| `nfpms` | Current | `.deb` / `.rpm` / `.apk` packages |
| `aur` | Current | Arch User Repository — the dev host's own package manager |
| `brews` | **Deprecated** | Not used |
| `homebrew_casks` | Current (`brews`'s replacement) | macOS distribution |
| `sboms` | Current | Supply-chain attestation |
| `signs` | Current | Artifact signing |

Four OS/arch build targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` —
matching the "macOS + Linux long-term" scope, dev host x86 Arch Linux via `mise` (`go = "latest"`,
currently `go1.27.1`).

Generated shell completions and the man tree (from [T3](decision-T3.md)'s cobra, via
`cobra.Command.GenManTree` and the built-in `completion` subcommand) are packaged into the
release archives, not left to `go install` users to generate themselves.

### Rationale

Embracing GoReleaser's idioms rather than working around them means letting its recommended path
(`ko`) determine a hasp design constraint, not the reverse: `ko`'s default posture assumes a
statically linked binary with no runtime dependency on anything not already in a distroless base
image. That assumption only holds because [T1](decision-T1.md) removed the `ssh-keygen` subprocess and
`libmagic` cgo dependency the predecessor needed — had either survived, `ko`'s scratch-based
image would silently break at the first key read, which is exactly why T1 and T9 are one linked
decision, not two independent ones, despite being numbered separately.

### Consequence

- `dockers`, `docker_manifests`, and `brews` are explicit **non-choices**: a future contributor
  reaching for any of them is reaching for a deprecated GoReleaser v2 section, and this entry is
  the record of why.
- Build info (`-X`-injected via ldflags: version, commit, build date) is consumed by
  `hasp version`; `runtime/debug.ReadBuildInfo()` (`func ReadBuildInfo() (info *BuildInfo, ok
  bool)`) is the fallback for a `go install`-built binary that never went through GoReleaser at
  all, so `hasp version` never reports "unknown" for a build that has *any* module information
  available.
- **Container caveat, stated plainly:** hasp manages the machine it runs *on* — §3.2 is explicit
  that hasp is *"not configuration management... does not reach across a fleet."* A container
  image is therefore for CI/scripted use, with the key directory bind-mounted in, and UID /
  `0700`-mode mapping across that boundary is the sharp edge: a mismatched UID inside the
  container silently produces a `~/.ssh` the host user cannot read. This is named here so it does
  not surface as a support question later.
