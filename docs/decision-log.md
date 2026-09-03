---
Status: APPROVED
DateCreated: 2026-08-26
DateLastReviewed: 2026-09-03
Related:
  - "[`docs/design.md`](design.md)"
  - "[`docs/tdd.md`](tdd.md)"
  - "[`docs/tech-decision-log.md`](tech-decision-log.md)"
  - "[`docs/roadmap.md`](roadmap.md)"
---

# hasp — Decision Log

**Purpose.** The permanent record of design questions that have been *settled*, and why.
`docs/design.md` describes what hasp is. This file records how it came to say that, which
alternatives were rejected, and what each decision commits the project to.

**How to use this file.**

- Entries are append-only. A decision is never edited to say something different — it is
  superseded or amended by a **later** entry that names it. The wrong turn stays visible.
- IDs are permanent and stable. `D3` means `D3` forever, including in code comments and
  commit messages.
- `docs/design.md` is the *current* statement of the design and always reflects every
  accepted decision here. When the two disagree, this log is the history and the design doc
  is the truth.
- An entry needs a **Consequence** to be complete. A decision whose cost nobody wrote down
  is a decision nobody actually made.

**Status values:** `Accepted` · `Open` · `Amended by Dn` · `Superseded by Dn` · `Rejected`

---

## Index

| ID | Decision | Status |
| --- | --- | --- |
| [D1](#d1) | "Profile" is the single organizing concept for personas | Accepted — amended by [D9](#d9) |
| [D2](#d2) | A key may belong to more than one profile | Accepted |
| [D3](#d3) | The nouns are key, host, profile; "config" is not a noun | Accepted — refined by [D10](#d10) |
| [D4](#d4) | hasp forgets records; it never destroys key material | Accepted — mechanism amended by [D12](#d12) |
| [D5](#d5) | An alias is a name hasp materializes as a link | Accepted |
| [D6](#d6) | The write model: preview is the default for every write | Accepted — collapsed by [D12](#d12) |
| [D7](#d7) | hasp owns only the regions of a file it marks as its own | Accepted — amends P2 |
| [D8](#d8) | The Git/GPG identity direction stays, as direction | Accepted |
| [D9](#d9) | Host groups exist and are SSH config files | Accepted |
| [D10](#d10) | Nouns have two classes: first-class and second-class | Accepted — grid resized by [D12](#d12), [D14](#d14) |
| [D11](#d11) | Offboarding and unified identity are ratified journeys | Accepted |
| [D12](#d12) | hasp has no persisted state; it is a pure function of the machine | Accepted — completed by [D13](#d13) |
| [D13](#d13) | Profile membership lives in the filesystem; taxonomy prefers natural structures | Accepted — adds P9; amended by [D15](#d15), [D16](#d16) |
| [D14](#d14) | Managed vs. unmanaged resources; `adopt` and `release` | Accepted — amends [D10](#d10), J1; amended by [D15](#d15), [D17](#d17), [D19](#d19) |
| [D15](#d15) | hasp reads the marker's presence, never its contents | Accepted — amends [D13](#d13), [D14](#d14); extended by [D18](#d18) |
| [D16](#d16) | A settings file exists; P9's last rung is reached | Accepted — amends [D13](#d13)'s P9 |
| [D17](#d17) | `new key` may ask for a passphrase at generation time | Accepted — narrows §3.2, P3; generalized by [D19](#d19) |
| [D18](#d18) | `release` never deletes content, for hosts as for profiles | Accepted — extends [D15](#d15) |
| [D19](#d19) | P3 is split: custody stays absolute, access becomes consent-gated | Accepted — narrows P3; generalizes [D17](#d17) |
| [D20](#d20) | J2 spans fingerprint schemes, not merely spellings | Accepted — amends J2 (§7) |
| [D21](#d21) | Investigation is a first-class purpose; J10 is ratified | Accepted — adds J10; amends §1, §3.1 |
| [D22](#d22) | Confidence is part of the fact; P10 is added | Accepted — adds P10; generalizes §5.1's `unknown` |

---

<a id="d1"></a>
## D1 — "Profile" is the single organizing concept for personas

**Date:** 2026-08-26 · **Status:** Accepted — consequence amended by [D9](#d9)

### Context

The original README promised "profiles and profile groups," with groups for `personal` and
`work` and multiple profiles per group. The original implementation instead built "key groups"
with one level of nesting, which it then flattened into dotted names like `work.foobarco`. Two
vocabularies, one underlying idea, and no statement of which was correct.

### Decision

**One concept, hierarchically named.** A profile is a persona. Profile names are paths:
`work.foobarco` is a profile, and `work` is its parent and also a profile. A "profile group" is
simply a profile that has children. Nesting is a naming convention, not a second type.

### Rationale

Collapsing the two satisfies the original requirement — multiple profiles per context
(`work.foobarco`, `work.acme`) — without a second entity. It also matches what the original code
*actually did* once it flattened its configuration, meaning the implementation had already
discovered this answer without anyone writing it down.

### Consequence

The words "key group" and "profile group" leave the project's vocabulary permanently.

The original form of this consequence also retired "host group." **[D9](#d9) reverses that
specific part**: host groups exist, but as a container concept on a different axis, not as
persona taxonomy. Profiles remain the sole *organizing* concept.

---

<a id="d2"></a>
## D2 — A key may belong to more than one profile

**Date:** 2026-08-26 · **Status:** Accepted

### Context

Whether key↔profile membership is one-to-many or many-to-many. One-to-many is tidier and
produces simpler output.

### Decision

**Many-to-many.** A key belongs to zero or more profiles.

### Rationale

A key genuinely shared between `personal` and `work.foobarco` is a real situation, not a modelling
error. Forcing a single choice would make the tool assert something false about the machine,
which violates the premise that hasp tells the truth about what is there.

### Consequence

"Which profile is this key in" is a list, not a value. [J8](#d11) (offboard) must handle the
case of a key scoped to a departing profile *and* to something being kept — which is precisely
the case where getting it wrong is most expensive.

---

<a id="d3"></a>
## D3 — The nouns are key, host, and profile; "config" is not a noun

**Date:** 2026-08-26 · **Status:** Accepted — refined by [D10](#d10)

### Context

"Config" meant two unrelated things: hasp's own settings, and the SSH configuration file. The
original implementation had both, under one word, and its verb grid carried a `config` noun
whose meaning was never pinned down — every one of its subcommands was an unimplemented stub,
which is itself evidence that nobody knew what it was for.

### Decision

**Drop `config` as a noun.** SSH configuration is the rendered consequence of hosts and
profiles; it is managed by managing hosts. hasp's own settings are *settings*, are not part of
the verb grid, and are edited directly.

### Rationale

An entity nobody can define is an entity that will be implemented inconsistently. Splitting the
overloaded word removes the ambiguity, and the remaining nouns each have an obvious referent.

### Consequence

Every scenario the old `config` noun was going to cover belongs to `host` or to `settings`.
[D10](#d10) refines this by establishing that the three nouns are the *first-class* set, and
that further nouns exist beneath them.

---

<a id="d4"></a>
## D4 — hasp forgets records; it never destroys key material

**Date:** 2026-08-26 · **Status:** Accepted — mechanism amended by [D12](#d12)

### Context

The abandoned implementation carried a full create/read/update/**delete** shape in its
scaffolding, but the command surface had no delete verb. The question of whether hasp destroys
things was never resolved — it was simply left in two states at once.

### Decision

**`forget` removes a record and never touches disk.** Adopted in the strong form:
**hasp has no operation that can destroy an irreplaceable secret.** This is canon.

### Rationale

`rm` already exists and is better understood. Reconciliation will notice the absence and update
the records on its own. A tool that has never destroyed anyone's private key has a perfect
safety record it can simply keep, and that record is worth more than the convenience of a
delete verb.

### Consequence

Dropping a record is cheap and fully reversible — reconciliation restores it if the material is
still there. Removing key material is the user's own act, performed with their own tools. This
is a permanent, load-bearing safety property, and it constrains every future operation: any
proposal that would give hasp the power to destroy key material contradicts ratified canon.

### Amended by D12

[D12](#d12) removed persisted state, which retires the `forget` **verb** — there are no records
to drop from. The **canon is unchanged and if anything is strengthened**: with nothing
persisted, forgetting is simply what happens when the material is gone and hasp next looks. The
safety property no longer depends on hasp declining to do something; it holds because hasp has
no mechanism through which it could.

---

<a id="d5"></a>
## D5 — An alias is a name hasp materializes as a link

**Date:** 2026-08-26 · **Status:** Accepted

### Context

Aliases exist because something outside your control insists on a name you didn't choose — a
tool that hardcodes `id_rsa`, a script you can't edit. The original implementation discovered
them from symlinks and stored them with globally unique names, but never stated what an alias
*is*.

### Decision

An alias is an **additional name** for a key. hasp discovers aliases when adopting a machine
and materializes them in the world as links. Names and aliases share one flat namespace.

### Rationale

Uniqueness is inherited from the filesystem rather than invented by hasp: within a directory,
two things cannot share a name, and hasp should not pretend to a richer model than the thing
every other tool actually reads.

### Consequence

hasp's alias model can never be more expressive than a directory. This is correct, not
limiting — the directory is the interface every other tool consumes.

---

<a id="d6"></a>
## D6 — The write model

**Date:** 2026-08-26 · **Status:** Accepted (2026-08-26) · **Collapsed by:** [D12](#d12)

### Context

Every write being previewable and reversible is desirable, but applying that ceremony
uniformly would make cheap, safe operations tedious.

### Proposal (not yet ratified)

**Two tiers.** Changes to hasp's own records are immediate — they are reversible by
re-reconciling, since the machine is the source of truth. Changes to key files or configuration
go through preview → confirm → back up → write.

### Open questions

- Is the tier a property of the *operation* or of what it touches at runtime?
- Does preview need to be the default for the risky tier, or an available mode?
- Where do operations that touch both tiers at once land?

### Consequence if accepted

"What would this do" must be answerable *before* a change is applied, which shapes how change
is represented from the very beginning. This cannot be added later without rework.

### Reshaped by D12

[D12](#d12) removes persisted state entirely, which **collapses the two tiers into one**. If
hasp keeps no records, there is no such thing as a "record-only change" — every write touches
the world, and every write therefore gets the full preview → confirm → back up → write cycle.

The first two open questions above are answered by that collapse; the third dissolves, since no
operation can span tiers that no longer exist. What survives is narrower and should be settled
on its own terms: **is preview the default for every write, or an available mode?**

### Decision

**Every write.** Preview is the default, not a mode the user has to remember to ask for.

*Rationale:* the alternative puts the burden of caution on the person with the least context at
the moment they act. A default that is safe only when you remember to opt into it is not a safe
default. At this scale (P8) the extra step costs a keystroke and buys the guarantee P5 was
written to make.

*Consequence:* there is exactly one write model, applied uniformly — **preview → confirm →
back up → write** — with no operation-by-operation judgement about which changes are risky
enough to warrant it. Non-interactive use must therefore have an explicit way to consent in
advance; its spelling is an interface concern and remains deferred.

---

<a id="d7"></a>
## D7 — hasp owns only the regions of a file it marks as its own

**Date:** 2026-08-26 · **Status:** Accepted (2026-08-26)
**Amends:** P2 · **Reshapes:** [D13](#d13)

### Context

The single hardest and most consequential question in the design, and the one that determines
whether "hand edits are sacred" is a real guarantee or an aspiration. Configuration files are
co-owned by hasp and the human.

### Proposal (not yet ratified)

**hasp owns marked regions; everything else is human territory, preserved exactly.** hasp does
not take custody of a file it did not fully author. Content it does not understand is not an
error and is never rewritten — it is simply not hasp's.

### Bearing of D9 on this question

[D9](#d9) materially changes the shape of this problem and should be resolved *before* D7 is
ratified. If host groups are separate files, then hasp files it created wholesale
(`~/.ssh/<group>.sshconfig`) may be fully hasp-owned, and marked-region co-ownership may be
needed only for the default `~/.ssh/config` — the one file the human has already written in.
That would narrow the hard problem considerably. Whether it narrows it *enough* to change the
answer is exactly what needs deciding.

### Open questions

- Does the default group file get marked-region treatment, or is it read-mostly with hasp
  writing only an include?
- What happens when a human edits inside a hasp-marked region?
- Are hasp-authored group files still preserved byte-for-byte outside their known constructs?

### Consequence if accepted

hasp must be able to work with a file it only partially comprehends, forever. This forecloses
any approach requiring a complete parse of a config file in order to write to it at all, and
is the primary constraint on the deferred parsing investigation.

### Decision

**Marked regions, accepted** — with elaborations that make the mechanism stronger than the
original proposal, and that answer all three open questions above.

**1. Inside its own markers, hasp is authoritative.** hasp-owned regions are *rigidly managed*:
hasp may normalize, reorder, and reformat them freely, because it wrote every byte.

**2. Manual edits inside a hasp region are intentionally lost.** This is the trade that buys
rigid management, and it is deliberate rather than regrettable. **It amends P2** — hand edits
are sacred *outside* hasp's markers; inside them they are forfeit. The marker is both the
boundary and the warning, and P2's guarantee is narrowed to match rather than quietly violated.

**3. hasp-owned regions may carry structured metadata as comments.** Anything hasp needs to
record that the format cannot express natively is written into its own region in comment form.
The metadata structure must be expressive enough to capture whatever the human intended, so
that a construct hasp does not model natively can still be carried rather than dropped.

**4. The default file is supported; explicit host groups are the nudge.** `~/.ssh/config` gets
marked-region treatment like any other file. But the smoothest experience is explicit host
group files ([D9](#d9)), and hasp should steer users there — the fully-owned file is the easy
case, and the co-owned one is where the sharp edges live.

### Why elaboration 3 matters

It **substantially dissolves the exhaustive-parsing problem**, which was the project's hardest
identified technical risk. hasp no longer needs a complete native model of the configuration
format in order to be safe with it: what it understands it manages natively, and what it does
not it carries as metadata. The parsing investigation shrinks from "model the whole format" to
"model what we manage, carry the rest."

### Is comment-metadata a violation of D12?

**No, and the distinction is worth stating explicitly**, because this is precisely the kind of
mechanism that could reintroduce persisted state through a side door.

[D12](#d12) prohibits *derived* state — a copy of facts the machine can already answer, which
drifts because the original changes without telling you. Comment-carried metadata is
**declared**: it is input, authored deliberately, mirroring nothing. It lives in the world
rather than beside it, is legible without hasp (P6), and is version-controllable with the file
that carries it.

The line to hold: **metadata may record what the machine cannot tell you. It may never cache
what the machine can.** A fingerprint written into a comment would be a D12 violation. A profile
assignment would not.

### Consequence

- **P2 is narrowed** to "sacred outside hasp's markers." The design document states the
  amendment where P2 is defined.
- hasp gains a general mechanism for recording declared intent inside files it owns, which
  **reshapes [D13](#d13)** — profile membership now has a candidate home that costs no new file.
- Rigid management means a hasp region is fully regenerable, so a corrupted or hand-mangled
  region is repairable by rewriting it rather than by parsing what someone did to it.
- The metadata format becomes load-bearing and needs its own design: it must round-trip unknown
  constructs, survive reformatting, and stay readable to a human with an editor.

---

<a id="d8"></a>
## D8 — The Git/GPG identity direction stays, as direction

**Date:** 2026-08-26 · **Status:** Accepted

### Context

The original README listed GPG key awareness and Git identity management under "Future Ideas."
The question was whether to keep them in the vision at all, given that the project's history is
a case study in what happens when a second front opens before the first is finished.

### Decision

**Keep it in the vision; keep it out of the near-term work.**

### Rationale

It stays because it is *why* profiles are shaped as they are — a persona that carries a signing
key and a commit identity is the reason "profile" is the organizing concept rather than "key
group." Dropping it would reduce hasp to a key-inventory tool. It waits because SSH alone is
not yet done.

### Consequence

The domain model must not make [J9](#d11) impossible. No near-term milestone may include it.
Reaching for it early is the specific failure mode this project has already experienced once.

---

<a id="d9"></a>
## D9 — Host groups exist, and a host group is an SSH config file

**Date:** 2026-08-26 · **Status:** Accepted · **Amends:** [D1](#d1)

### Context

[D1](#d1)'s original consequence retired "host group" along with "key group" and "profile
group," on the reasoning that profiles are the one grouping concept. On review this was wrong —
not because profiles are the wrong organizing concept, but because "host group" was being read
as *taxonomy* when its natural meaning here is *container*.

### Decision

A **host group is a physical container for host entries: an SSH config file.**

- The default host group is `~/.ssh/config`.
- A custom host group named `foo` is `~/.ssh/foo.sshconfig` — the file is named literally after
  the group.
- The default group is the sole exception to that naming rule.

### Rationale

SSH configuration is already multi-file in practice; the format supports composition natively.
Modelling that reality directly gives hasp a materialization strategy that is legible from
outside the tool: you can see the grouping by listing a directory, and any other tool — or a
human with an editor — reads exactly the same structure hasp does. It requires no invented
metadata, and it keeps hasp's records a projection of the world rather than a claim about it.

It also isolates blast radius. A group hasp authored is a file hasp authored, which is a much
easier thing to reason about than a region inside a file someone else wrote.

### Relationship to profiles

Host groups and profiles are **different axes and must not be conflated**:

- A **profile** is *who you are being* — persona, organizing taxonomy, spans keys and hosts and
  eventually Git/GPG identity.
- A **host group** is *where a host entry physically lives* — a file on disk.

They will often correlate, and a sensible default may be to materialize a profile's hosts into
a group file of the same name. They remain distinct concepts, and nothing requires them to
align.

### Consequence

- [D1](#d1)'s consequence is amended: "host group" returns to the vocabulary with this specific
  meaning. "Key group" and "profile group" stay retired.
- hasp must model *where a host lives* in addition to *what it is* and *who it belongs to*.
- Materializing a group means creating and owning a file; hasp's write model must account for
  files it fully authored as a distinct case from files it co-owns. This bears directly on
  [D7](#d7), which should be settled with this in hand.
- The default group is a file hasp did not create and probably should not claim, which is the
  hardest case and the one D7 must answer.

---

<a id="d10"></a>
## D10 — Nouns have two classes: first-class and second-class

**Date:** 2026-08-26 · **Status:** Accepted · **Refines:** [D3](#d3)

### Context

[D3](#d3) settled that the nouns are key, host, and profile. But the domain plainly contains
more nouns than three — alias, binding, host group — and it was unclear whether each needed a
place in the verb grid, which would multiply the command surface for concepts that are not
independently meaningful.

### Decision

Nouns come in two classes.

- **First-class nouns** — **key**, **host**, **profile**. These are the entities that exist in
  their own right. The verb + noun grammar applies to these and only these.
- **Second-class nouns** — alias, binding, host group, and others yet to be identified. These
  belong *to* a first-class noun and are addressed as modifiers of an operation on their owner,
  not as operations in their own right.

Illustratively: `hasp edit key "foo-key" --add-alias="bar-key"`, rather than a top-level
`alias` noun with its own verbs.

### Rationale

A second-class noun has no independent existence — an alias without a key is meaningless, a
binding without a host and a key is nothing. Promoting such concepts to the verb grid would
grow the command surface combinatorially while offering operations that are either nonsensical
or duplicative.

Keeping the first-class set at three preserves the property that made the original verb×noun
grid the best idea in the abandoned implementation: learning one noun teaches you the others,
and adding a noun is mechanical rather than inventive.

### Consequence

- The verb grid does not grow as the domain gets richer. (It was 3 × 8 when written; [D12](#d12)
  retired two verbs and [D14](#d14) added two, so it is 3 × 8 again. The *count* moves with the
  verb set; the point of this decision is that it does not move with the number of concepts.)
- Every new concept must be classified on arrival. "Is this first-class?" becomes a standing
  design question, and the bar is independent existence.
- Second-class nouns surface as named arguments. Their exact spelling is an interface concern
  and remains deferred; the *classification* is design and is settled here.

---

<a id="d11"></a>
## D11 — Offboarding and unified identity are ratified journeys

**Date:** 2026-08-26 · **Status:** Accepted

### Context

Two journeys were proposed as extensions beyond the original vision rather than restatements
of it, and were flagged as such for explicit acceptance or rejection.

### Decision

Both are accepted as part of the vision.

- **J8 — Offboard.** *I'm leaving a context. Show me everything scoped to that profile — every
  key, every host that depends on it — so I know what to revoke and what will break.*
- **J9 — Unify identity.** *A profile carries not just SSH keys but the Git identity and
  signing key that go with the persona.* Post-v1, per [D8](#d8).

### Rationale

J8 is the practical payoff of profiles being first-class, and it is the strongest justification
for modelling bindings as a relation rather than a directive: the valuable question runs
*backwards* along the binding — which hosts use this key — and that is exactly what you need on
your last day somewhere.

J9 is the original "Future Ideas" section promoted from a list of features to the reason the
domain model has the shape it does.

### Consequence

- J8 makes the binding relation load-bearing rather than incidental, and it must work correctly
  for keys shared across profiles ([D2](#d2)).
- J9 is bound by [D8](#d8): it shapes the model, but no near-term milestone may include it.

---

<a id="d12"></a>
## D12 — hasp has no persisted state; it is a pure function of the machine

**Date:** 2026-08-26 · **Status:** Accepted · **Reshapes:** [D6](#d6) · **Opens:** [D13](#d13)

### Context

The original implementation kept a TOML state file. It was introduced for a modest reason — a
debug and development aid, for visibility into what the tool was seeing — and then quietly
became the state model. In September 2020 it began migrating to SQLite with an ORM, and that
migration is exactly where the project died: `sync` half-written, `# TODO: Left off here` in
the middle of the reconciliation logic, and two competing stores wired into different commands
for the next six years.

The question this entry settles is whether hasp needs persisted state at all.

### Decision

**It does not.** hasp derives everything it reports from the machine at the moment it is asked.
There is no index, no database, no state file. hasp is a pure function of the filesystem.

**Corollary — no cache either.** Not merely "no cache for now": a cache requires a *measured*
justification — an actual timing, on an actual `~/.ssh`, showing a delay a human notices. If
one is ever admitted it must be disposable, auto-invalidating, never consulted for correctness,
and deletable mid-run with no effect other than latency.

### Rationale

**`sync` dissolves.** The verb exists only to reconcile an index against reality. With no index
there is nothing to reconcile. The single most-worked and least-finished part of the abandoned
codebase — the part with the abandonment marker literally inside it — is not fixed by this
decision, it is *deleted* by it. That is the strongest available evidence the direction is right.

**The sin was persisting derivable facts, not persistence as such.** Derived state drifts,
because it is a copy of something that changes without telling you. Declared state does not,
because it is not mirroring anything. The old state file stored fingerprints, algorithms, bit
lengths, formats, and aliases — every one of them a question the machine answers in
milliseconds. That is what rotted.

**An audit found almost nothing that must be declared.** Fingerprint, algorithm, size, comment,
format: from the key artifact. Aliases: from symlinks. Hosts, host settings, bindings: from
config files. Host groups: from files in the key directory ([D9](#d9)). The only residue is
profile membership, which is [D13](#d13).

**A cache does not pay at this scale.** Tens of keys, a few milliseconds each (P8).
For a process that starts and exits, a cache only helps if it is on disk — which means
invalidation, staleness checks, and a second thing that can be wrong. It would reintroduce the
exact failure being eliminated in order to save time a human cannot perceive.

**And a cache is how this happened the first time.** The TOML file started as "just a
visibility aid." A cache admitted "just in case" is the same nose under the same tent, which is
why the bar above is deliberately set at measurement rather than judgement.

**Statelessness is more honest.** Where hasp cannot derive a fact it must say so. A stateful
hasp would confidently report a fingerprint it can no longer verify.

**Three properties come free.** There is no corruption mode, because there is no index to leave
inconsistent. Testing becomes tractable — hasp is a function of a directory, so a fixture tree
is the entire setup, which directly addresses why the original has zero tests. And abandonment
costs nothing, since nothing is trapped in a format nobody remembers.

### The derivation gap

One case was tested rather than assumed, and it is real:

| Case | Fingerprint derivable? |
| --- | --- |
| Public half present | Yes |
| OpenSSH format, encrypted, no public half | **Yes** — the public half is stored unencrypted inside the private file; only the comment is lost |
| Legacy PEM, encrypted, no public half | **No** — `ssh-keygen` reports `is not a key file` |

The last row matters here specifically: seven of the thirteen keys in the author's own recovered
inventory are PEM format.

**Resolution: report the fact as unknown.** hasp does not prompt for a passphrase (P3), and it
does not cache a value it cannot re-verify. "Unknown" is the correct answer and a stateful
design would answer it worse.

### Consequence

- **`sync` and `forget` are retired from the verb set**, which drops from eight verbs to six.
  `forget` meant "drop this from my records"; there are no records. [D10](#d10)'s grid is
  now 3 × 6.
- **P1 strengthens** from "the index is rebuildable" to "there is no index." **P6** — records
  legible without hasp — is satisfied maximally and vacuously, and is restated rather than kept
  as written.
- **[D6](#d6)'s two tiers collapse into one.** See that entry.
- **Storage leaves the deferred list.** There is nothing to defer.
- **Every read is a scan.** Correctness now depends on derivation being complete and cheap,
  which makes the derivation gap above a permanent design fact rather than an edge case.
- **[D13](#d13) is opened** by the one thing that could not be derived.

---

<a id="d13"></a>
## D13 — How profile membership is expressed in the world

**Date:** 2026-08-26 · **Status:** Accepted (2026-08-27) · **Opened by:** [D12](#d12)
**Adds:** P9 · **Depends on:** [D14](#d14) · **P9 amended by:** [D16](#d16)

### Context

[D12](#d12) removed persisted state on the finding that nearly everything hasp reports is
derivable from the machine. Profile membership is the exception: nothing on disk says that
`id_rsa_foobarco` is a *work* key. It must therefore either be carried by the world in some
structural form, or be the one thing hasp declares.

### Options

**(i) The filesystem carries the taxonomy.** A profile is a directory:
`~/.ssh/work.foobarco/id_rsa`. Tools that hardcode `~/.ssh/id_rsa` are satisfied by a mechanism
already ratified — aliases ([D5](#d5)) are symlinks, so the top-level name remains and points
into the profile directory.

This completes a pleasing symmetry with [D9](#d9): **directory = profile, file = host group,
symlink = alias, key file = key, config stanza = host, and nothing else exists.** The
filesystem *is* the database. Everything is visible to `ls`, greppable, and version-controllable
without hasp.

*Cost:* hasp becomes opinionated about `~/.ssh` layout, and adopting it means **moving key
files** — the most invasive act hasp would ever perform. It sits awkwardly against J1, whose
promise is that adoption changes nothing.

**(ii) Infer membership from bindings.** A key referenced by `work.sshconfig` is a work key.
Zero file moves. *Cost:* profiles cannot exist independently of hosts, and a key bound to
nothing has no profile — which is precisely the key you most need to reason about when
offboarding (J8).

**(iii) A hand-authored manifest.** Declared, not derived, so it does not drift — it is input
rather than cache, and does not reintroduce what D12 removed. *Cost:* it is a file to keep, and
it is the shape of the thing this project has twice failed to maintain.

**(iv) Metadata in a hasp-owned region.** *Added by [D7](#d7), after this entry was written.*
Ratifying marked regions gave hasp a general mechanism for recording declared intent inside
files it already owns. Profile membership could live there — declared rather than derived, so
D12-compatible; carried in a file that already exists, so no new artifact to maintain; and
legible with an editor (P6). *Cost:* it ties key taxonomy to config files, which are about
hosts, so a key belonging to no host still needs a home. This is (iii) without the separate
file, and it is the option D7 made possible.

### Open questions

- Can adoption stay read-only and layout-agnostic under option (i), with reorganizing as a
  separate opt-in act?
- Does option (i) break anything that reads `~/.ssh` directly and does not follow symlinks?
- Under (i) or (ii), where does a profile's non-SSH identity live when [J9](#d11) arrives —
  Git identity has no natural home in a key directory.

### Decision

**Option (i): the filesystem carries the taxonomy.** A profile is a directory under the key
directory, and a key's profile membership is the directory it lives in.

**Directories nest; `.hasp` decides.** Profile name segments map to path segments, so
`work.foobarco` is `~/.ssh/work/foobarco/`. A directory is a profile **only if it carries a `.hasp`
marker**, which means `work` can be a profile in its own right, or merely a container that
groups others, and the marker settles which. The marker file was originally required to stay
**empty**, to keep it from becoming a manifest — option (iii) through the back door.
**[D15](#d15) restates that rule** at the right altitude: what matters is that hasp never
*reads* the file, not that the file has no bytes.

### The general rule this establishes — promoted to P9

The reasoning generalizes well beyond profile membership, so it is promoted to a principle
rather than left as a local answer:

> **P9 — Taxonomy rides on existing structure.** Whenever possible, organizing facts are carried
> by natural structures that already exist — filesystem layout, or the configuration data of the
> tool being managed. Where that is impossible, fall back to private metadata alongside the
> thing. Only as a last resort, a hasp settings file. No case has yet required the last resort.

P9 will govern the Git and GPG expansion ([J9](#d11)) as much as it governs SSH: the natural
structures there are `gitconfig` conditional includes and repository layout, not a hasp-owned
registry. Recording it as a principle means that question is pre-answered.

### Rationale

Option (i) completes the symmetry begun by [D9](#d9): **directory = profile, file = host group,
symlink = alias, key file = key, config stanza = host, and nothing else exists.** The filesystem
*is* the database. Everything is visible to `ls`, greppable, and version-controllable without
hasp — P6 satisfied not by choosing a legible format but by having no format at all.

Option (ii) was rejected because a key bound to no host would have no profile, and that is
precisely the key that matters most when offboarding ([J8](#d11)). Options (iii) and (iv)
were rejected as unnecessary once [D14](#d14) made (i)'s cost affordable.

### How (i)'s cost was paid

(i)'s stated cost was that hasp becomes opinionated about `~/.ssh` layout, and that adoption
would mean moving key files — awkward against J1's promise that adoption changes nothing.

**[D14](#d14) pays that cost** by distinguishing managed from unmanaged resources. Unmanaged
resources are reported read-only and are never moved, so the survey journey keeps its promise
exactly. Only a resource the user explicitly adopts is reorganized. Without D14 this decision
would not have been affordable.

### Consequence

- **Profile membership is derived from location.** It is not declared anywhere, which makes the
  model fully derivable — [D12](#d12)'s "no persisted state" is now literally true, with no
  asterisk.
- Moving a key between profiles is `mv`. The world remains the interface.
- **SSH's default identity probing breaks when a key moves.** `ssh` auto-tries
  `~/.ssh/id_ed25519`, `~/.ssh/id_rsa` and friends; a key that moves into a profile directory
  stops being found by anything that relied on that. Adoption must therefore leave a top-level
  alias symlink behind ([D5](#d5)) or write an explicit `IdentityFile`. This is a requirement of
  adoption, not a nicety.
- A key in no profile directory sits at the top level and belongs to no profile, which is a
  legitimate state and exactly what `check` should report.

---

<a id="d14"></a>
## D14 — Managed vs. unmanaged resources; `adopt` and `release`

**Date:** 2026-08-27 · **Status:** Accepted · **Amends:** [D10](#d10), J1 · **Enables:** [D13](#d13)
**Carve-out narrowed by:** [D17](#d17)

### Context

[D13](#d13) put profile membership in the filesystem, which requires moving key files — the
most invasive act hasp would ever perform, and in direct tension with J1's promise that
surveying a machine changes nothing. Something had to make that affordable.

### Decision

**Every resource is either managed or unmanaged, and the difference is consent.**

| | Unmanaged | Managed |
| --- | --- | --- |
| How hasp treats it | Read-only reporting | Full write capability, plus features that depend on complete metadata |
| How it is recognized | Absence of a marker | A marker (below) |
| Whose territory | The user's | hasp's, by explicit opt-in |

**Markers, per noun:**

- **Profile** — a directory carrying a `.hasp` marker file ([D13](#d13), [D15](#d15)).
- **Host** — a stanza inside hasp's markers ([D7](#d7)).
- **Key** — **its location.** A key inside a managed profile directory is managed. There is no
  intrinsic per-key marker; see below.

**Two new verbs**, uniform across all three nouns:

- **`adopt`** — reorganize an unmanaged resource into a managed one.
- **`release`** — the inverse: return a managed resource to unmanaged, undoing the
  reorganization and removing the marker.

### Why keys are marked by location and not by comment

The original proposal was that a managed key carries a hasp marker in its **comment**. That was
tested and rejected on evidence:

| Case | Result |
| --- | --- |
| Comment change alters the fingerprint? | **No** — identity is preserved. The one good result. |
| Encrypted key, non-interactive | **Fails.** `ssh-keygen -c` must decrypt the private key: `Cannot load private key: incorrect passphrase`, exit 255. Marking an encrypted key would require a passphrase, which **P3 forbids** — so the keys a careful user is most likely to hold could never become managed. |
| Legacy PEM key | **Silently converts the private key to OpenSSH format.** Verified: header changes from `BEGIN RSA PRIVATE KEY` to `BEGIN OPENSSH PRIVATE KEY` and the file digest changes. PEM has no comment field, so the comment lands only in the `.pub`. This changes `format` — a fact hasp *reports* — and can break tooling that requires PEM. Seven of the thirteen keys in the author's own recovered inventory are PEM. |

Beneath both failures is a more general objection: marking a key means **rewriting a private key
file**, which is the closest hasp would ever come to [D4](#d4)'s line. Not destruction, but the
one operation where a bad moment costs an irreplaceable secret.

**Location is sufficient anyway.** Under [D13](#d13) location *is* the taxonomy, so an intrinsic
marker is redundant with [P1](#d12) — if a key moves, it moved, and the world is the truth. It
also removes a class of contradictions that an intrinsic marker would have created: a marked key
sitting outside a managed directory, or an unmarked key sitting inside one. With location as the
only signal, those states cannot arise.

**Standing rule: hasp never writes to a private key file.** The sole exception is generating a
new key, where hasp authors the file outright. If an intrinsic marker is ever wanted, the safe
form is a marker in the `.pub` only, on keys hasp generated — never retrofitted, never touching
the secret.

### Why `release` exists

`adopt` moves files and writes markers. Without an inverse it is a **one-way door**, and a
one-way door turns "try hasp on my real `~/.ssh`" from an experiment into a commitment. That
contradicts the spirit of [D4](#d4) and [P6](#d12) — nothing should be trapped inside hasp.
Mechanically the inverse is cheap: move back, remove the marker.

### Consequence

- **[D10](#d10)'s grid grows from 3 × 6 to 3 × 8.** `adopt` and `release` are first-class verbs
  applying uniformly to key, host, and profile. This is the *classification* rule working as
  intended, not an exception to it.
- **J1 is renamed.** It was called "Adopt" and defined as awareness *without changing anything* —
  the exact opposite of the new verb. J1 becomes **Survey**; `adopt` takes the word.
- **J5 (Tidy) implies adopt-first.** Repairing an unmanaged resource requires adopting it, since
  unmanaged is read-only. Reporting untidiness does not.
- **The milestones sharpen.** M1 is now entirely the unmanaged read-only path — hasp ships its
  first useful milestone **without ever writing to `~/.ssh`**. `adopt` and managed writes arrive
  in M2.
- Managed status is an explicit, revocable grant. hasp's authority over a resource is something
  the user hands it and can take back.

---

<a id="d15"></a>
## D15 — hasp reads the marker's presence, never its contents

**Date:** 2026-08-28 · **Status:** Accepted · **Amends:** [D13](#d13), [D14](#d14) · **Extended by:** [D18](#d18)

### Context

[D13](#d13) required the `.hasp` profile marker to stay **empty**, on the reasoning that a file
with content becomes a manifest — option (iii) readmitted through the back door, undermining
[D12](#d12).

That rule was stated at the wrong altitude. It conflated the invariant with a crude proxy for
it. Bytes sitting in a file are not declared state; bytes a *program reads* are. "Empty" banned
the wrong noun, and in doing so it foreclosed something worth keeping: the ability to leave a
note for a human, and room for configuration to grow naturally at that scope if a case ever
genuinely arises that observation cannot cover.

### Decision

**The invariant is restated:**

> **hasp reads the marker's *presence*, never its *contents*.**

Precise, testable, and it protects [D12](#d12) exactly as well as "empty" did — while permitting
what "empty" needlessly forbade.

**1. The file's contents belong to the user.** They may write whatever they like in it. hasp
never parses, validates, or reports on it.

**2. Commentary is prefixed `#`.** One syntax, not two.

**3. `adopt` creates the file with a `#` header** stating what the file is, that its *presence*
is the signal, and that `#` lines are ignored.

**4. `release` must show the file's contents in its preview**, so that removing a marker the
user has written in is never a silent loss.

### Why `#`, and only `#`

**P9 decides it.** The neighbouring file in that very directory — `ssh_config` — uses `#`, as do
shell, `gitconfig`, TOML, and YAML. `//` matches nothing in this ecosystem. The convention that
already exists is the one to ride.

Blessing two syntaxes would be a parsing decision made years early, handing a future reader an
ambiguity for no benefit. And if data ever does land in this file, TOML is its likely shape
given the project's history — `#` is already TOML's comment character, so that door opens
cleanly rather than needing a migration.

### Why elaboration 3 is the load-bearing one

`#`-prefixing only protects a future parser if commentary is *actually* prefixed — and nothing
enforces that, precisely because hasp never reads the file. A user who writes bare prose today
hands a someday-parser something indistinguishable from data.

A `#` header written at creation time converts the convention from folklore into something
visible the first time anyone opens the file. Files written today are then entirely `#` lines,
which a future reader parses as zero data. That is what makes backward compatibility real rather
than hoped for.

### Why not rename the file instead

The alternative considered was renaming the marker to convey intent — `.hasp-owned` or similar.
Rejected: a self-documenting header does the naming's job from inside the file, so the name can
stay short. `.hasp` also ages better. When [J9](#d11) arrives the same marker plausibly scopes a
Git repository, where "owned" is the wrong word for what it marks.

### The gate for "someday"

**The day hasp reads this file's contents, it reopens §5.7 of the design document.**

That section is kept named-and-empty as a standing test: anything hasp must be *told*, rather
than able to look up, requires its own decision with [P9](#d13) as the bar. Content in `.hasp`
that hasp *reads* is precisely that. The flexibility is preserved and the tripwire already
exists; D15 simply points the two at each other.

This matters because the file has the same *shape* as the original failure — the TOML state file
began as "just a debug aid for visibility" and became the state model. The difference now is
that the door is explicitly a door, with a decision required to walk through it, rather than
something that drifts open one convenient field at a time.

### Consequence

- **[D13](#d13)'s "stays empty" is superseded** by the presence/contents rule. The protection is
  unchanged; the prohibition is narrower and aimed at the right thing.
- **§5.7 stays closed today.** Commentary in the marker is human-to-human, not human-to-hasp, so
  nothing hasp must be *told* has been reintroduced.
- **`adopt` gains a requirement:** write the `#` header when creating the marker.
- **`release` gains a requirement:** preview shows the marker's contents, not just its removal.
- Forward compatibility is structural — every file hasp creates is self-describing, and every
  file written under the convention parses as zero data to any future reader.

---

<a id="d16"></a>
## D16 — A settings file exists; P9's last rung is reached

**Date:** 2026-08-29 · **Status:** Accepted · **Amends:** [D13](#d13)'s P9
**Opened by:** [`docs/tdd.md`](tdd.md) §8, [`T7`](tech-decision-log.md#t7)

### Context

[D13](#d13) promoted a ladder to a principle, and gave it a rung nobody expected to climb:
*"Only as a last resort, a hasp settings file. No case has yet required the last resort."*

The technical design found the first case that does. `new key` has to be runnable with no human
present — in a script, in CI — which means the question *"does the generated key carry a
passphrase?"* must be answerable without a prompt. That answer is a **preference**. It is not a
fact about a key, a host, or a profile, so P9's first two rungs have nowhere to put it: no
filesystem layout expresses it, and no `ssh_config` directive carries it. A flag answers it for
one invocation. Nothing answers it for *"always, on this machine."*

This entry exists because the alternative is worse. The predecessor's TOML file arrived as an
implementation detail — [D12](#d12) records it as *"a debug and development aid, for visibility
into what the tool was seeing"* that *"quietly became the state model."* A settings file that
arrives the same way, named only in a downstream technical document, is the same story with the
same ending. If the last rung is going to be climbed, it gets climbed here, on the record.

### Options

**(i) Flags only.** Every invocation states its intent; nothing is remembered. *Cost:* a scripted
caller repeats itself forever, and there is no way to express a standing preference for this
machine. Defensible — it is what hasp does for everything else.

**(ii) An environment variable only.** `HASP_NEW_KEY_PASSPHRASE_MODE`, set in a shell profile.
*Cost:* ambient state that `ls` cannot show you and no editor can find, which is the opposite of
[P6](#d12). It is also per-shell rather than per-machine, so hasp's behavior would depend on how
the process happened to be started — a worse property than the one it is fixing.

**(iii) A settings file.** The last rung. *Cost:* a file with hasp's name on it, which is the
shape of the thing this project has already failed to maintain twice.

### Decision

**Option (iii) — the rung is climbed, once, and gated.**

- **Format and location.** TOML, `#` comments, at `os.UserConfigDir()/hasp/settings.toml`.
  Deliberately **outside `~/.ssh`**: it is not key material and not arrangement, so it must never
  be swept into the profile scanner or into the backup rotation.
- **hasp reads settings and never writes them.** There is no `hasp config set` and no
  settings-mutation verb of any kind. [D3](#d3) already said so — *"hasp's own settings are
  settings, are not part of the verb grid, and are edited directly"* — and this entry walks
  through a door D3 named rather than cutting a new one.
- **Optional throughout.** A missing file is not an error and never triggers a first-run
  initialization step. Every key has a built-in default, so `docs/design.md` §6.3's *"works on
  first run, on a machine it has never seen, with nothing configured"* holds exactly as written.

**The admission rule — the actual content of this decision:**

> Settings may hold **only** user preferences that (a) cannot be derived from the machine, and
> (b) exist to enable non-interactive execution. They may **never** hold facts about the machine.
> They may **never** hold taxonomy. Anything failing this test stays a flag.

Both prohibitions name a specific past failure. *Facts about the machine* is what the abandoned
TOML state file held, and [D12](#d12) exists to keep them out. *Taxonomy* is [D13](#d13)'s
rejected option (iii) — the hand-authored manifest — which would otherwise walk back in through a
door D13 never closed, because D13 was arguing about profile membership and not about this file.

### Rationale

The rung is climbed because the case genuinely clears the bar P9 sets: show that existing
structure cannot carry the fact. It cannot. A passphrase-mode preference is a fact about *the
user*, and P9's first two rungs carry only facts about *the machine*.

The file is safe because of the admission rule, not because of anyone's good intentions. The
original sin, in [D12](#d12)'s own words, was **"persisting derivable facts, not persistence as
such."** A file that structurally cannot hold a derivable fact cannot repeat that sin: anything
derivable fails the rule on sight and is refused a place in it.

And the rule is not left to review discipline. [`docs/tdd.md`](tdd.md) §12 requires a test
asserting the *exact field set* of the type settings decode into, so any change adding a key
changes a visible expected list. The guard that matters most is the one that does not get to rely
on a reviewer noticing.

### Consequence

- **P9's text changes.** *"No case has yet required one"* is false the moment this is accepted,
  and `docs/design.md` §4 states the amended form. The ladder is unchanged; only the claim about
  its last rung is.
- **§5.7 stays closed, and the reason is worth stating.** The intent category guards facts hasp
  would have to be *told* because the machine cannot report them — organizing facts about the
  world. A preference about hasp's own behavior is not one of those: nothing about the machine is
  being declared, so nothing derivable has been displaced. `docs/design.md` states that
  distinction explicitly rather than leaving the suspicion, because this file has the same
  *shape* as the thing §5.7 stands guard against.
- **The v1 keyset is `[new_key]` and nothing else**, so the file does not matter until **M2**.
  M1's *"hasp never writes to `~/.ssh` at all"* is untouched — and hasp never writes this file at
  any milestone.
- Every future settings key must justify itself against the admission rule in its own right.
  That discipline is the whole reason the rule is written here rather than left in the technical
  document that discovered the need for it.
- **A second case reaching this rung needs its own entry.** One case climbing the ladder is not a
  licence for the next one.

---

<a id="d17"></a>
## D17 — `new key` may ask for a passphrase, at generation time only

**Date:** 2026-08-29 · **Status:** Accepted · **Narrows:** §3.2, P3
**Opened by:** [`docs/tdd.md`](tdd.md) §9, [`T6`](tech-decision-log.md#t6)

### Context

`docs/design.md` §3.2 lists nine hard boundaries under the strongest sentence in that document:
*"These are hard boundaries. Each one, if crossed, turns hasp into a different and worse
project."* One of the nine is *"**Not a passphrase manager.** hasp never asks for, stores, or
transmits a passphrase."*

`new key` generates a keypair. A generated key either carries a passphrase or it does not, and
hasp is the process authoring the file, so something must supply the answer. The honest starting
point — stated in [T6](tech-decision-log.md#t6), which corrected an earlier draft of itself for
getting exactly this wrong — is that **nothing in the ratified design licenses hasp asking.**

[D14](#d14)'s carve-out is the sentence that looks like it might: *"hasp never writes to a private
key file. The sole exception is generating a new key, where hasp authors the file outright."* It
licenses hasp **authoring** the file. It says nothing about **asking** for a passphrase while
doing so. Those are different acts, and reading the second out of the first is precisely the kind
of citation error this log exists to catch.

So: an amendment, not a reinterpretation.

### Options

**(i) Never ask; flags only.** `--no-passphrase` and `--passphrase-stdin`, no prompt ever. §3.2
survives untouched, since accepting piped bytes is not hasp *asking*. *Cost:* the interactive
path — overwhelmingly the common one — becomes typing a secret into a pipe with no masking, no
confirmation, and no signal that the terminal is waiting. The strictest option produces the worst
interactive experience, and the predictable response is that people stop setting passphrases.

**(ii) Ask, with the boundary narrowed.** A real prompt, no local echo, at generation time only.

**(iii) Ask, and treat the existing carve-out as already permitting it.** Rejected on sight: that
is the citation error above, and adopting it would make this document claim to say something it
does not.

### Decision

**Option (ii).** §3.2's passphrase boundary is **narrowed**, by exactly this case and no other.

hasp may accept a passphrase **only** at generation time, and the narrowing carries four
constraints that are its entire scope:

1. **Generation only.** Never to decrypt, unlock, or re-encrypt an existing key.
2. **Never persisted.** Not in settings, not in metadata, not in a log, not in a backup.
3. **Never transmitted.** §3.2's *"transmits"* is not narrowed at all.
4. **Zeroed after use.** The buffer does not outlive the write.

### Rationale

The boundary was written to keep hasp out of *custody* of secrets — to stop it becoming a vault,
an agent, or a thing you must trust with the keys to everything. A passphrase that exists for one
function call and is then zeroed puts hasp in custody of nothing. It puts hasp exactly where
`ssh-keygen` already stands, for the same few milliseconds, doing the one job hasp is already
ratified to do.

Refusing to narrow would not have protected anything. It would have pushed users toward keys with
no passphrase at all — a worse security outcome produced by the stricter rule, which is the
failure mode where a principle held literally defeats the purpose it was written for.

**The enforcement is structural, not documentary.** No function in hasp's domain or application
layer accepts a passphrase alongside an *existing* key value. That absence is the mechanism: a
diff adding one is a visible violation on sight — [P6](#d12)'s legibility applied to a safety
property rather than to data.

### Consequence

- **§3.2 and P3 both change**, and `docs/design.md` states the narrowed form in both places. The
  bullet remains a hard boundary with one named exception; it does not become a preference.
- **Every existing citation of P3-as-passphrase-prohibition is unaffected, and this is the
  precision that matters.** `docs/design.md` §5.1's derivation gap still refuses to prompt in
  order to fingerprint an *existing* key — that is a read of something already on disk, and D17
  does not reach it. [D14](#d14)'s rejection of comment-marking still stands on exactly the
  ground it was decided on: marking an encrypted key would require decrypting it. **D17 narrows
  the boundary for a file hasp is authoring, and loosens nothing about keys that already exist.**
- Non-interactive callers use `--no-passphrase` or `--passphrase-stdin`, and a machine-wide
  default is expressible in settings ([D16](#d16)). With no mode resolvable and no TTY, `new key`
  **fails closed** rather than silently generating an unprotected key.
- **hasp is still not a passphrase manager.** It cannot store one, retrieve one, or use one to
  open anything, and it has no operation that takes a passphrase together with an existing key.
  The one thing it can now do is hold a secret it never keeps, for a file it is writing itself.

---

<a id="d18"></a>
## D18 — `release` never deletes content, for hosts as for profiles

**Date:** 2026-08-29 · **Status:** Accepted · **Extends:** [D15](#d15)
**Opened by:** [`docs/tdd.md`](tdd.md) §9

### Context

[D15](#d15) gave `release` a requirement: *"`release` must show the file's contents in its
preview, so that removing a marker the user has written in is never a silent loss."* It states
that for **profiles**, because the `.hasp` marker is what D15 was about.

`release host` has no such rule, and needs one more than profiles do. A profile marker holds a
note the user may have written. A host stanza holds *the user's working SSH configuration* — the
hostname, the port, the proxy command, the thing that makes a connection succeed. `adopt host`
wraps an existing hand-written stanza in hasp's markers ([D7](#d7), [D14](#d14)); if
`release host` simply removed what sits inside them, adoption would be a trap — hand hasp a
working stanza, get an empty region back.

The technical design proposed the obvious answer and then flagged that it had no authority to
make it: this document says the rule for profiles and not for hosts. That is a gap in the
ratified design, so it gets an entry rather than an extrapolation.

### Decision

**`release host` moves the stanza out of hasp's marked region and re-inserts it as plain text
immediately after that region. Content is never deleted, only unmanaged.**

Stated once, generally, so it covers the three nouns and any noun added later:

> **`release` is content-preserving.** It withdraws hasp's authority over a resource. It never
> destroys what the resource contains.

### Rationale

This is [D14](#d14)'s own argument for why `release` exists, followed to its end: *"`adopt` moves
files and writes markers. Without an inverse it is a **one-way door**, and a one-way door turns
"try hasp on my real `~/.ssh`" from an experiment into a commitment."* An inverse that hands back
an empty region instead of the stanza is not an inverse — it is a one-way door with an extra step.

It is also [D4](#d4)'s spirit at a different altitude. D4's letter is about key material, and a
host stanza is not irreplaceable the way a private key is. But *hasp has no operation that
destroys what the user cannot get back* is the reason D4 reads as it does, and a hand-written
`ProxyCommand` that took an afternoon to get right sits close enough to that line to be treated
the same way. Backups (P4) would technically recover it — but requiring someone to go digging
through a backup directory to undo an operation named `release` is not a safety property, it is
an apology.

### Consequence

- **`release` is now symmetric across all three nouns**, and the symmetry is worth naming,
  because it is what makes adoption reversible in practice rather than in principle:

  | Noun | `adopt` does | `release` does |
  | --- | --- | --- |
  | **key** | Moves the file into a managed profile directory, leaving a top-level alias | Moves the file back out |
  | **host** | Wraps the stanza in hasp's markers | Re-inserts the stanza as plain text outside them |
  | **profile** | Writes a `.hasp` marker with a `#` header | Removes the marker, showing its contents first |

- **The re-inserted stanza is unmanaged the instant it lands**, so [P2](#d7) protects it again:
  outside hasp's markers it is human territory, preserved byte for byte from that point on.
- **Rigid regeneration ([D7](#d7)) is untouched.** Content inside a marked region is still hasp's
  to rewrite freely. `release` is the act of moving content *out* of the region, not an exception
  to what happens while it is in there.
- A future first-class noun inherits the general rule above, and must say how it satisfies it.

---

<a id="d19"></a>
## D19 — P3 is split: custody stays absolute, access becomes consent-gated

**Date:** 2026-09-03 · **Status:** Accepted · **Narrows:** P3 · **Generalizes:** [D17](#d17)

### Context

P3 bundles three distinct jobs: a scope boundary (hasp is not a vault, agent, or passphrase
manager, §3.2), a data-flow rule (private key bytes never enter hasp's own records), and an
access rule (hasp never asks for a passphrase to open something that already exists). Only the
access rule has ever blocked anything, and it has now blocked twice: [D17](#d17) narrowed it once,
for `new key`. The AWS created-RSA fingerprint scheme — which hashes the *private* key, not the
public half — and the comment on an encrypted OpenSSH key both need it narrowed again, and neither
is a case D17 covers, because both are reads of a key that already exists. Worth stating plainly,
now that it has recurred: the access rule was a reasonable default chosen to limit complexity at
project liftoff, not a requirement anything else in the design actually rests on.

### Decision

Keep the scope boundary and the data-flow rule **absolute and unchanged.** Replace the access rule
with the consent rule — [D17](#d17)'s four constraints, promoted from a special case covering one
write path to the general rule covering every read: hasp may read private key material only when
the user has explicitly asked for an operation that requires it, only for that operation's
duration, held in memory and zeroed after use; never automatically, and never as a side effect of
a read that did not ask for it.

### Rationale

Retiring P3 outright was considered and rejected. It has 18 citations across `docs/` and 4 in the
Go source, and nearly all of them cite the scope boundary or the data-flow rule, both of which
remain true without qualification. Retiring the whole principle would orphan [D14](#d14)'s
reasoning for not annotating existing private keys, §5.1's derivation-gap rationale,
[D17](#d17)'s own narrowing, and [T6](tech-decision-log.md#t6)'s structural audit — which states
that *"a reviewer reading a diff that adds passphrase-accepting code anywhere else in the tree is
reading a P3 violation on sight"* — every one of which is an argument about custody or data flow,
not about access. Splitting the principle preserves all of them; narrowing the whole thing again,
the way D17 did, would not.

### Consequence

The audit heuristic changes shape rather than disappearing. Passphrase-accepting code is no longer
forbidden outside `new key`, so what a reviewer checks at a call site is no longer "does this
exist" but whether it is **user-initiated, scoped to one operation, and zeroed** — the same three
tests D17 already applied to key generation, now the general review question for any read. §5.1's
derivation gap is also no longer permanent for every case it names: a fact that was `unknown`
because deriving it needed a passphrase may become derivable the moment the user explicitly asks.

---

<a id="d20"></a>
## D20 — J2 spans fingerprint schemes, not merely spellings

**Date:** 2026-09-03 · **Status:** Accepted · **Amends:** J2 (§7)

### Context

J2 currently reads *"Punctuation and case shouldn't matter,"* which frames normalization as
cosmetic — as though there were one fingerprint per key and only its spelling varied. That is
false: AWS alone computes an RSA fingerprint one of two different ways depending on a key's
provenance (SHA-1 over the private key if AWS generated it, MD5 over the public key if it was
imported), and a third way again for ED25519, so the same key can carry more than one valid
fingerprint and the console shows exactly one of them. A user pasting a console fingerprint into a
tool that only knows the scheme its own key format natively produces gets silence back — and
silence reads as "you don't have this key," which is a confidently wrong answer, not a missing
one.

### Decision

J2's requirement is matching across every **scheme** hasp knows, not just every spelling.
Normalization spans hash algorithm and encoding, not only punctuation and case — including
stripping base64 `=` padding, which AWS emits and `ssh-keygen` omits.

### Rationale

The failure mode is silent and produces a confident wrong answer, which is worse than an explicit
error. This is exactly the question §2 opens with: a fingerprint from an AWS console, and which of
the user's keys it is.

### Consequence

`find` acquires a scheme registry. A false negative here is now a defect against a ratified
journey, not a missing convenience.

---

<a id="d21"></a>
## D21 — Investigation is a first-class purpose; J10 is ratified

**Date:** 2026-09-03 · **Status:** Accepted · **Adds:** J10 · **Amends:** §1, §3.1

### Context

hasp's documented framing is management, with comprehension (P7) as the read-side goal. But the
project was born from an investigation: a Bash loop running `ssh-keygen -l -f` across `~/.ssh`,
grepping for a fingerprint held in hand from an AWS console. §2 already asks the question that
founding story motivates; what the design has never said is that *answering* it is a distinct kind
of work from managing an arrangement.

### Decision

Name **visibility** as the root purpose, serving two ends: **investigation** (answering a question
whose shape you do not yet know) and **operational management** (arranging what you already have).
Ratify **J10 — Investigate**. ID order is not chronological here any more than it is in the
technical log, where [T27](tech-decision-log.md#t27) was decided first and numbered last: J10 is
numbered last and is v1 scope, while J9 remains post-v1.

### Rationale

An investigation needs *more* data than comprehension does — including signals hasp cannot itself
interpret, because the human is looking for a thread hasp was never taught to see. That is the
opposite of P7's instinct, and it needs its own name so the two do not fight. P7 does not need
amending — it already scopes itself to *"The default output of a read"* — but the inversion is
worth stating rather than leaving implicit: under `--investigate` the human correlating output
*is* the intended workflow, exactly the case P7 elsewhere calls a failure.

### Consequence

`--investigate` has a journey to cite, and future commands may grow the same mode. §2's founding
story is recorded rather than left as folklore.

---

<a id="d22"></a>
## D22 — Confidence is part of the fact; P10 is added

**Date:** 2026-09-03 · **Status:** Accepted · **Adds:** P10 · **Generalizes:** §5.1's `unknown`

### Context

§5.1 already established that hasp reports an epistemic status rather than only a value —
`unknown` is a first-class answer, because asserting an unverifiable fingerprint is worse than
admitting ignorance. Investigation needs the same honesty for the *middle* of that range: an
origin inferred from a matched console fingerprint is certain for RSA, an origin guessed from the
key alone is a hint, and an ED25519 origin can never be more than a hint — AWS's own fingerprint
scheme for ED25519 carries no origin signal at all, because created and imported keys produce an
identical fingerprint.

### Decision

Add **P10 — Confidence is part of the fact.** Every reported fact carries a status from a closed
set: `derived` (read directly from the artifact), `confirmed` (established by matching external
evidence the user supplied), `possible` (consistent with the evidence, not established), `unknown`
(cannot be determined). Weak assertions are permitted and useful, but must be marked as such and
must never be rendered in a way that reads as established.

### Rationale

This one feature needs all four statuses, which is the evidence the taxonomy is right-sized: a
matched RSA console fingerprint is `confirmed`; an ED25519 guess from the key alone is `possible`;
a value read straight off an artifact is `derived`; an underivable value stays `unknown`. Three of
the four words cost nothing new — they are already this project's vocabulary. [T12](tech-decision-log.md#t12)'s
duplicate detection already splits findings into a *confirmed* duplicate and one it reports as
*possible*, "cannot confirm"; `unknown` is §5.1's own word for the derivation gap. Only `derived`
is genuinely new, because nothing before this needed to distinguish "read from the artifact" from
"matched against something the user supplied."

**Alternative rejected:** recording an explicit origin as metadata when hasp creates a key. Three
reasons, the third decisive. §5.8's metadata channel exists only inside marked regions of
configuration files, and §5.9 states plainly that *"there is no per-key marker, because writing
one would mean rewriting a private key file."* Origin-at-creation is also **history**, which §5.7
already places out of scope: *"'This key replaced that one' is history — genuinely underivable,
and out of scope."* And even setting both of those aside, it would help only hasp-created keys —
exactly the population that never needs investigating, because their origin is already known at
the moment of creation.

### Consequence

The confidence vocabulary becomes a **compatibility surface**: a consumer filtering on `derived`
is making a safety decision, so unlike [T29](tech-decision-log.md#t29)'s deliberately re-tunable
`severity`, this vocabulary is closed and permanent. Downstream work must carry it into
[T31](tech-decision-log.md#t31)'s contract table.

---

## Still open

**Nothing.** D1 through D22 are all ratified.

[D13](#d13) closed the last of the original fifteen by putting profile membership in the
filesystem, which makes [D12](#d12)'s "no persisted state" literally true with no asterisk: hasp
declares nothing and derives everything. [D14](#d14) is what made that affordable, by
distinguishing resources hasp merely reports on from resources the user has explicitly handed it.

**D16 through D18 arrived from the other direction — upstream, out of the technical design — and
that is the process working as intended.** Deriving [`docs/tdd.md`](tdd.md) from this document
surfaced two places where an honest implementation needed something the design did not license
([D16](#d16)'s settings file, [D17](#d17)'s passphrase prompt) and one place where the design
stated a rule for one noun that plainly belonged to all three ([D18](#d18)). None of the three
was resolved by reinterpreting a sentence already here. Each is an amendment, which is the only
way this document is allowed to change.

**Deliberately undecided is not the same as open**, and both remaining items are named where they
live: cross-machine synchronization (`docs/design.md` §3.3) and multi-directory key locations
(§10). Neither is a question waiting on an answer; both are doors held shut on purpose.

The technical questions this document deferred are settled downstream in
[`docs/tech-decision-log.md`](tech-decision-log.md) under `T1`–`T39` — including the two
`docs/design.md` §10 called load-bearing: the **metadata format** ([D7](#d7)'s comment channel,
`T10`) and the **configuration parsing approach** (`T2`).

**D19 through D22 arrived from a third direction — from use.** Answering a fingerprint held in
hand from a cloud console needed a hash computed over a *private* key ([D19](#d19)), a `find` that
spans schemes rather than spellings ([D20](#d20)), a name for the investigative work hasp was
actually born from ([D21](#d21)), and a way to say how sure hasp is rather than only what it found
([D22](#d22)). None of the four came from deriving a downstream document; all four came from a
question the tool could not yet answer. [D19](#d19) is the sharpest of them, because it retires a
stance — P3's absolute prohibition on reading an existing key's material — that was a reasonable
default at liftoff and had since been mistaken for a requirement.
