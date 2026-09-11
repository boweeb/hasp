---
Status: APPROVED
DateCreated: 2026-08-29
DateApproved: 2026-08-29
DateLastReviewed: 2026-09-04
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

### Status — where the project actually is (2026-09-04)

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
| §5.5 M3.5 — Hardening | Complete | `v0.6.0` | `9243f1a` |
| §5.6 M3.6 — Investigation | Current — chunks 0–5 landed, unmerged; `v1.0.0` not yet cut | — | — |
| §6 M4 — Identity | Gated | — | — |

`v0.4.0` tags the commit that configured GoReleaser's Gitea publishing and added the repository's
`LICENSE` — release plumbing that followed M3's merge, not a milestone of its own, listed here
because it is the tag sitting immediately behind M3.5's own work. That Gitea publishing target was
since dropped in favour of GitHub — `origin` is now `git@github.com:boweeb/hasp.git`
([T40](tech-decision-log.md#t40)) — so a reader should not take this note as meaning Gitea is
still live. This reassessment split what had been a single M3.5 into [§5.5](#55-m35--hardening)
and [§5.6](#56-m36--investigation) — see the new milestone's own opening note for why the split,
and why the section number it landed on.

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
- **Distribution** — archives, generated completions and man pages, an SBOM, cosign keyless
  signing (`signs`), and a `ko`-published container image all shipping now
  ([`tdd.md` §13](tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought),
  [T33](tech-decision-log.md#t33), [T40](tech-decision-log.md#t40)); `aur` and `homebrew_casks`
  staying deferred, each pending a separate repository the author must create and maintain.
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
- **SSH key inspection** — no longer scoped here. What began as an input-needed question was
  answered from an unexpected direction, downstream in [§5.6](#56-m36--investigation): rather than
  adding fields to `list key`/`show key`'s plain output, `--investigate`
  ([`tdd.md` §18](tdd.md#18-investigation--schemes-confidence-and-gated-derivation),
  [T35](tech-decision-log.md#t35)–[T39](tech-decision-log.md#t39)) answers a superset of the
  candidate additions this bullet used to carry, behind a flag rather than in the default read.

**Chunks**

Sequenced by dependency. **Status** carries the landing commit once a chunk lands — anchored like
§1's table, so this stays a completion record and not the task tracker §8 rules out.

| # | Chunk | After | Merge Commit |
| --- | --- | --- | --- |
| M3.5.1 | Build contract — `magefiles/`, the `mage.go` bootstrap, the target set, pinned `golangci-lint` and Go toolchain | — | `957ce9b` |
| M3.5.2 | CI shim — the workflow invokes one Mage target and nothing else; darwin targets compile | 1 | `c61de03` |
| M3.5.3 | Doc verification — the `Docs` target: links, anchors, `Dn`/`Tn` citations, log invariants, verbatim quotations | 1 | `572d97f` |
| M3.5.4 | Generated reference — man pages, shell completions, CLI markdown, and the staleness check that keeps them honest | 1 | `1bfff84` |
| M3.5.5 | Release pipeline — tag-triggered; archives, completions, man pages, `sboms`, `signs`, `ko`; build info stamped so `hasp version` reports it | 1, 4 | `b3b4ce6` |
| M3.5.6 | Narrative docs — root `README.md`, the usage guide by journey, what hasp leaves on disk, `SECURITY.md` | — | `93d1f8a` |
| M3.5.7 | Close-out — stale claims; the default branch (independent of everything, and needed *before* criterion 6's clean-room run); the SemVer policy honoured at this milestone's own tag (`v1.0.0` and the surface freeze are [§5.6](#56-m36--investigation)'s) | all | `9243f1a` |

**Explicitly out of scope**

Anything touching J9 or M4. `aur` and `homebrew_casks` — [T33](tech-decision-log.md#t33),
[T40](tech-decision-log.md#t40) — deferred pending repositories the author has not yet chosen to
create and maintain. Multi-directory / non-default key locations — `design.md` §10 leaves them
deferred and this milestone does not reopen them. **Investigation**
([§5.6](#56-m36--investigation)) — this milestone hardens what M1–M3 already built; it does not
extend the read surface.

**Exit criteria**

1. A release is tagged, and it carries archives for all four targets plus generated completions,
   man pages, an SBOM, a signature, and a container image. **`v1.0.0` itself is not claimed
   here** — it is reserved for [M3.6](#56-m36--investigation)'s close, per `design.md` §9, once
   the investigation surface is frozen alongside everything this milestone hardens.
2. A clean checkout passes `go run mage.go ci` with no platform-specific step, and that same
   target is what the workflow shim invokes — there is one shim, GitHub Actions, since
   [T40](tech-decision-log.md#t40) dropped Gitea.
3. The darwin targets compile in CI.
4. The doc-verification target passes with zero broken links, zero unresolved `Dn`/`Tn`
   citations, and matching log index/anchor counts — and fails the build when it does not.
5. The committed CLI reference, man pages, and completions are byte-identical to freshly
   generated output; CI fails if they are stale.
6. **The clean-room test passes**: in an isolated environment, installing the `hasp` binary under
   test exclusively via the published `README.md`'s `go install` step against the public module
   proxy — never built from a local checkout ([T45](tech-decision-log.md#t45)) — `hasp list key`
   returns an accurate inventory of a synthetic `~/.ssh` fixture, `hasp adopt` then `hasp release`
   returns that fixture **byte-identical** to where it started, and the whole run is covered by
   §3's write-nothing snapshot guard. This is the mechanical form of "a stranger can install it,
   trust it, and walk away from it" — the milestone's own one-line claim — and it is deliberately
   a script rather than a judgement, because every other exit criterion in this document is.
7. `golangci-lint` and the Go toolchain are both pinned to exact versions.
8. **The repository a stranger actually lands on is this project.** GitHub's default branch is
   `main`, and the archived Python predecessor's `master` history has been moved to cold storage
   and removed. This is criterion 6's hidden precondition rather than a cosmetic tidy: a clean-room
   run that clones the default branch gets the predecessor, not hasp, and criterion 6 would fail
   for a reason no amount of `README.md` work could fix
   ([T40](tech-decision-log.md#t40)).

> **Settled.** `origin` is `git@github.com:boweeb/hasp.git` ([T40](tech-decision-log.md#t40)) —
> the Go module path, `go install`, and all four distribution channels
> [T33](tech-decision-log.md#t33) named unblocked at once, exactly as predicted. `signs` and `ko`
> join this milestone's shipping set; `aur` and `homebrew_casks` stay deferred for a different
> reason — see **Explicitly out of scope** above. M3.5 carries no open input-needed item.

---

## 5.6. M3.6 — Investigation

Sits at `§5.6`, immediately after [M3.5](#55-m35--hardening) and before `§6`, for the same reason
M3.5 sits at `§5.5`: §2 through §5 are cited by roughly twenty Go source comments, and renumbering
any of them would break every one of those citations invisibly.

> *hasp answers a fingerprint held in hand, computed under any scheme, and says how sure it is.*
> Covers **J10** (Investigate) and makes **J2** (Identify) honest — matching a clue across every
> scheme hasp knows, not only the one its own tooling natively produces.

**Work**

- **P3's split landing in code** ([D19](decision-log.md#d19)) — the consent-gated access rule
  generalized from `new key`'s passphrase prompt to any read that explicitly asks.
- **The fingerprint scheme registry** ([`tdd.md` §18](tdd.md#18-investigation--schemes-confidence-and-gated-derivation),
  [T35](tech-decision-log.md#t35)) — open, pure-Go, no subprocess, no network.
- **Multi-scheme `find`** ([T36](tech-decision-log.md#t36)) — a clue's shape routes the search
  across every registered scheme, not only the SSH-native one.
- **The confidence vocabulary, across both renderers** ([T37](tech-decision-log.md#t37)) —
  `derived`/`confirmed`/`possible`/`unknown`, closed and permanent, in both the human and `--json`
  output.
- **The `ssh-agent` source** ([T38](tech-decision-log.md#t38)) — a loaded key's public half and
  comment, labelled `agent-sourced`, with no passphrase and no secret handling.
- **Passphrase-gated derivation** ([T39](tech-decision-log.md#t39)) — explicit, lazy, one
  passphrase per invocation, degrading rather than failing closed with no TTY.
- **`--investigate` on `show key` and `list key`** ([`tdd.md` §9](tdd.md#9-command-surface--the-38-grid-spelled-out)) —
  opt-in, out of the default read path entirely.

**Chunks**

Sequenced by dependency, the same way [M3.5](#55-m35--hardening)'s table is. **Merge Commit**
carries the landing commit once a chunk lands, so this stays a completion record rather than the
task tracker [§8](#8-what-this-document-is-not) rules out.

**A header note, not a silent workaround:** unlike M3.5's table, whose commits are already on
`main`, chunks 0–5 below landed across two still-unmerged branches, not one. Chunks 0–3 landed on
`m36-investigation`; chunks 4–5 landed on `m36-closeout`, a worktree branched from
`m36-investigation`'s own HEAD (`f37a474`) to finish this milestone's remaining close-out work
without disturbing the branch chunks 0–3 already occupy. `m36-investigation` itself remains
unmerged into `main` as of this writing. "Merge Commit" is therefore not yet literally accurate for
any row in this table, and this table does not invent a merge that has not happened. The hashes
recorded are each chunk's own landing commit on whichever of the two branches it landed on; the
column keeps its name for consistency with M3.5's own header until both branches actually merge,
at which point the column is accurate again without needing a rename.

| # | Chunk | After | Merge Commit |
| --- | --- | --- | --- |
| M3.6.0 | Default-read baseline — the committed capture of `list key`'s pre-M3.6 output that criterion 7 is asserted against, taken before any behaviour changes and therefore first | — | `551c93f` |
| M3.6.1 | Vocabulary and registry — P10's closed confidence set and the `origins` shape; the four schemes and each one's declared material requirement; the committed AWS and legacy vectors, the MD5-collision assertion, the confidence golden list, and the no-network guard | 0 | `e07bd40` (fix: `0f99139`) |
| M3.6.2 | Derivation sources — the `ssh-agent` client and its fake-agent test; passphrase-gated material opening, one prompt per invocation, and the no-TTY degrade | 1 | `5a1b48a` (fixes: `5b4563b`) |
| M3.6.3 | Multi-scheme `find key` — shape routing, both MD5 candidates computed rather than one guessed, and confidence-graded match evidence in both renderers | 1 | `61877d7` (fixes: `ad842a3`) |
| M3.6.4 | `--investigate` on `show key` and `list key` — the projection assembled across every source, both renderers, and the guard that the default read is untouched. **Owes the "agent tried first" ordering** ([T39](tech-decision-log.md#t39), [T38](tech-decision-log.md#t38)): `internal/app.PassphraseGate` cannot enforce it itself (`tdd.md` §18's `ssh-agent` subsection), so this chunk's own call site is where the agent lookup must run, and its result merge into a candidate key's material, before `PassphraseGate.Derive` is ever called for that key | 2, 3 | `9e51dc8` |
| M3.6.5 | Close-out — the regenerated reference surface, narrative documentation, this table's own completion record, and the `v1.0.0` tag criterion 8 reserves for it | all | `4c2ac5d` |

**Explicitly out of scope**

The keyring-backed passphrase cache — considered and rejected for now (`design.md` §10); the
two rungs this milestone ships (agent, explicit prompt) are expected to suffice without one.
Anything touching J9 or M4.

**Exit criteria**

1. A committed fingerprint under each of the **four** registered schemes is matched by
   `hasp find key` to the correct fixture key, and the reported origin's `confidence` is
   `confirmed` for the two AWS RSA schemes and `possible` for both SSH-native/ED25519 and legacy
   SSH MD5.
   1a. **The MD5 collision resolves both ways**: the same fixture's imported-RSA and legacy SSH
   MD5 fingerprints are different values of identical shape, and a 47-character clue matching
   either one finds the key. A regression that collapses the MD5 shape to a single scheme must
   fail this criterion rather than silently miss every legacy clue
   ([T36](tech-decision-log.md#t36)).
2. `hasp show key --investigate --json` emits every registered scheme's value or an explicit
   `unknown` with a machine-readable reason; no scheme is silently omitted.
3. The confidence vocabulary matches its golden list, and no value outside the closed set is
   emitted anywhere.
4. With a key loaded into a fake `ssh-agent`, the comment of an OpenSSH-format **encrypted** key is
   reported and labelled agent-sourced; with `SSH_AUTH_SOCK` unset the same command degrades to
   `unknown` and exits `0`.
5. `hasp show key --investigate --json </dev/null` with no TTY exits `0`, reports what is
   derivable, and marks the rest `unknown` with `passphrase-required-no-tty`.
6. The no-network guard passes: no scheme opens a socket.
7. `hasp list key` **without** `--investigate` never diverges because `--investigate` exists — the
   investigation mode never leaks into the default read (P7). Originally stated, and asserted, as
   byte-identical to the literal pre-M3.6 golden; [T52](tech-decision-log.md#t52)'s structural fix
   (`render.marshalData`'s recursive nil-slice normalization, landed on this same branch before the
   `v1.0.0` tag, while correcting structure is still free per the maintainer's own stated stance)
   deliberately regenerated that golden — `"profiles": null` becomes `"profiles": []` at both
   occurrences, the JSON form only, the human form byte-for-byte unchanged. The mechanism is
   therefore byte-identity against the T52-corrected golden, not literally the pre-M3.6 one; the
   criterion's *intent* — investigation mode never leaks into the default read — is untouched and
   still asserted the same way it always was, by `TestDefaultRead_ListKey_Golden`'s byte-for-byte
   comparison plus its companion below.
8. **`v1.0.0` is tagged at this milestone's close** — not M3.5's — once criterion 7 above confirms
   the default read is untouched and [M3.5](#55-m35--hardening)'s own exit criteria have already
   passed.

> **Close-out, chunk M3.6.5.** Each of the eight exit criteria above, accounted for:
>
> - **1** and **1a** (all four schemes matched by `find key`, correct confidence per scheme; the
>   MD5 collision resolves both ways) were discharged by chunk M3.6.3
>   ([T36](tech-decision-log.md#t36)) and re-verified unregressed when chunk M3.6.4 landed
>   (`9e51dc8`'s own commit message says so directly).
> - **6** (the no-network guard) was discharged by [T47](tech-decision-log.md#t47), ahead of this
>   milestone's own investigation surface existing to test, and was likewise re-verified
>   unregressed by M3.6.4.
> - **2**, **3**, **4**, **5**, and **7** are each covered by a test named for the criterion it
>   proves, landed in chunk M3.6.4 (`internal/app/investigate_test.go`,
>   `internal/cli/investigate_test.go`, `internal/cli/defaultread_golden_test.go`) — criterion 7 in
>   particular is the default-read golden guard chunk M3.6.0 committed before any
>   `--investigate` code existed, so "untouched" is asserted against a baseline captured before the
>   feature, not after it.
> - **Criterion 7's golden was deliberately regenerated after this chunk landed**, by a
>   structural-correctness fix ([T52](tech-decision-log.md#t52)) applied on this same branch ahead
>   of the `v1.0.0` tag: `render.marshalData` now normalizes a nil slice to `[]` at every nesting
>   depth it can reach, not only at the top-level `data` value, closing a gap T51's own Consequence
>   named and deferred (`"profiles": null`, `"hosts": null`). `list-key.json.golden` changed —
>   `"profiles": null` → `"profiles": []`, both occurrences — and `list-key.human.golden` did not
>   change at all. This is not a silent weakening of criterion 7: its mechanism (byte-for-byte
>   comparison) and its proof that `--investigate` is not a no-op
>   (`TestDefaultRead_ListKey_Investigate_DiffersFromGolden`) are both unchanged; only the
>   golden's own bytes moved, once, deliberately, to match corrected structure — exactly the
>   maintainer's own stated stance for everything landing before this tag, and exactly the kind of
>   change the byte-frozen-artifact discipline this same criterion's *language* borrows from
>   begins enforcing only **at** the tag, not before it.
> - **8 is explicitly NOT discharged by this chunk, or by this milestone's own work at all.** The
>   `v1.0.0` tag is the maintainer's to cut, by hand, after `m36-investigation` (carrying chunks
>   0–3) and `m36-closeout` (carrying chunks 4–5) both merge to `main` — this document does not
>   tag itself, and no commit in this milestone claims otherwise. Everything above this line is
>   ready for that merge; the merge and the tag are not part of what M3.6.5 discharges.

---

## 6. M4 — Identity

> *A profile answers "who am I being" across SSH, Git, and signing.* Covers **J9**.

**Not before M3 is finished, and this is a hard gate.** [`D8`](decision-log.md#d8) is explicit
that no near-term milestone may include it, and
[`docs/project-assessment-2026-08.md`](project-assessment-2026-08.md) documents this project
having already died once from opening a second front before the first was done.

**[M3.5 — Hardening](#55-m35--hardening) and [M3.6 — Investigation](#56-m36--investigation) now
sit between M3 and this milestone.** The gate above is unweakened by their insertion — neither
touches J9 work, and both exist precisely to keep this milestone from starting early, not to
soften the wait for it. When M4 does land, it ships as **v1.1.0**, additive over the v1.0.0
surface M3.6 freezes at its close ([T31](tech-decision-log.md#t31),
[T37](tech-decision-log.md#t37)).

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
| **The compatibility surface freezes at v1.0.0 while the module path was still unsettled** — a problem that would have become unfixable-after-the-fact the moment the tag was cut | The v1.0.0 tag ([M3.6](#56-m36--investigation)) | Resolved. [T31](tech-decision-log.md#t31) named it a prerequisite instead of leaving it implicit; [T40](tech-decision-log.md#t40) settled it — `origin` moved to `git@github.com:boweeb/hasp.git`, a milestone ahead of the tag it gated, not at the last moment |
| **CI logic drifts back into a platform's YAML** — the failure mode the Mage shim exists to prevent | Whenever a new platform is added | [T32](tech-decision-log.md#t32)'s thin-shim rule and the single `CI` aggregate target |
| **Generated documentation goes stale** — a CLI reference, man pages, or completions that silently disagree with the binary | Continuously, every time a flag or command changes | [T34](tech-decision-log.md#t34)'s CI staleness check |
| **Investigation scope grows past what J10 actually needs** — an open-ended registry and an opt-in mode are both natural places for "just one more candidate field" to accrete | [M3.6](#56-m36--investigation) | Held by that milestone's own exit criteria (a fixed set of fixture-backed assertions, not a growing wishlist) and by the keyring rung staying explicitly deferred (`design.md` §10) |
| **The confidence vocabulary leaks a fifth value, or one of the four gets silently redefined** — the exact failure P10's closed-set promise exists to prevent | [M3.6](#56-m36--investigation) onward | The golden-list guard ([T37](tech-decision-log.md#t37), [`tdd.md` §12](tdd.md#12-testing-strategy--a-function-of-a-directory)) — the same mechanism already used for `check`'s finding `id`s and the settings field set |

---

## 8. What this document is not

- **Not a schedule.** No dates, no estimates. See §1.
- **Not technical design.** Every mechanism above is a citation, not a specification. If this
  document and [`docs/tdd.md`](tdd.md) ever disagree about *how* something works, the TDD is
  right and this file has a bug.
- **Not a backlog.** Task-level tracking, if it ever exists, lives outside the documentation set.
