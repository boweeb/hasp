---
id: decision-T16
title: 'Implicit default-identity probing is a distinct, labelled binding kind'
date: '2026-08-28'
status: 'Accepted — amends T5'
---
### Context

[T5](decision-T5.md) derives a host's profile membership from the keys its `IdentityFile` lines resolve to.
That rule is correct and stays — but as first written it modelled only **explicit** `IdentityFile`
directives, and `ssh` does not require one. `ssh_config(5)` states the fallback plainly: *"The
default is ~/.ssh/id_rsa, ~/.ssh/id_ecdsa, ~/.ssh/id_ecdsa_sk, ~/.ssh/id_ed25519,
~/.ssh/id_ed25519_sk and ~/.ssh/id_mldsa44_ed25519."*

A stanza with no `IdentityFile` is not an exotic case; it is the ordinary shape of a hand-written
config, which is precisely the artifact [J1](../../docs/design.md#7-journeys) points hasp at. The omission
was self-contradictory as well as wrong: `adopt`'s own rationale ([D13](decision-D13.md))
turns on leaving a top-level alias *because* `ssh` probes those names, so the write path already
depended on a behaviour the read path did not model.

### Decision

Model **two binding kinds**, carried as `Binding.Kind` through `internal/domain` and into the JSON
envelope: `Explicit` (an `IdentityFile` directive resolved per §5) and `ImplicitDefault` (a member
of the default set above that exists in the key directory).

Exactly one thing suppresses the implicit set: **`IdentityFile none`**. `IdentitiesOnly yes` does
**not** — it constrains which identities an *agent* may offer beyond *"the configured
authentication identity and certificate files (either the default files, or those explicitly
configured...)"*, and the default files are named there as still in scope.

### Rationale

Merging the two kinds into one undifferentiated set would answer [J8](../../docs/design.md#7-journeys)
correctly but report it dishonestly — "you wrote this binding" and "`ssh` would fall back to this
binding" are different facts about the machine, and P1's whole claim is that hasp reports what is
actually there. Keeping them labelled costs one enum field and preserves the distinction for any
consumer that needs it.

Not modelling implicit bindings at all was the alternative, and it fails twice over: `check`'s
"host bound to no key" finding would fire on nearly every stanza in an uncurated `~/.ssh` —
turning J1's first impression into a false-positive flood — and J8 would silently omit hosts that
genuinely depend on a departing profile, which is the one journey where an omission is expensive.

### Consequence

- `check`'s **"host bound to no key"** finding fires only when *neither* kind resolves to
  anything. A stanza relying on default probing is correctly reported as bound.
- A key at the top level of the key directory can now confer profile membership on a host without
  any directive naming it. This is correct — it is what `ssh` actually does — but it means
  `adopt`'s alias symlink is load-bearing for *reporting*, not only for connectivity.
- The default list is version-sensitive: OpenSSH has both added (`id_mldsa44_ed25519`) and removed
  (`id_dsa`) entries over time. It is recorded here as read from `ssh_config(5)` on the
  development host and must be re-checked against the man page rather than from memory.
