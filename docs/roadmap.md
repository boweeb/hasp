---
Status: APPROVED
DateCreated: 2026-08-29
DateApproved: 2026-08-29
DateLastReviewed: 2026-09-02
Related:
  - "[`docs/README.md`](README.md)"
  - "[`docs/design.md`](design.md)"
  - "[`docs/tdd.md`](tdd.md)"
  - "[`docs/decision-log.md`](decision-log.md)"
  - "[`docs/tech-decision-log.md`](tech-decision-log.md)"
---

# hasp — Roadmap

> **Scope of this document.** This is the project-management document, kept deliberately separate
> from [`docs/tdd.md`](tdd.md) so that neither contaminates the other. It carries **sequencing,
> scope boundaries, and exit criteria** — what gets built in what order, and how you know a
> milestone is finished. It contains **no technical design**: where a milestone needs a mechanism,
> this document cites the TDD section that specifies it rather than restating it.
>
> `design.md` §9 defines M1–M4 in user-visible terms and is upstream of this document. What is
> added here is the ordering, the work breakdown, and the tests each milestone must pass.

---

## 1. How this document sequences work

**By dependency, not by calendar.** There are no dates and no estimates in this document, and
their absence is deliberate rather than an oversight: this is one person's project, worked on in
the time that exists, and a date invented now would be a number nobody committed to. What *is*
committed to is the ordering — each milestone unblocks the next, and each one leaves the tool
more useful than it found it.

That last property is the discipline that was missing the first time. `design.md` §9 states it
directly: *"Each is independently valuable — the project should be worth using at the end of every
one."* The predecessor died mid-refactor with no working state to fall back to; every milestone
boundary here is a point where the tool works and could be abandoned without loss.

**M1 is the milestone to defend.** `design.md` §9: *"M1 is therefore the only milestone whose
scope should be defended aggressively."* Anything that can be argued into M2 should be.

### Status — where the project actually is (2026-09-02)

This document has sequenced work since it was written, but never once recorded which of that work
actually finished — a roadmap with no completion record is only half a roadmap. This reassessment
pass fixes that drift and gives it one, rather than adding it as a separate tracking document this
document's own §8 already forbids.

| Milestone | Status | Tag | Merge commit |
| --- | --- | --- | --- |
| §2 M0 — Bootstrap | Complete | *(no tag — folded into M1's early history)* | — |
| §3 M1 — Inventory | Complete | `v0.1.0` | `41a7a1a` |
| §4 M2 — Curation | Complete | `v0.2.0` | `d009473` |
| §5 M3 — Configuration | Complete | `v0.3.0` | `4577e17` |
| *(release plumbing, not a milestone)* | — | `v0.4.0` | `d7699c6` |
| §5.5 M3.5 — Hardening | Current | — | — |
| §6 M4 — Identity | Gated | — | — |

`v0.4.0` tags the commit that configured GoReleaser's Gitea publishing and added the repository's
`LICENSE` — release plumbing that followed M3's merge, not a milestone of its own, listed here
because it is the tag sitting immediately behind M3.5's own work.

---

## 2. M0 — Bootstrap

*Not a milestone in `design.md` §9.* It exists because M1–M4 all assume a repository with code in
it, and this one currently has only documentation. M0 delivers nothing user-visible and should be
finished in one sitting.

**Work**

- `go.mod`, `mise.toml` pinning the toolchain ([`tdd.md` §2](tdd.md)).
- The four-layer package skeleton — `cmd/hasp`, `internal/domain`, `internal/app`,
  `internal/adapter/*`, `internal/cli` ([`tdd.md` §3](tdd.md)).
- **The layering guard**, wired before there is anything to violate it: `go list -deps` asserting
  `internal/domain` imports nothing outside the standard library
  ([`T13`](tech-decision-log.md#t13)). Adding this after the fact means fixing violations instead
  of preventing them.
- Lint, `go vet`, and CI running both plus `go test ./...`.
- **The 28 key fixtures** and the dev-only script that generates them
  ([`tdd.md` §12](tdd.md)) — 24 across the algorithm × format × encryption × public-half matrix,
  plus 4 hand-constructed DSA fixtures. These gate M1's entire read path, so they come first.
- The CST shape fixtures: CRLF, no-trailing-newline, and a 0-byte file.
- A GoReleaser skeleton that builds the four targets ([`T9`](tech-decision-log.md#t9)) — not
  releasing anything yet, just proving `CGO_ENABLED=0` holds from the first commit.
- A repository `README.md` and `.gitignore` (neither exists yet; `.idea/` is currently untracked
  and unignored).

**Exit criteria**

- `go build ./...` and `go test ./...` pass in CI on a clean checkout.
- The layering guard fails when someone adds a third-party import to `internal/domain` — verified
  by trying it once, deliberately.
- All 28 key fixtures exist, are committed, and each parses to the algorithm, format, and
  encryption state its filename claims.

---

## 3. M1 — Inventory

> *hasp tells the truth about this machine.* Covers **J1** (Survey), **J2** (Identify),
> **J3** (Comprehend).

**The whole milestone is read-only.** Everything is unmanaged ([`D14`](decision-log.md#d14)), so
hasp never writes to `~/.ssh` at all. That safety is structural rather than a matter of care, and
it is the reason M1 is the right first milestone: the tool becomes genuinely useful to its author
before it has ever been trusted with a write.

**Work**

- The `keyfile` adapter: parse, classify, fingerprint, read the comment
  ([`T1`](tech-decision-log.md#t1), [`T23`](tech-decision-log.md#t23)) — including the derivation
  gap returning `unknown` rather than guessing.
- The `scan` → classify → resolve → project pipeline ([`tdd.md` §5](tdd.md)), with
  key identity by fingerprint-else-path ([`T12`](tech-decision-log.md#t12)).
- **The `sshconfig` CST, read side only** ([`T2`](tech-decision-log.md#t2)) — `Parse` plus
  `Render`, marked-region *recognition*, and `MarkerDefect` reporting. Note that `Render` is in
  scope even though nothing writes yet: `Render(Parse(b)) == b` is how `Parse` is proven lossless,
  and the fuzz test is the only thing standing between M3 and a P2 violation. Building it here
  means the contract is under test for an entire milestone before a write depends on it.
- Binding resolution: tokens, symlinks, relative paths ([`T28`](tech-decision-log.md#t28)), and
  implicit default-identity probing ([`T16`](tech-decision-log.md#t16)).
- Host profile derivation ([`T5`](tech-decision-log.md#t5)).
- `list`, `show`, `find` across all three nouns; `show profile` recursing by default
  ([`T21`](tech-decision-log.md#t21)).
- `check`, reporting only, with the finding schema fixed from the start
  ([`T29`](tech-decision-log.md#t29)).
- The output contract: human renderer, `--json` envelope, four exit codes
  ([`T14`](tech-decision-log.md#t14)).

**Explicitly out of scope**

Every verb that writes — `new`, `edit`, `adopt`, `release`. The `Plan` type and the `Applier`
(§4), backups, the settings file, and passphrase handling. If a task in M1 requires any of them,
it belongs to M2.

**Exit criteria**

1. Point hasp at the author's real `~/.ssh` — thirteen keypairs, two symlink aliases, mixed
   RSA/Ed25519, mixed PEM/OpenSSH — and `hasp list key` returns an accurate inventory **on one
   screen** (P7).
2. `hasp find key` identifies a key from a fingerprint fragment regardless of punctuation and
   case (J2).
3. `hasp show key <name>` lists every host that binds it, with explicit and implicit-default
   bindings **labelled distinctly** (`T16`).
4. **The write-nothing guard passes**: a pre/post snapshot of the entire temp `HOME` tree is
   byte-identical after the whole M1 suite runs ([`tdd.md` §12](tdd.md)). This is the mechanical
   proof of J1's promise.
5. The CST fuzz test runs clean in CI over the committed seed corpus.
6. `check` exits `1` with findings and `0` clean; the emitted finding ids match the golden list.
7. Every read has a `--json` form, and `hasp <any read> --json | jq .version` returns `1`.

---

## 4. M2 — Curation

> *hasp changes the arrangement safely.* Covers **J4** (Create), **J5** (Tidy), **J6** (Rearrange).

**Every write in the project arrives at once, and so does the machinery that makes writes safe.**
That is deliberate: `D6` requires preview to be uniform, so the first write and the tenth go
through the identical path.

**Work**

- The `Plan` / `Change` / `Applier` model ([`tdd.md` §4](tdd.md),
  [`T4`](tech-decision-log.md#t4)) with previews carrying real diffs
  ([`T26`](tech-decision-log.md#t26)).
- The safety mechanics ([`tdd.md` §11](tdd.md)): atomic write, symlink-through, mode preservation,
  D4's copy→verify→replace move rule, and the witness that catches the preview/apply race
  ([`T30`](tech-decision-log.md#t30)).
- `BackupStore` ([`T8`](tech-decision-log.md#t8)).
- `new key`, including passphrase handling ([`D17`](decision-log.md#d17),
  [`T6`](tech-decision-log.md#t6)) and the settings file that makes it non-interactive
  ([`D16`](decision-log.md#d16), [`T7`](tech-decision-log.md#t7)).
- `adopt` / `release` for **key** and **profile** — with `adopt key`'s alias-preserving move
  ([`T20`](tech-decision-log.md#t20)) and cross-profile membership via alias location
  ([`T19`](tech-decision-log.md#t19)).
- `edit key`: rename, alias, replace material, move between profiles.
- `check`'s repair paths for managed resources.

**Explicitly out of scope**

Anything touching a host stanza or a host-group file. `adopt host` and `release host` are M3, and
so is every `WriteRegion` change — M2 writes key files, symlinks, and `.hasp` markers only.

**Exit criteria**

1. `adopt` a key, then `release` it, and the key directory is **byte-identical to where it
   started** — the two-way door proven, not asserted ([`D14`](decision-log.md#d14)).
2. `hasp new key`'s reported facts are **identical** to what a fresh `hasp show key` of the
   finished artifact reports (J4's actual requirement).
3. Every write previews first; with no TTY and no `--yes`, every write fails closed.
4. A backup exists for every change that modified something hasp did not create (P4).
5. The injected-failure test: interrupt `adopt` at each step and assert a complete, readable key
   remains at its original path every time ([`tdd.md` §12](tdd.md)).
6. The settings field-set test passes, and `~/.config/hasp/settings.toml` is byte-identical before
   and after the entire suite — hasp reads it and never writes it.

---

## 5. M3 — Configuration

> *hasp edits SSH config without ever losing a hand edit.* Covers **J7**, and makes **J8**
> (Offboard) answerable.

**Work**

- `WriteRegion` and rigid regeneration ([`tdd.md` §6](tdd.md)).
- Marker syntax for both cases: a wholly-owned group file, and marked regions inside the co-owned
  `~/.ssh/config` ([`D7`](decision-log.md#d7)).
- Host-group composition and `Include` ordering ([`T11`](tech-decision-log.md#t11)), plus the
  shadowed-stanza finding that falls out of it.
- `new host`, `edit host`, `adopt host`, and `release host` — the last one content-preserving per
  [`D18`](decision-log.md#d18).
- The metadata channel ([`T10`](tech-decision-log.md#t10),
  [`T25`](tech-decision-log.md#t25)). The mechanism is built because `D7` requires it be
  available; **its keyset is empty**, and adding a field to it needs its own decision.
- `show profile` completing J8's offboarding answer across keys and hosts.

**Exit criteria**

1. Take a real `~/.ssh/config` with hand comments, odd whitespace, and unrecognized directives;
   have hasp add a host through it; and assert **every byte outside hasp's markers is unchanged**
   (P2, J7). This is the single most important test in the project.
2. `adopt host` then `release host` returns the stanza to plain text with its content intact
   (`D18`).
3. A symlinked `~/.ssh/config` is written **through**, and the symlink's target inode is unchanged
   ([`T15`](tech-decision-log.md#t15)).
4. A file with a malformed marker pair is **readable** (findings reported, defects flagged by kind
   and line) and **unwritable** (fail-closed) — both stances in one fixture
   ([`T18`](tech-decision-log.md#t18)).
5. J8 answered end to end: `hasp show profile work` names every key and every host that would
   break if that profile went away.

---

## 5.5. M3.5 — Hardening

The `.5` is deliberate, not a placeholder. §2 through §5 above are cited directly by roughly
twenty Go source comments — `roadmap.md §2` for M0, `§3` for M1, `§4` for M2, `§5` for M3 — and
renumbering any of them to make room for a new milestone would break every one of those citations
invisibly, with no compiler error to catch it. This section sits at `§5.5` for exactly that
reason, and §6 through §8 keep the numbers they already had.

> *hasp is something someone else could install, trust, and walk away from.* Covers no new
> journey — it makes **J1** (Survey) survivable by a stranger, and discharges **P6**.

**Work**

- **Versioning** — the SemVer commitment and the compatibility surface it freezes
  ([`tdd.md` §16](tdd.md#16-versioning-and-the-compatibility-surface),
  [T31](tech-decision-log.md#t31)).
- **CI/CD** — the Mage target set and the thin-shim rule
  ([`tdd.md` §17](tdd.md#17-continuous-integration-and-release-automation),
  [T32](tech-decision-log.md#t32)).
- **Distribution** — archives, generated completions and man pages, and an SBOM shipping now; the
  four publicly-origin-blocked channels staying deferred
  ([`tdd.md` §13](tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought),
  [T33](tech-decision-log.md#t33)).
- **User-facing documentation** — the generated/hand-written split
  ([T34](tech-decision-log.md#t34)).
- **Doc hygiene** — the stale claims a new reader trips over first: the root `README.md`'s
  "M1" / "Writes … arrive in M2" status section; `docs/README.md`'s closing claim that no root
  `README.md` exists yet; `.goreleaser.yaml`'s header comment claiming M0 scope and that nothing
  is being released; and two passages inside `docs/tech-decision-log.md` that have quietly drifted
  from current truth — [T20](tech-decision-log.md#t20)'s worked-example ordering for `adopt key`
  and [T22](tech-decision-log.md#t22)'s `Consequence` text describing
  `edit key --replace-material` — both superseded by `tdd.md`'s later reasoning. Per this log's
  own append-only rule, neither gets edited in place; each needs a new entry naming what it
  amends. Also there: `docs/tech-decision-log.md`'s declared status vocabulary (`Accepted`,
  `Open`, `Amended by Tn`, `Superseded by Tn`, `Rejected`) no longer matches what the index
  actually uses — `extended by`, `refined by`, `ratified upstream as` and `staged by` all appear
  — so either the declaration or the usage should be reconciled.
- **Key inspection** — the input-needed item below. Scoping it needs an answer this document does
  not have, so it is carried by exit criterion 8 rather than by a criterion of its own: the
  milestone may close with the question **answered** or **explicitly deferred**, but not with it
  merely unaddressed.

**Explicitly out of scope**

Anything touching J9 or M4. The four distribution channels [T33](tech-decision-log.md#t33)
defers. Multi-directory / non-default key locations — `design.md` §10 leaves them deferred and
this milestone does not reopen them.

**Exit criteria**

1. `v1.0.0` is tagged, and the release it produces carries archives for all four targets plus
   generated completions, man pages, and an SBOM.
2. A clean checkout passes `go run mage.go ci` with no platform-specific step, and that same
   target is what both workflow shims invoke.
3. The darwin targets compile in CI.
4. The doc-verification target passes with zero broken links, zero unresolved `Dn`/`Tn`
   citations, and matching log index/anchor counts — and fails the build when it does not.
5. The committed CLI reference, man pages, and completions are byte-identical to freshly
   generated output; CI fails if they are stale.
6. **The clean-room test passes**: in a container with no repository checkout, following only the
   published `README.md`'s install steps, `hasp list key` returns an accurate inventory of a
   synthetic `~/.ssh` fixture, `hasp adopt` then `hasp release` returns that fixture
   **byte-identical** to where it started, and the whole run is covered by §3's write-nothing
   snapshot guard. This is the mechanical form of "a stranger can install it, trust it, and walk
   away from it" — the milestone's own one-line claim — and it is deliberately a script rather
   than a judgement, because every other exit criterion in this document is.
7. `golangci-lint` and the Go toolchain are both pinned to exact versions.
8. **Both input-needed items below are closed** — each either answered and recorded in its own
   `Tn` entry, or explicitly deferred by a recorded decision. Neither may close by being
   forgotten. The public-origin question is additionally gated by criterion 1, since tagging
   `v1.0.0` freezes the module path ([T31](tech-decision-log.md#t31)).

> **⚠ INPUT NEEDED — SSH key inspection detail.**
>
> What hasp reports today (`tdd.md` §9): `list key` gives name, algorithm,
> fingerprint-or-`unknown`, format, encrypted?, profiles, and comment. `show key` adds every
> location/alias, profiles, and the hosts that bind it.
>
> Candidate additions below are *prompts, not decisions* — this milestone does not pick among
> them: RSA modulus size and ECDSA curve; a second fingerprint form (MD5 alongside SHA256) for
> consoles that still print it; the OpenSSH-format KDF/cipher and rounds on an encrypted key; file
> mode and mtime; whether the public half actually matches the private; certificate detection for
> `*-cert.pub`; and cross-referencing `authorized_keys`/`known_hosts`.
>
> Any answer must satisfy: **P7** (the default read stays one screen, so new detail likely
> belongs behind `show` or a flag rather than in `list`); **P1** (every added fact is derived at
> read time, never stored); **§5.1's derivation gap** (any new fact needs an honest `unknown`);
> and **T31's contract** (a new field in `--json` is additive, changing an existing one is not).
> The answer needs its own `T` entry.

> **⚠ INPUT NEEDED — a publicly reachable origin.**
>
> Whether hasp gets one — a GitHub or GitLab mirror, or a public Gitea — settles the Go module
> path (frozen the moment v1.0.0 tags, [T31](tech-decision-log.md#t31)), `go install`, and all
> four distribution channels [T33](tech-decision-log.md#t33) defers, all at once.
> **Recommendation, offered rather than decided:** if the long-term CI target is GitHub or GitLab
> anyway, a mirror is what makes the existing `github.com/boweeb/hasp` module path honest rather
> than aspirational.

---

## 6. M4 — Identity

> *A profile answers "who am I being" across SSH, Git, and signing.* Covers **J9**.

**Not before M3 is finished, and this is a hard gate.** [`D8`](decision-log.md#d8) is explicit
that no near-term milestone may include it, and
[`docs/project-assessment-2026-08.md`](project-assessment-2026-08.md) documents this project
having already died once from opening a second front before the first was done.

**[M3.5 — Hardening](#55-m35--hardening) now sits between M3 and this milestone.** The gate above
is unweakened by its insertion — M3.5 touches no J9 work and exists precisely to keep this
milestone from starting early, not to soften the wait for it. When M4 does land, it ships as
**v1.1.0**, additive over the v1.0.0 surface M3.5 freezes
([T31](tech-decision-log.md#t31)).

No work breakdown is written here on purpose. `tdd.md` §14 states only the constraints M4 must not
foreclose — that `Profile` stays additive, that P9 governs Git/GPG taxonomy exactly as it governs
SSH, and that `D16`'s admission rule survives J9's arrival unweakened. Designing further than that
now would be the failure mode this gate exists to prevent.

---

## 7. Risks

| Risk | Where it bites | What holds it |
| --- | --- | --- |
| **The round-trip contract fails on a real config** — P2 is the promise the whole project is judged on, and the co-owned `~/.ssh/config` is where it is hardest | M3 | The CST is built and fuzzed in **M1**, a full milestone before anything writes through it. If it cannot hold, that is discovered while it is still cheap |
| **`adopt` loses a key file** — the most invasive operation, and the only one that moves key material | M2 | D4's copy→verify→replace ordering ([`T20`](tech-decision-log.md#t20)), plus an injected-failure test at every step |
| **The derivation gap surprises someone** — an undecidable key cannot be deduplicated, ever | M1 | It is reported as its own finding rather than hidden; `unknown` is a first-class answer ([`T12`](tech-decision-log.md#t12)) |
| **Scope creep toward J9** — the documented, already-experienced failure mode | Any | `D8`, and M4's hard gate above |
| **`settings.toml` grows into the old state file** — it has exactly that shape | M2 onward | `D16`'s admission rule, enforced by the field-set test rather than by review discipline |
| **The docs and the code drift apart** — the thing that killed the predecessor's design | Any | Every `Dn`/`Tn` ID is stable and citable from code comments and commit messages; a change contradicting one is supposed to be caught by that citation failing to make sense |
| **The compatibility surface freezes at v1.0.0 while the module path is still unsettled** — a problem that becomes unfixable-after-the-fact the moment the tag is cut | The v1.0.0 tag ([M3.5](#55-m35--hardening)) | [T31](tech-decision-log.md#t31) makes it a named prerequisite, tracked as an input-needed item in §5.5 rather than left implicit |
| **CI logic drifts back into a platform's YAML** — the failure mode the Mage shim exists to prevent | Whenever a new platform is added | [T32](tech-decision-log.md#t32)'s thin-shim rule and the single `CI` aggregate target |
| **Generated documentation goes stale** — a CLI reference, man pages, or completions that silently disagree with the binary | Continuously, every time a flag or command changes | [T34](tech-decision-log.md#t34)'s CI staleness check |

---

## 8. What this document is not

- **Not a schedule.** No dates, no estimates. See §1.
- **Not technical design.** Every mechanism above is a citation, not a specification. If this
  document and [`docs/tdd.md`](tdd.md) ever disagree about *how* something works, the TDD is
  right and this file has a bug.
- **Not a backlog.** Task-level tracking, if it ever exists, lives outside the documentation set.
