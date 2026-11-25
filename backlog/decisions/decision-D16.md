---
id: decision-D16
title: 'A settings file exists; P9''s last rung is reached'
date: '2026-08-29'
status: 'Accepted · Amends: D13''s P9'
---
**Opened by:** [`docs/tdd.md`](../../docs/tdd.md) §8, [`T7`](decision-T7.md)

### Context

[D13](decision-D13.md) promoted a ladder to a principle, and gave it a rung nobody expected to climb:
*"Only as a last resort, a hasp settings file. No case has yet required the last resort."*

The technical design found the first case that does. `new key` has to be runnable with no human
present — in a script, in CI — which means the question *"does the generated key carry a
passphrase?"* must be answerable without a prompt. That answer is a **preference**. It is not a
fact about a key, a host, or a profile, so P9's first two rungs have nowhere to put it: no
filesystem layout expresses it, and no `ssh_config` directive carries it. A flag answers it for
one invocation. Nothing answers it for *always, on this machine.*

This entry exists because the alternative is worse. The predecessor's TOML file arrived as an
implementation detail — [D12](decision-D12.md) records it as *"a debug and development aid, for visibility
into what the tool was seeing"* that *"quietly became the state model."* A settings file that
arrives the same way, named only in a downstream technical document, is the same story with the
same ending. If the last rung is going to be climbed, it gets climbed here, on the record.

### Options

**(i) Flags only.** Every invocation states its intent; nothing is remembered. *Cost:* a scripted
caller repeats itself forever, and there is no way to express a standing preference for this
machine. Defensible — it is what hasp does for everything else.

**(ii) An environment variable only.** `HASP_NEW_KEY_PASSPHRASE_MODE`, set in a shell profile.
*Cost:* ambient state that `ls` cannot show you and no editor can find, which is the opposite of
[P6](decision-D12.md). It is also per-shell rather than per-machine, so hasp's behavior would depend on how
the process happened to be started — a worse property than the one it is fixing.

**(iii) A settings file.** The last rung. *Cost:* a file with hasp's name on it, which is the
shape of the thing this project has already failed to maintain twice.

### Decision

**Option (iii) — the rung is climbed, once, and gated.**

- **Format and location.** TOML, `#` comments, at `os.UserConfigDir()/hasp/settings.toml`.
  Deliberately **outside `~/.ssh`**: it is not key material and not arrangement, so it must never
  be swept into the profile scanner or into the backup rotation.
- **hasp reads settings and never writes them.** There is no `hasp config set` and no
  settings-mutation verb of any kind. [D3](decision-D3.md) already said so — *"hasp's own settings are
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
TOML state file held, and [D12](decision-D12.md) exists to keep them out. *Taxonomy* is [D13](decision-D13.md)'s
rejected option (iii) — the hand-authored manifest — which would otherwise walk back in through a
door D13 never closed, because D13 was arguing about profile membership and not about this file.

### Rationale

The rung is climbed because the case genuinely clears the bar P9 sets: show that existing
structure cannot carry the fact. It cannot. A passphrase-mode preference is a fact about *the
user*, and P9's first two rungs carry only facts about *the machine*.

The file is safe because of the admission rule, not because of anyone's good intentions. The
original sin, in [D12](decision-D12.md)'s own words, was **"persisting derivable facts, not persistence as
such."** A file that structurally cannot hold a derivable fact cannot repeat that sin: anything
derivable fails the rule on sight and is refused a place in it.

And the rule is not left to review discipline. [`docs/tdd.md`](../../docs/tdd.md) §12 requires a test
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
