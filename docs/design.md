---
Status: APPROVED
DateCreated: 2026-08-26
DateApproved: 2026-08-28
DateLastReviewed: 2026-08-29
Supersedes: >
  the "Primary Goals" / "Future Ideas" sections of the predecessor project's `README.rst`,
  which is not carried into this repository — see
  "[`docs/project-assessment-2026-08.md`](project-assessment-2026-08.md)"
Related:
  - "[`docs/README.md`](README.md)"
  - "[`docs/decision-log.md`](decision-log.md)"
  - "[`docs/tdd.md`](tdd.md)"
  - "[`docs/tech-decision-log.md`](tech-decision-log.md)"
  - "[`docs/roadmap.md`](roadmap.md)"
  - "[`docs/project-assessment-2026-08.md`](project-assessment-2026-08.md)"
---

# hasp — Design & Vision

> **Scope of this document.** This captures *what hasp is for* and *what it must be true of*.
> It deliberately says nothing about programming language, storage format, parsing strategy,
> libraries, or interface technology. Those decisions are downstream and will be derived from
> this document, not smuggled into it. Where a constraint here sounds technical, it is stated
> as an observable property the finished tool must have — never as a mechanism.
>
> If a future argument about implementation cannot be settled by appeal to something in
> §4 (Principles), §5 (Domain Model), or §8 (Decisions), then this document has a gap and
> should be amended rather than reinterpreted. Every settled question, and the reasoning
> behind it, is recorded permanently in `docs/decision-log.md`.

---

## 1. Thesis

**hasp manages identity.**

Not secrets — *identity*. The distinction is the whole project. A secret is a thing you must
never reveal. An identity is the answer to "who am I being right now, to this machine, in this
repository, from this laptop?" Identity is made of secrets, but it is not itself secret. It is
arrangement, naming, membership, and intent — and all of that is currently scattered across
filenames, a config file nobody wants to touch, and human memory.

SSH is the first and primary domain, because it is where the pain is sharpest and the artifacts
are most concrete. Git and GPG identity are named directions, not v1 commitments (§7).

The name fits: a hasp is the fastener a padlock passes through. It is not the lock, and it is
not the secret behind the door. It is the piece that makes the arrangement possible.

---

## 2. The problem

Accumulate a decade of SSH keys, and you end up where this project's author actually is: a
`~/.ssh` holding thirteen keypairs, two of them symlink aliases, comments ranging from
`jesse@thor` to `no comment`, a mix of RSA and Ed25519, PEM and OpenSSH formats, and a config
file that has grown by accretion and is now slightly frightening to edit.

The questions you cannot answer without an afternoon of `ssh-keygen -l` and squinting:

- A fingerprint appears in a GitHub settings page, an AWS console, a `sshd` log line, a
  colleague's message. **Which of my keys is that?** Do I even still have it?
- **Which of these keys are still used?** By what?
- **Are any of these the same key under two names?** (Two of them are.)
- **Which of these are work and which are personal?** Only the filenames hint, and they lie.
- I'm setting up a new machine. **What actually needs to come with me?**
- I'm leaving a job. **What do I revoke, and what breaks when I do?**
- I want to add a host to my SSH config. **Will I lose the comments I wrote three years ago?**

None of these are hard problems. They are *bookkeeping* problems, and no tool does the
bookkeeping, so it doesn't get done. The cost is not catastrophe; it is a permanent low-grade
tax on knowing your own machine — and, occasionally, a key that should have been revoked
three years ago and wasn't.

hasp exists to make those questions answerable in one command, and to make changing the
arrangement safe enough that you actually do it.

---

## 3. Scope

### 3.1 What hasp is

- **A registry.** An accurate, queryable inventory of the identity material on this machine.
- **A curator.** Safe, reversible operations for arranging that material — naming, aliasing,
  grouping, deduplicating, repairing.
- **A safe editor.** A way to change SSH configuration without fear of losing what is already
  there.
- **A single pane of glass.** One place that answers "what is my SSH situation."
- **A personal tool.** For one human, on one machine, with tens — not thousands — of keys.

### 3.2 What hasp is not

These are hard boundaries. Each one, if crossed, turns hasp into a different and worse project.

- **Not a secret store or vault.** hasp never takes custody of private key material. Key
  material lives where it already lives; hasp records facts *about* it.
- **Not an agent.** hasp does not hold, cache, forward, or broker credentials at connection
  time. `ssh-agent` exists and is not the problem.
- **Not a passphrase manager.** hasp never asks for, stores, or transmits a passphrase — with a
  single exception, at the one moment hasp authors a key file itself. `new key` may prompt for the
  passphrase it is about to write, hold it for that operation, and zero it. Never persisted, never
  reused, never accepted alongside a key that already exists. (D17.)
- **Not a keyring replacement.** For GPG specifically — stated in the original vision and
  reaffirmed here — hasp is an *overlay* that is aware of keys, never an authority over them.
- **Not configuration management.** hasp manages this machine. It is not Ansible, it does not
  reach across a fleet, and it does not manage other people's machines.
- **Not a multi-user or team tool.** No shared state, no server, no permissions model.
- **Not a key distribution mechanism.** hasp does not push public keys to hosts or providers.
  (It may eventually help you *find* the key you need to paste. It will not paste it.)
- **Not a connection tool.** hasp does not wrap, proxy, or replace the `ssh` command.
- **Not a destroyer of key material.** hasp has no operation that can destroy an
  irreplaceable secret. Removing key material is the user's own act, with their own tools —
  hasp simply stops seeing it. (D4, canon.)

### 3.3 Deliberately undecided

Whether hasp ever syncs an arrangement between machines is **open**. It is attractive — it
answers "I'm setting up a new laptop" — and it is also the single fastest route to violating
§3.2. Not in scope for the foreseeable milestones; not foreclosed forever.

---

## 4. Principles

Nine invariants. Every one of them is meant to be sharp enough to settle an argument.

### P1 — The world is the truth; hasp keeps nothing

What exists on disk is authoritative. If a key file is present, it exists, whether or not hasp
knows about it. If a key file is gone, it is gone, whether or not hasp still has a record.
What hasp reports is a *projection* of reality, computed on the spot, and reality wins every
time they disagree — because there is nothing on hasp's side to disagree with it.

*Consequence:* **there is no index.** hasp derives everything it reports from the machine at
the moment it is asked — no database, no state file, no cache. It is a pure function of the
filesystem. Anything hasp would have to remember is a liability that must be justified
explicitly — and after D13, **nothing survives that test.** There is no residue. hasp declares
nothing and derives everything, including which profile a key belongs to (§5.7). (D12, D13.)

### P2 — Hand edits are sacred outside hasp's markers

Configuration files are co-owned by hasp and the human. The human's contributions — comments,
ordering, whitespace, spelling, oddities, things hasp does not understand — survive contact
with hasp **byte for byte**. A tool that reformats your config on write is a tool you stop
trusting on the second write, and this project has exactly one chance to get that right.

**The boundary is the marker (D7).** Inside a hasp-owned region hasp is authoritative: it
wrote every byte, it manages the region rigidly, and **manual edits placed inside it are
intentionally lost**. That forfeit is deliberate, and it is what buys the rigidity — a region
hasp fully controls is a region hasp can regenerate, normalize, and repair. Everywhere else,
the guarantee above is absolute.

Stating the exception is the point. A principle with a silent exception is a principle nobody
can rely on; a principle with a *visible, marked* exception is one you can work beside.

*Consequence:* hasp must be able to round-trip a file it only partially understands, and must
preserve constructs it has no model for — outside its own regions. Inside them, it need preserve
nothing, because it can rewrite the region from scratch (§5.8).

### P3 — hasp manages arrangement, not secrets

Restatement of §3.2 as an operating rule: hasp reads metadata *about* key material
(fingerprints, algorithms, comments, formats, locations) and writes *arrangement* (names,
aliases, membership, references). Private key bytes pass through hasp's hands only when it is
generating a key or moving a file, never into hasp's own records.

**The passphrase rule, stated exactly** (D17): hasp never asks for a passphrase to open something
that already exists. It may ask for one only while *authoring* a new key — the same moment §5.1
already permits hasp to write a private key file at all — and holds it for that single operation.
Everywhere P3 is cited against touching an existing key (§5.1's derivation gap, D14's rejection
of comment-marking), it forbids exactly what it always did.

### P4 — Every write is reversible

Before hasp modifies anything it did not create, the prior state is preserved. Rolling backups,
stated in the original vision, are the mechanism; the principle is that "I can undo that" is
always true. The user should never need to have thought ahead.

### P5 — Reads are safe; writes are previewable

Any operation that only answers a question is guaranteed side-effect-free and always safe to
run, including on a machine hasp has never seen. Any operation that changes something can first
be asked to describe exactly what it *would* change, and says so before it acts.

*Consequence:* the ability to preview is not a convenience flag bolted on later; it constrains
how change is modeled from the beginning.

### P6 — Nothing is trapped inside hasp

Everything hasp knows must be visible and repairable without hasp — with `ls`, an editor, and
the tools that were already there. If hasp breaks, or is abandoned again, the user loses a
convenience and nothing else.

P1 satisfies this almost entirely: what hasp knows *is* the filesystem, which is maximally
legible and already under the user's hand. P6 survives as the standing test applied to anything
that would be added — if a proposal creates a thing only hasp can read, it fails. (D12.)

*Consequence:* a cache requires a **measured** justification, not a plausible one — a real
timing on a real key directory showing a delay a human notices. If ever admitted it must be
disposable, auto-invalidating, never consulted for correctness, and deletable mid-run with no
effect but latency. A cache admitted "just in case" is how the abandoned state file began.

### P7 — Answer the question in one screen

The point of the tool is comprehension. If understanding your SSH situation requires running
five commands and correlating the output yourself, hasp has failed at the thing it exists to
do. The default output of a read is the *useful* answer, not a data dump.

### P8 — The scale is one laptop

Tens of keys. Hundreds of host entries at the extreme. A single human operating interactively.
This licenses simplicity everywhere — and it means that any design justified by performance,
concurrency, or scale is almost certainly solving a problem hasp does not have.

### P9 — Taxonomy rides on existing structure

Organizing facts — which profile a key belongs to, which group a host lives in — are carried by
natural structures that **already exist**: the filesystem's layout, or the configuration data of
the tool being managed. Where that is impossible, fall back to private metadata alongside the
thing itself (§5.8). Only as a last resort, a hasp settings file. (D13.)

This is why profiles are directories and host groups are files: neither needed inventing, both
are visible to `ls`, and both are meaningful to someone who has never heard of hasp.

**One case has now reached the last rung** (D16): `new key` needs a machine-wide default for
whether a generated key carries a passphrase, so that it can run with no human present. That is a
preference about hasp's own behavior, not an organizing fact — and the first two rungs carry only
facts about the machine, which is why neither could hold it. The rung is not thereby open. It is
gated by an admission rule, ratified as part of D16 and restated in §5.10: **settings may hold
only preferences that cannot be derived and that exist to enable non-interactive execution —
never a fact about the machine, never taxonomy.** A second case reaching this rung needs its own
decision; one climb is not a licence for the next.

*Consequence:* the ladder is ordered, and the order is the point. A proposal that reaches for a
hasp-owned registry must first show that the existing structure genuinely cannot carry the fact.
P9 governs the Git and GPG expansion (J9) as much as SSH — the natural structures there are
`gitconfig` conditional includes and repository layout, not a registry of hasp's own.

---

## 5. Domain model

This is the vocabulary. It is the part most in need of ratification, because the original
vision and the original code used different words for the same things and the same word for
different things (§8).

### 5.1 Key

A cryptographic keypair. **The first-class object.**

- Its *intrinsic* identity is its fingerprint — independent of filename, location, or anything
  hasp does. Two files with the same fingerprint are the same key.
- Its *handle* within hasp is a **name**: unique, stable, human-chosen, and by default the
  filename it was discovered under.
- Its observable facts — algorithm, size, comment, fingerprint, format, whether a public half
  is present, whether it is passphrase-protected — are read from the artifact, never asserted
  by hasp.

**A key's profile is where it lives** (§5.3, D13). Membership is not declared anywhere — it is
read off the path, and moving a key between profiles is a move. A key at the top level belongs
to no profile, which is a legitimate state and one `check` should report.

**hasp never writes to a private key file.** The single exception is generating a new key, where
hasp authors the file outright. Marking or annotating an existing private key is forbidden: on
an encrypted key it would require a passphrase (P3 forbids asking), and on a legacy PEM key the
standard tooling silently rewrites the file into a different format — changing `format`, a fact
hasp reports, and potentially breaking tools that require PEM. (D14.)

A key's **name** and its **material** are independent and separately changeable. Renaming a key
must not disturb its material; replacing its material must not disturb its name. (Both appear
as explicit scenarios in the original test outlines and both must remain possible.)

**The derivation gap.** Because hasp keeps no records (P1), every fact must be readable on
demand — and one case cannot be. A key that is *legacy PEM format*, *passphrase-encrypted*, and
*missing its public half* yields no fingerprint. (Modern OpenSSH-format keys are unaffected:
the public half is stored unencrypted inside the private file, so only the comment is lost.)
hasp **reports the fact as unknown**. It does not prompt for a passphrase (P3), and it does not
remember a value it can no longer verify — a stateful design would answer this case worse, by
asserting a fingerprint it could not confirm.

### 5.2 Alias

An additional name for a key.

An alias exists because something outside your control insists on a name you didn't choose —
a tool that hardcodes `id_rsa`, a script you can't edit. hasp models the alias as a *name*,
and materializes it in the world as a link to the real key.

Names and aliases share one namespace, and it is flat: within a key directory, two things
cannot have the same name. This is not a hasp rule, it is the filesystem's rule, and hasp
should not pretend otherwise.

### 5.3 Profile

A persona: who you are being. `personal`. `work`. `work.foobarco`.

A profile is the organizing concept of the entire tool, and it is what makes the Git and GPG
directions coherent rather than bolted-on. A profile gathers:

- the **keys** used when acting as that persona,
- the **hosts** reached as that persona,
- and eventually the non-SSH identity facts — the name and email you commit under, the key you
  sign with.

Profile names are **hierarchical**, written as a path. `work.foobarco` is a profile; `work` is its
parent and is also a profile. There is no separate "group" type — a "profile group" in the
original vision is simply a profile with children. This collapses two concepts into one,
satisfies the original requirement of multiple profiles per context (`work.foobarco`,
`work.acme`), and matches what the original implementation actually did once it flattened its
nested configuration into dotted names.

Membership is **many-to-many**: a key or a host belongs to zero or more profiles. A key
genuinely shared between `personal` and `work.foobarco` is a real situation, and forcing a single
choice would make hasp assert something false about the machine. (D2.)

**A profile is a directory** (P9, D13). Name segments map to path segments, so `work.foobarco` is
`~/.ssh/work/foobarco/`, and a key's profile membership is simply the directory it lives in —
derived, never declared.

**A directory is a profile only if it carries a `.hasp` marker.** That marker settles an
otherwise ambiguous question: `work` may be a profile in its own right, or merely a container
that groups `work.foobarco` and `work.acme` without being one itself.

**hasp reads the marker's *presence*, never its *contents*** (D15). The file's contents belong
to the user, who may write whatever they like in it — commentary prefixed `#`, matching
`ssh_config` in the same directory (P9). hasp creates it with a `#` header saying exactly that,
so the convention is visible the first time anyone opens the file rather than being folklore.

The distinction matters: bytes in a file are not declared state; **bytes a program reads** are.
That is the invariant protecting P1, and it is narrower and more exact than requiring the file
be empty. It also leaves room for configuration to grow at that scope some day — behind a gate,
not a drift (§5.7).

This completes the symmetry begun by D9: **directory = profile, file = host group, symlink =
alias, key file = key, config stanza = host, and nothing else exists.** The filesystem is the
whole model.

### 5.4 Host

A destination you authenticate to: a name or pattern, plus the settings that apply to it.

Hosts are the content of SSH configuration. A host belongs to zero or more profiles, and
references the key it should authenticate with. Hosts are where P2 bites hardest, because host
configuration is the file humans have already written in by hand.

A host has two independent affiliations, and keeping them apart matters: *who it belongs to*
(a profile) and *where it physically lives* (a host group, §5.5).

### 5.5 Host group

**A host group is a physical container for host entries: an SSH config file.** (D9.)

- The default host group is `~/.ssh/config`.
- A custom host group named `foo` is `~/.ssh/foo.sshconfig` — the file is named literally
  after the group.
- The default group is the sole exception to that naming rule.

SSH configuration is already multi-file in practice and the format composes natively, so
modeling that reality directly gives hasp a materialization strategy that is legible from
outside the tool: the grouping is visible by listing a directory, and any other program — or a
human with an editor — reads exactly the structure hasp does. It requires no invented metadata,
and it keeps the grouping a fact about the world (P1) rather than a claim hasp has to store.

**Host groups and profiles are different axes and must not be conflated.** A profile is *who
you are being*; a host group is *where a host entry lives*. They will often correlate, and
materializing a profile's hosts into a group file of the same name may be a sensible default,
but they are distinct concepts and nothing requires them to align.

A group hasp authored is a file hasp authored — a far easier thing to reason about than a
region inside a file someone else wrote. The default group is the opposite case: a file hasp did
not create and cannot claim outright. D7 settled that asymmetry: hasp owns its own group files
wholesale and owns only *marked regions* inside the default (§5.8). Because the co-owned case is
where the sharp edges live, explicit host groups are the path hasp should steer users toward.

### 5.6 Binding

The relationship "this host is reached with this key."

In a config file this is just a directive, but hasp must reason about it as a relation, because
the valuable questions run backwards along it: *which hosts use this key?* — which is what you
need when deciding whether a key can be retired, and what you need on your last day at a job.

### 5.7 Intent — the category that closed

This section once held the residue: the things hasp would have to be *told*, because the machine
could not report them. **It is now empty, and that is the design's strongest result.**

Everything is derivable on demand. Fingerprints, algorithms, formats and comments come from the
key artifacts; aliases from symlinks; hosts, settings and bindings from config files; host groups
from the files themselves (§5.5). The one holdout was profile membership — nothing on disk said
that `id_rsa_foobarco` was a *work* key — and D13 closed it by making the profile a **directory**
(§5.3). Membership is now read off the path like everything else.

The category is kept here, named and empty, as a **standing test**: anything proposed that hasp
would have to be told, rather than able to look up, reopens this section and needs a decision of
its own. The bar is P9 — show that no existing structure can carry the fact.

**The clearest case to watch is the `.hasp` marker** (§5.3). Its contents are the user's today
and hasp never reads them, so nothing is declared. The day hasp *reads* that file, it reopens
this section — deliberately, by decision, rather than one convenient field at a time. That file
has the same shape as the original state file, which began as "just a debug aid for visibility";
the difference is that this door is explicitly a door. (D15.)

**The settings file is not a reopening** (D16, §5.10). It carries a preference about hasp's own
behavior — whether `new key` prompts when nobody is present to answer — and this section is about
*organizing facts hasp would have to be told about the world*. Nothing on the machine is being
declared, so nothing derivable has been displaced and what hasp reports is unchanged. The
distinction is exact, and it is what keeps the standing test meaningful: **a preference is about
the user; intent is about the machine.** A settings file that ever held the second would reopen
this section — which is why D16's admission rule forbids it in writing, rather than trusting the
boundary to stay self-evident.

*Two things that look like intent are not.* "This key replaced that one" is history — genuinely
underivable, and out of scope. "I deliberately keep this unused key" is a suppression: marked
regions (§5.8) would make one expressible, but `check` is deliberately advisory (§6.2) and no
suppression mechanism is planned. That is a scope choice, not a capability limit, and it should
be revisited only if the noise proves real in practice.

### 5.8 Marked region

A **delimited span inside a configuration file that hasp owns outright.** (D7.)

Everything outside a marked region belongs to the human and is preserved byte for byte (P2).
Everything inside belongs to hasp, which manages it rigidly — normalizing, reordering, and
rewriting freely, because it authored all of it. Hand edits placed inside a region are lost on
the next write, by design.

**A region may carry structured metadata as comments.** This is how hasp records anything the
underlying format cannot express natively. The metadata structure must be expressive enough to
capture whatever the human intended, so that a construct hasp does not model natively is
*carried* rather than dropped.

That capability is worth more than it first appears: it means **hasp never needs a complete
native model of the configuration format.** What it understands it manages; what it does not it
carries. The problem shrinks from "model the whole format" to "model what we manage, carry the
rest" — which retires the largest technical risk this design had identified.

**Metadata is declaration, not cache — and the line matters.** Recording a profile assignment in
a comment is legitimate: the machine cannot tell you that. Recording a fingerprint would be a
violation of P1, because the machine can. **Metadata may record what the machine cannot tell
you; it may never cache what the machine can.** Held to that line, comment-carried metadata is
input rather than an index, and does not reintroduce what D12 removed.

**Where regions appear.** A host group file hasp created is owned wholesale — the easy case. The
default `~/.ssh/config` is supported too, but as a co-owned file with marked regions inside it.
Because co-ownership is where the sharp edges live, hasp should **nudge users toward explicit
host groups** (§5.5) as the smoother path, without refusing the default.

### 5.9 Managed and unmanaged

**Every resource is either managed or unmanaged, and the difference is consent.** (D14.)

| | Unmanaged | Managed |
| --- | --- | --- |
| How hasp treats it | Read-only reporting | Full write capability, plus features needing complete metadata |
| Recognized by | Absence of a marker | A marker, per noun below |
| Whose territory | The user's | hasp's, by explicit opt-in |

- A **profile** is managed when its directory carries a `.hasp` marker (§5.3).
- A **host** is managed when its stanza sits inside hasp's markers (§5.8).
- A **key** is managed when it lives inside a managed profile directory. **Location is the only
  signal** — there is no per-key marker, because writing one would mean rewriting a private key
  file (§5.1), and because under D13 location already *is* the taxonomy. A marked key outside a
  managed directory, or an unmarked key inside one, are contradictions that simply cannot arise.

**This distinction is what makes the filesystem-as-taxonomy affordable.** Putting profiles in
directories means adoption moves key files — the most invasive thing hasp does. Unmanaged
resources are never moved and never written, so J1's promise that surveying changes nothing
holds exactly. Only what the user explicitly hands over gets reorganized.

**Managed status is a revocable grant.** `adopt` gives it; `release` takes it back (§6.2).
hasp's authority over any resource is something the user grants and can withdraw.

### 5.10 What is deliberately absent

- **Grouping as taxonomy, other than profiles.** Profiles are the one *organizing* concept.
  A host group (§5.5) is not a counterexample — it is a container, on a different axis. There
  is no "key group" and no "profile group"; both are simply profiles. (D1, D9.)
- **A "config" object.** See D3 — SSH configuration is the *rendered consequence* of hosts,
  groups, and profiles, not a thing you create and edit as an entity in its own right.
- **A state file, database, or index.** hasp persists nothing and derives everything (P1, D12).
  The original state file began as a debug aid, became the state model, then became the
  migration the project died inside. There is no successor to it.

  *The settings file (D16) is not one, and the line is worth drawing precisely.* It is not the
  location or the format that separates them — the abandoned state file was TOML too. It is what
  the file is permitted to contain: **settings hold preferences; the state file held facts.**
  hasp reads settings and never writes them, and nothing in them can drift out of step with the
  machine because there is no machine fact in them to drift. The admission rule is the
  enforcement, not the intention: settings may hold only preferences that cannot be derived and
  that exist to enable non-interactive execution — never a fact about the machine, never
  taxonomy. Anything failing that test stays a flag.
- **Anything declared *about the machine*.** Not merely no *derived* store — after D13 there is
  no *declared* one either. Profile membership, the last candidate, is carried by directory
  layout (P9). The `.hasp` marker is a marker, not a manifest: hasp reads that it exists and
  never what it says (§5.3, D15). Settings are not a counterexample, for the reason the previous
  bullet gives: a preference is not a fact about the machine, and nothing hasp reports about
  `~/.ssh` comes from a file the user wrote.

---

## 6. Capabilities

What a person can do, stated as intent. This is an interaction *grammar*, not an interface
specification — it does not presuppose how the tool is invoked or presented.

### 6.1 Grammar

Operations are **verb + noun**. The verbs are uniform across nouns, so that learning one noun
teaches you the others, and adding a noun later is mechanical rather than inventive. This
uniformity was the strongest idea in the original implementation and it survives ratification
intact.

Nouns come in **two classes** (D10):

- **First-class nouns** — **key**, **host**, **profile**. Entities that exist in their own
  right. The verb + noun grammar applies to these and only these.
- **Second-class nouns** — *alias*, *binding*, *host group*, and others yet to be identified.
  These belong *to* a first-class noun and are addressed as modifiers of an operation on their
  owner, never as operations in their own right. Illustratively:
  `hasp edit key "foo-key" --add-alias="bar-key"`, rather than a top-level `alias` noun with
  its own verbs.

The test for first-class status is **independent existence**. An alias without a key is
meaningless; a binding without a host and a key is nothing. Promoting such concepts to the verb
grid would grow the command surface combinatorially while offering operations that are either
nonsensical or duplicative.

This keeps the grid at 3 × 8 permanently, however rich the domain becomes. Every new concept
must be classified on arrival — "is this first-class?" is a standing design question.

### 6.2 The verbs

| Verb | Intent | Class |
| --- | --- | --- |
| **list** | Show me everything of this kind | read |
| **show** | Show me this one thing in full | read |
| **find** | Which thing matches this clue? | read |
| **new** | Bring a new thing into existence | write |
| **edit** | Change this thing's arrangement | write |
| **check** | Tell me what's wrong or untidy | read |
| **adopt** | Take this unmanaged thing under management | write |
| **release** | Hand this managed thing back; stop managing it | write |

**Two verbs were retired by D12.** `sync` existed to reconcile hasp's records against reality;
with no records there is nothing to reconcile, and every read is already a fresh look at the
machine. `forget` meant "drop this from my records"; there are no records to drop from. Their
disappearance is the clearest confirmation that removing persisted state was correct — `sync`
was the most-worked and least-finished part of the abandoned implementation, and the
abandonment marker sits literally inside it. (D13 may reintroduce a materialization verb; until
it is settled, none is assumed.)

Notes on the four that are new or changed relative to the original:

- **find** is not a filtered `list`. Its purpose is *identification from a partial clue* —
  most importantly a fingerprint encountered somewhere else, in whatever punctuation and case
  it happened to arrive in. Clue normalization is a requirement, not a nicety.
- **check** collects the maintenance questions the original vision listed as scenarios —
  duplicate keys under different names, keys missing their public half, keys belonging to no
  profile, hosts referencing keys that don't exist, fingerprints that cannot be derived (§5.1).
  It reports; repair is a separate, explicit act. `check` is **advisory**: findings are not
  suppressible. Marked-region metadata (§5.8) would make a suppression list expressible without
  violating P1, so this is a deliberate scope choice rather than a limitation — revisit it only
  if the noise proves real.
- **adopt** reorganizes an unmanaged resource into a managed one (§5.9): moving a key into its
  profile directory, wrapping a host stanza in markers, placing a `.hasp` in a directory. It is
  the only operation that moves key files, and it is always explicit. **Adopting a key must
  leave its top-level name reachable** — an alias symlink (D5) or an explicit `IdentityFile` —
  because `ssh` probes `~/.ssh/id_ed25519`, `~/.ssh/id_rsa` and friends by default, and a key
  that moves without that provision silently stops being found. Creating a profile marker,
  `adopt` writes it with a `#` header documenting the file (§5.3, D15).
- **release** is the inverse, and exists so that `adopt` is not a one-way door. Trying hasp on a
  real `~/.ssh` should be an experiment, not a commitment; nothing is trapped inside hasp (P6).
  **`release` is content-preserving across every noun** (D18): a released key's file moves back
  out of its profile directory; a released host's stanza is re-inserted as plain text just outside
  hasp's markers rather than removed along with them; and a released profile's marker is deleted
  only after the preview has shown its **contents**, so a note the user wrote is never a silent
  loss (D15). `release` withdraws hasp's authority over a resource. It never destroys what the
  resource contains.

**hasp still has no operation that destroys key material.** That is canon (D4) and is unchanged
by the loss of `forget`: with nothing persisted, "forgetting" is simply what happens when the
material is gone and hasp next looks.

### 6.3 Cross-cutting requirements

- Every read has a **machine-readable form**. hasp lives in a terminal beside other tools; a
  tool whose output can only be looked at is half a tool.
- **Every write previews by default** (D6) — preview → confirm → back up → write, applied
  uniformly, with no per-operation judgment about which changes are risky enough to warrant it.
  Non-interactive use needs an explicit way to consent in advance.
- hasp **works on first run**, on a machine it has never seen, with nothing configured. There
  is no state to set up before the tool becomes useful, and no first-run initialization step —
  every invocation looks at the machine and tells you what is there (P1).
- Every read is a **fresh scan**. Correctness therefore depends on derivation being both
  complete and cheap, which makes the derivation gap in §5.1 a permanent design fact rather
  than an edge case.

---

## 7. Journeys

The stories that define success. The first seven are the original vision restated; the last
two were proposed as extensions and are ratified (D11).

**J1 — Survey.** I have a `~/.ssh` I did not curate. Point hasp at it and get a true inventory
without changing anything. *This is the cold-start journey and the tool's first impression. It
works entirely on unmanaged resources (§5.9), so the promise that nothing changes is structural
rather than a matter of care.*

**J2 — Identify.** I have a fingerprint from somewhere else and I want to know which of my
keys it is, if any. Punctuation and case shouldn't matter.

**J3 — Comprehend.** Show me every key I have, what kind it is, what it's called, what it's
aliased to, which profile it belongs to, and what uses it — on one screen (P7).

**J4 — Create.** Make me a new key for a given persona, record it correctly, and have its
recorded facts be *identical* to what an inventory of the finished artifact would report.

**J5 — Tidy.** Tell me what's untidy — duplicates, missing public halves, orphans, dangling
references — and help me fix each one deliberately. *Reporting works on anything; repairing
requires the resource to be managed first (§5.9).*

**J6 — Rearrange.** Rename a key without touching its material. Replace a key's material
without disturbing its name, aliases, or profile membership. Add an alias. Move a key between
profiles.

**J7 — Configure safely.** Add or change a host entry and *know* that everything I had written
in that file by hand is still exactly as I left it (P2).

**J8 — Offboard.** I'm leaving a context. Show me everything scoped to that profile — every
key, every host that depends on it — so I know what to revoke and what will break. *This is
the practical payoff of profiles being first-class, and it is the journey that most justifies
§5.6 modeling bindings as a relation.*

**J9 — Unify identity.** *(post-v1, per D8)* A profile carries not just SSH keys but the Git
identity and signing key that go with the persona, so that "who am I being" has one answer
instead of three. *This is the original "Future Ideas" section, promoted from a list of
features to the reason the domain model is shaped as it is.*

---

## 8. Decisions

The questions this document raised have been reviewed and **all eighteen are ratified**.
Settled decisions live in **`docs/decision-log.md`** — the reasoning, the alternatives rejected,
and what each one commits the project to. That file is the permanent record and the place to
look when asking "why is it like this?"; this document is the current truth.

**D1** profile is the single organizing concept · **D2** keys and hosts may belong to many
profiles · **D3** the nouns are key, host, profile; "config" is not one · **D4** hasp never
destroys key material · **D5** an alias is a name materialized as a link · **D6** preview is the
default for every write · **D7** hasp owns only the regions it marks, and those regions may
carry declared metadata · **D8** Git/GPG stays as direction, out of near-term work · **D9** host
groups exist and are SSH config files · **D10** nouns are first- or second-class · **D11**
offboarding and unified identity are in scope · **D12** hasp has no persisted state and is a
pure function of the machine · **D13** profile membership lives in the filesystem, and taxonomy
prefers structures that already exist (P9) · **D14** resources are managed or unmanaged, and
`adopt` / `release` move between the two · **D15** hasp reads a marker's presence, never its
contents · **D16** a settings file exists, gated by an admission rule (P9's last rung) · **D17**
`new key` may ask for a passphrase, at generation time only · **D18** `release` never deletes
content, for hosts as for profiles.

Their substance is folded into §3–§7 above. **No question remains open.**

D13 was the last of the original fifteen to close, and closing it made D12 literally true: hasp
declares nothing and derives everything.

**D16, D17 and D18 arrived afterwards, and from the opposite direction.** Deriving
`docs/tdd.md` from this document found two places where an honest implementation needed something
this document did not license — a settings file (§4 P9) and a passphrase prompt (§3.2) — and one
place where a rule stated for a single noun plainly belonged to all three (§6.2). Each was
amended here rather than reinterpreted downstream, which is the only way this document is
permitted to change. That the technical work sent three questions back upstream is the process
functioning, not a defect in the design it was derived from.

What remains is downstream work — §10 — not amendments to this document.

---

## 9. Definition of done

Milestones in user-visible terms. Each is independently valuable — the project should be
worth using at the end of every one, which is the discipline that was missing the first time.

**M1 — Inventory.** *hasp tells the truth about this machine.*
Survey an uncurated `~/.ssh`, get an accurate one-screen picture, identify a key from a
fingerprint. Everything is unmanaged (§5.9), so **hasp never writes to `~/.ssh` at all** — the
milestone's safety is structural, not a matter of discipline. Covers J1, J2, J3.

**M2 — Curation.** *hasp changes the arrangement safely.*
`adopt` arrives, and with it every write. Create keys, rename, re-alias, re-profile, repair
untidiness. Every change reversible, and `release` undoes adoption itself. Covers J4, J5, J6.

**M3 — Configuration.** *hasp edits SSH config without ever losing a hand edit.*
Manage host entries, groups, and bindings with D7 fully honored. Covers J7, and makes J8
answerable.

**M4 — Identity.** *A profile answers "who am I being" across SSH, Git, and signing.*
Covers J9. Not before M3 is finished.

The tool becomes genuinely useful to its author at **M1**, and genuinely better than the status
quo at **M2**. M1 is therefore the only milestone whose scope should be defended aggressively.

---

## 10. Explicitly deferred

### Closed downstream

Every item below was deferred by this document and has since been settled in
[`docs/tdd.md`](tdd.md), with its reasoning in [`docs/tech-decision-log.md`](tech-decision-log.md).
They are listed here, rather than deleted, because "where did this get decided?" is a question
this document should be able to answer about its own deferrals.

| Deferred item | Settled by |
| --- | --- |
| **Implementation stack** — language, runtime, distribution, dependencies | `T27` (Go, and a rewrite rather than a repair), `T9` (distribution) |
| **Configuration parsing** — the approach satisfying D7, and the composition mechanism behind host groups (§5.5) | `T2` (a lossless CST hasp owns), `T11` (composition by `Include` order) |
| **The metadata format** — the structure carried in hasp-owned regions (§5.8) | `T10` (sentinel-prefixed TOML), `T25` |
| **Interface** — the concrete command surface and output formats | `T3`, `T14`, `T29`; `tdd.md` §9–§10 |

D7 substantially shrank the parsing problem before it was ever solved: because marked regions
carry what hasp does not model (§5.8), the investigation was never "model the whole format" but
"model what we manage, carry the rest." `T2` records that the survey of existing libraries was
done first, and that none offered the round-trip fidelity P2 requires.

**Storage was never deferred at all after D12** — there is nothing to store, so there was nothing
to defer. What remained was a derivation question, not a persistence one: how cheaply and
completely hasp can read the machine (§5.1, §6.3).

### Still deferred

- **Multi-directory / non-default key locations** — assume one key directory for now; the model
  should not foreclose more, and `tdd.md` §15 confirms nothing in the derivation pipeline
  hardcodes a single directory.
- **Cross-machine synchronization** — §3.3. Deliberately undecided rather than merely unscheduled:
  it is the single fastest route to violating §3.2.

---

## 11. Glossary

| Term | Meaning |
| --- | --- |
| **Key** | A keypair. Identified intrinsically by fingerprint, handled by name. |
| **Name** | A key's unique, stable handle within hasp. |
| **Alias** | An additional name for a key, materialized as a link. |
| **Profile** | A persona. Hierarchically named. Gathers keys, hosts, and identity facts. |
| **Host** | A destination and its settings; the content of SSH configuration. |
| **Host group** | A physical container for hosts: an SSH config file. Default is `~/.ssh/config`. |
| **First-class noun** | An entity with independent existence: key, host, profile. Takes verbs. |
| **Second-class noun** | A concept owned by a first-class noun (alias, binding, host group). Surfaces as an argument. |
| **Binding** | The "this host uses this key" relation. |
| **Marked region** | A delimited span in a config file that hasp owns outright and manages rigidly. |
| **Marker** | A file or delimiter whose *presence* signals management. hasp never reads a marker file's contents. |
| **Metadata** | Declared facts hasp records as comments inside a marked region — never derived facts. |
| **Settings** | User preferences hasp *reads and never writes*, in a file outside the key directory. Gated by D16's admission rule: preferences only, never facts about the machine, never taxonomy. Not a state file (§5.10) and not a noun (D3). |
| **Intent** | What the human knows that the machine cannot report. The only thing hasp truly owns. |
| **Derive** | Read a fact from the machine at the moment it is asked for. hasp's only way of knowing anything. |
| **Adopt** | Reorganize an unmanaged resource into a managed one. Moves files; always explicit. |
| **Release** | The inverse of adopt: hand a resource back, undoing the reorganization. |
| **Survey** | Report on a machine without changing anything. The cold-start journey (J1). |
| **Managed** | A resource the user has handed to hasp; hasp may write to it. |
| **Unmanaged** | A resource hasp only reports on. The default for everything it did not create. |

**Retired vocabulary:** *key group*, *profile group* (→ **profile**); *config* as an object
(→ **host**, **host group**, or **settings**); *state*, *index*, and *sync* as concepts of any
kind (→ **derive**; D12 removed the thing they referred to).

*"Host group" is **not** retired — see D9. It was retired in D1's original consequence and
reinstated with a container meaning rather than a taxonomic one.*
