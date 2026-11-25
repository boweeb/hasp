---
id: decision-T31
title: 'Semantic versioning, and the compatibility surface v1.0.0 freezes'
date: '2026-09-02'
status: 'Accepted · Amends: T14, T29'
---
### Context

Four tags exist — `v0.1.0` through `v0.4.0` — and none of them rests on a stated compatibility
policy. [T14](decision-T14.md) already makes the `--json` envelope a contract, and [T29](decision-T29.md) already makes
a `check` finding's `id` permanent; a compatibility surface is therefore *implied* by decisions
already ratified, but it has never been written down as a surface in its own right — only as
promises scattered across individual entries. hasp cannot promise stability it has never
enumerated, and `design.md` §6.3 is explicit about why the promise exists at all: *"Every read has
a **machine-readable form**. hasp lives in a terminal beside other tools; a tool whose output can
only be looked at is half a tool."* [P6](../../docs/design.md#4-principles) is the other half of the
argument: nothing hasp knows may be trapped inside it, which is only true in practice if a
consumer built against hasp today can trust what still holds tomorrow.

### Decision

hasp adopts **Semantic Versioning**. **v1.0.0 is tagged at the close of
[M3.5](../../docs/roadmap.md#55-m35--hardening)** — the milestone that closes the SSH story and is the
natural point at which a compatibility surface can be frozen rather than merely described. M4/J9
then lands as **v1.1.0**, additive by construction: `tdd.md` §14 already constrains `Profile` to
stay additive across that boundary, which is precisely what a minor bump requires and a major one
would not.

The **public contract** — breaking any of the following needs a major version bump:

| # | Surface | Ratified by |
| --- | --- | --- |
| 1 | The four exit codes and their meanings, including that `1` stays `check`-exclusive | [T14](decision-T14.md) |
| 2 | The `--json` envelope's shape: `version`, `kind`, `data`, `warnings` | [T14](decision-T14.md) |
| 3 | `kind` strings (`key.list`, `host.show`, `check.report`, …) — permanent, retired rather than recycled, the same guarantee a `Dn`/`Tn` ID carries | [T14](decision-T14.md) |
| 4 | `check` finding `id`s | [T29](decision-T29.md) |
| 5 | The verb×noun grid and the global flag names and semantics — removing or renaming is breaking, adding is additive | [D10](decision-D10.md), `tdd.md` §9 |
| 6 | The on-disk marker syntax and metadata format — a change that makes an existing hasp-marked region unreadable by the new binary is breaking | `tdd.md` §6, §7; [T10](decision-T10.md), [T25](decision-T25.md) |

Row 6 is the least obvious and the most damaging of the six, because the artifact this promise
governs outlives the binary that wrote it: a `~/.ssh/config` marked in 2026 has to still parse
under whatever hasp binary someone runs against it in 2030.

The explicit **non-contract**, which matters as much as the contract does: human-readable output
([P7](../../docs/design.md#4-principles) governs it, and [T14](decision-T14.md) already permits the two renderers to
diverge in *form* — a script needing stability uses `--json`); finding `severity`
([T29](decision-T29.md) makes it deliberately re-tunable); `--verbose` stderr diagnostics; and the exact
filename format inside `~/.ssh/.hasp-backups/` ([T8](decision-T8.md) — [P6](../../docs/design.md#4-principles)
guarantees those backups stay legible and recoverable without hasp, not that their names never
change).

**The relationship between the two version numbers runs one way only.** [T14](decision-T14.md)'s own struct
comment describes `Envelope.Version` as "independent of hasp's own release version" — true, but a
phrasing that invites the wrong inference. Stated exactly: **a change to `Envelope.Version`
implies a major hasp version bump; a major hasp version bump does not imply a change to
`Envelope.Version`.** The envelope can stay stable across several major hasp releases; it cannot
change without one.

`hasp version`
([`tdd.md` §13](../../docs/tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought))
is the surface that makes all of this checkable at runtime rather than merely asserted in a
document, so this entry is where the command is finally documented as design — [T9](decision-T9.md)
mentioned it only as a build-info consumer.

**Release notes are generated, not hand-maintained.** GoReleaser's `changelog: use: git` output is
the changelog of record; there is no hand-maintained `CHANGELOG.md`, because a file that must be
remembered is a file that drifts, and the git history already is the truth.

**The historical tag mapping**, recorded here so it is not mysterious later: `v0.1.0` → M1,
`v0.2.0` → M2, `v0.3.0` → M3, `v0.4.0` → the GoReleaser/Gitea release-plumbing commit that
followed M3's merge.

### Rationale

A compatibility surface stated after the first breaking change is a postmortem, not a promise. Six
of the nine principles in `design.md` are already load-bearing on this surface without saying so —
enumerating it now, before v1.0.0, is the only order in which "breaking this needs a major bump"
means anything.

### Consequence

**The Go module path problem is now load-bearing, not cosmetic.** `go.mod` declares
`github.com/boweeb/hasp`; the only remote is a local Gitea (`git@localhost:stuff/hasp.git`), so
`go install github.com/boweeb/hasp/cmd/hasp@latest` cannot resolve today, and Go's own module
rules mean a future v2 requires a `/v2` path suffix regardless of what the current path resolves
to. **v1.0.0 freezes the module path** as surely as it freezes the six rows above, so this has to
be settled *before* the tag, not after — [`roadmap.md` §5.5](../../docs/roadmap.md#55-m35--hardening) carries
it as a named input-needed item rather than deciding it here, which is not this entry's place to
do.
