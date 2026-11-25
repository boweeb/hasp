---
Status: APPROVED
DateCreated: 2026-08-29
DateApproved: 2026-08-29
DateLastReviewed: 2026-11-25
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
> scope boundaries, and the reasoning behind them** — what gets built in what order, and why. It
> contains **no technical design**: where a milestone needs a mechanism, this document cites the
> TDD section that specifies it rather than restating it. Per-milestone work breakdowns and exit
> criteria are tasks, and live in `backlog/` ([T56](../backlog/decisions/decision-T56.md)).
>
> `design.md` §9 defines M1–M4 in user-visible terms and is upstream of this document. What is
> added here is the ordering and the scope boundaries; what each milestone must pass is in its
> backlog tasks.

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

### Status — where the project actually is

Progress is tracked in the backlog: `backlog milestone list` shows each milestone's completion, and every completed milestone's tasks carry their landing commits and tags ([T56](../backlog/decisions/decision-T56.md)). What stays here is only the commit-anchored record below — each row is anchored to a tag or a commit in this repository's history, never to free-text task state, so it cannot drift on its own.

| Milestone | Status | Tag | Landing commit |
| --- | --- | --- | --- |
| §2 M0 — Bootstrap | Complete | *(no tag — folded into M1's early history)* | `04d6b8d` |
| §3 M1 — Inventory | Complete | `v0.1.0` | `c595bc2` |
| §4 M2 — Curation | Complete | `v0.2.0` | `cd2abce` |
| §5 M3 — Configuration | Complete | `v0.3.0` | `8b13b0a` |
| *(release plumbing, not a milestone)* | — | `v0.4.0` | `f99a813` |
| §5.5 M3.5 — Hardening | Complete | `v0.6.0` | `65e2866` |
| §5.6 M3.6 — Investigation | Merged to `main`; `v1.0.0` not yet cut | `v1.0.0-rc.1` | `6e4c138` |
| *(release candidate, not a milestone)* | — | `v1.0.0-rc.2` | `5a1cb76` |
| §6 M4 — Identity | Gated | — | — |

The commits above are the ones present in this repository's history. Earlier revisions of this table and of the M3.5 chunk table recorded merge hashes (for example `41a7a1a`, `d009473`, `957ce9b`) that do not exist in it; they were replaced with tag-derived and first-parent-merge hashes, not lost ([T56](../backlog/decisions/decision-T56.md)).

`v0.4.0` tags the commit that configured GoReleaser's Gitea publishing and added the repository's
`LICENSE` — release plumbing that followed M3's merge, not a milestone of its own, listed here
because it is the tag sitting immediately behind M3.5's own work. That Gitea publishing target was
since dropped in favour of GitHub — `origin` is now `git@github.com:boweeb/hasp.git`
([T40](../backlog/decisions/decision-T40.md)) — so a reader should not take this note as meaning Gitea is
still live. This reassessment split what had been a single M3.5 into [§5.5](#55-m35--hardening)
and [§5.6](#56-m36--investigation) — see the new milestone's own opening note for why the split,
and why the section number it landed on.

---

## 2. M0 — Bootstrap

*Not a milestone in `design.md` §9.* It exists because M1–M4 all assume a repository with code in
it, and this one currently has only documentation. M0 delivers nothing user-visible and should be
finished in one sitting.

**Work breakdown and exit criteria** live in the backlog, not here ([T56](../backlog/decisions/decision-T56.md)): the `M0 Bootstrap` milestone and its tasks (`backlog milestone list`, `backlog task list --plain -m "M0 Bootstrap"`). This section keeps the scope narrative and the boundaries.

---

## 3. M1 — Inventory

> *hasp tells the truth about this machine.* Covers **J1** (Survey), **J2** (Identify),
> **J3** (Comprehend).

**The whole milestone is read-only.** Everything is unmanaged ([`D14`](../backlog/decisions/decision-D14.md)), so
hasp never writes to `~/.ssh` at all. That safety is structural rather than a matter of care, and
it is the reason M1 is the right first milestone: the tool becomes genuinely useful to its author
before it has ever been trusted with a write.

**Work breakdown and exit criteria** live in the backlog, not here ([T56](../backlog/decisions/decision-T56.md)): the `M1 Inventory` milestone and its tasks (`backlog milestone list`, `backlog task list --plain -m "M1 Inventory"`). This section keeps the scope narrative and the boundaries.

**Explicitly out of scope**

Every verb that writes — `new`, `edit`, `adopt`, `release`. The `Plan` type and the `Applier`
(§4), backups, the settings file, and passphrase handling. If a task in M1 requires any of them,
it belongs to M2.

---

## 4. M2 — Curation

> *hasp changes the arrangement safely.* Covers **J4** (Create), **J5** (Tidy), **J6** (Rearrange).

**Every write in the project arrives at once, and so does the machinery that makes writes safe.**
That is deliberate: `D6` requires preview to be uniform, so the first write and the tenth go
through the identical path.

**Work breakdown and exit criteria** live in the backlog, not here ([T56](../backlog/decisions/decision-T56.md)): the `M2 Curation` milestone and its tasks (`backlog milestone list`, `backlog task list --plain -m "M2 Curation"`). This section keeps the scope narrative and the boundaries.

**Explicitly out of scope**

Anything touching a host stanza or a host-group file. `adopt host` and `release host` are M3, and
so is every `WriteRegion` change — M2 writes key files, symlinks, and `.hasp` markers only.

---

## 5. M3 — Configuration

> *hasp edits SSH config without ever losing a hand edit.* Covers **J7**, and makes **J8**
> (Offboard) answerable.

**Work breakdown and exit criteria** live in the backlog, not here ([T56](../backlog/decisions/decision-T56.md)): the `M3 Configuration` milestone and its tasks (`backlog milestone list`, `backlog task list --plain -m "M3 Configuration"`). This section keeps the scope narrative and the boundaries.

---

## 5.5. M3.5 — Hardening

The `.5` is deliberate, not a placeholder. §2 through §5 above are cited directly by roughly
twenty Go source comments — `roadmap.md §2` for M0, `§3` for M1, `§4` for M2, `§5` for M3 — and
renumbering any of them to make room for a new milestone would break every one of those citations
invisibly, with no compiler error to catch it. This section sits at `§5.5` for exactly that
reason, and §6 through §8 keep the numbers they already had.

> *hasp is something someone else could install, trust, and walk away from.* Covers no new
> journey — it makes **J1** (Survey) survivable by a stranger, and discharges **P6**.

**Work breakdown and exit criteria** live in the backlog, not here ([T56](../backlog/decisions/decision-T56.md)): the `M3.5 Hardening` milestone and its tasks (`backlog milestone list`, `backlog task list --plain -m "M3.5 Hardening"`). This section keeps the scope narrative and the boundaries.

**Explicitly out of scope**

Anything touching J9 or M4. `aur` and `homebrew_casks` — [T33](../backlog/decisions/decision-T33.md),
[T40](../backlog/decisions/decision-T40.md) — deferred pending repositories the author has not yet chosen to
create and maintain. Multi-directory / non-default key locations — `design.md` §10 leaves them
deferred and this milestone does not reopen them. **Investigation**
([§5.6](#56-m36--investigation)) — this milestone hardens what M1–M3 already built; it does not
extend the read surface.

> **Settled.** `origin` is `git@github.com:boweeb/hasp.git` ([T40](../backlog/decisions/decision-T40.md)) —
> the Go module path, `go install`, and all four distribution channels
> [T33](../backlog/decisions/decision-T33.md) named unblocked at once, exactly as predicted. `signs` and `ko`
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

**Work breakdown and exit criteria** live in the backlog, not here ([T56](../backlog/decisions/decision-T56.md)): the `M3.6 Investigation` milestone and its tasks (`backlog milestone list`, `backlog task list --plain -m "M3.6 Investigation"`). This section keeps the scope narrative and the boundaries. Exit criterion 8 — the `v1.0.0` tag — is deliberately the maintainer's to cut by hand and is its own open task there.

**Explicitly out of scope**

The keyring-backed passphrase cache — considered and rejected for now (`design.md` §10); the
two rungs this milestone ships (agent, explicit prompt) are expected to suffice without one.
Anything touching J9 or M4.

---

## 6. M4 — Identity

> *A profile answers "who am I being" across SSH, Git, and signing.* Covers **J9**.

**Not before M3 is finished, and this is a hard gate.** [`D8`](../backlog/decisions/decision-D8.md) is explicit
that no near-term milestone may include it, and
[`docs/project-assessment-2026-08.md`](project-assessment-2026-08.md) documents this project
having already died once from opening a second front before the first was done.

**[M3.5 — Hardening](#55-m35--hardening) and [M3.6 — Investigation](#56-m36--investigation) now
sit between M3 and this milestone.** The gate above is unweakened by their insertion — neither
touches J9 work, and both exist precisely to keep this milestone from starting early, not to
soften the wait for it. When M4 does land, it ships as **v1.1.0**, additive over the v1.0.0
surface M3.6 freezes at its close ([T31](../backlog/decisions/decision-T31.md),
[T37](../backlog/decisions/decision-T37.md)).

No work breakdown is written here on purpose. `tdd.md` §14 states only the constraints M4 must not
foreclose — that `Profile` stays additive, that P9 governs Git/GPG taxonomy exactly as it governs
SSH, and that `D16`'s admission rule survives J9's arrival unweakened. Designing further than that
now would be the failure mode this gate exists to prevent. The backlog holds M4 only as drafts
(`backlog draft list`) — ideas that commit to nothing, which become tasks only once the gate above
has opened.

---

## 7. Risks

| Risk | Where it bites | What holds it |
| --- | --- | --- |
| **The round-trip contract fails on a real config** — P2 is the promise the whole project is judged on, and the co-owned `~/.ssh/config` is where it is hardest | M3 | The CST is built and fuzzed in **M1**, a full milestone before anything writes through it. If it cannot hold, that is discovered while it is still cheap |
| **`adopt` loses a key file** — the most invasive operation, and the only one that moves key material | M2 | D4's copy→verify→replace ordering ([`T20`](../backlog/decisions/decision-T20.md)), plus an injected-failure test at every step |
| **The derivation gap surprises someone** — an undecidable key cannot be deduplicated, ever | M1 | It is reported as its own finding rather than hidden; `unknown` is a first-class answer ([`T12`](../backlog/decisions/decision-T12.md)) |
| **Scope creep toward J9** — the documented, already-experienced failure mode | Any | `D8`, and M4's hard gate above |
| **`settings.toml` grows into the old state file** — it has exactly that shape | M2 onward | `D16`'s admission rule, enforced by the field-set test rather than by review discipline |
| **The docs and the code drift apart** — the thing that killed the predecessor's design | Any | Every `Dn`/`Tn` ID is stable and citable from code comments and commit messages; a change contradicting one is supposed to be caught by that citation failing to make sense |
| **The compatibility surface freezes at v1.0.0 while the module path was still unsettled** — a problem that would have become unfixable-after-the-fact the moment the tag was cut | The v1.0.0 tag ([M3.6](#56-m36--investigation)) | Resolved. [T31](../backlog/decisions/decision-T31.md) named it a prerequisite instead of leaving it implicit; [T40](../backlog/decisions/decision-T40.md) settled it — `origin` moved to `git@github.com:boweeb/hasp.git`, a milestone ahead of the tag it gated, not at the last moment |
| **CI logic drifts back into a platform's YAML** — the failure mode the Mage shim exists to prevent | Whenever a new platform is added | [T32](../backlog/decisions/decision-T32.md)'s thin-shim rule and the single `CI` aggregate target |
| **Generated documentation goes stale** — a CLI reference, man pages, or completions that silently disagree with the binary | Continuously, every time a flag or command changes | [T34](../backlog/decisions/decision-T34.md)'s CI staleness check |
| **Investigation scope grows past what J10 actually needs** — an open-ended registry and an opt-in mode are both natural places for "just one more candidate field" to accrete | [M3.6](#56-m36--investigation) | Held by that milestone's own exit criteria (a fixed set of fixture-backed assertions, not a growing wishlist) and by the keyring rung staying explicitly deferred (`design.md` §10) |
| **The confidence vocabulary leaks a fifth value, or one of the four gets silently redefined** — the exact failure P10's closed-set promise exists to prevent | [M3.6](#56-m36--investigation) onward | The golden-list guard ([T37](../backlog/decisions/decision-T37.md), [`tdd.md` §12](tdd.md#12-testing-strategy--a-function-of-a-directory)) — the same mechanism already used for `check`'s finding `id`s and the settings field set |

---

## 8. What this document is not

- **Not a schedule.** No dates, no estimates. See §1.
- **Not technical design.** Every mechanism above is a citation, not a specification. If this
  document and [`docs/tdd.md`](tdd.md) ever disagree about *how* something works, the TDD is
  right and this file has a bug.
- **Not a backlog.** Task-level tracking, if it ever exists, lives outside the documentation set.
  It now exists, and it does live outside this set: `backlog/` ([T56](../backlog/decisions/decision-T56.md)),
  managed with the `backlog` CLI. Milestones, work breakdowns, and exit criteria are there; the
  sequencing and scope reasoning stays here.
