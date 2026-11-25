---
id: decision-D13
title: 'How profile membership is expressed in the world'
date: '2026-08-26'
status: 'Accepted (2026-08-27) · Opened by: D12'
---
**Adds:** P9 · **Depends on:** [D14](decision-D14.md) · **P9 amended by:** [D16](decision-D16.md)

### Context

[D12](decision-D12.md) removed persisted state on the finding that nearly everything hasp reports is
derivable from the machine. Profile membership is the exception: nothing on disk says that
`id_rsa_foobarco` is a *work* key. It must therefore either be carried by the world in some
structural form, or be the one thing hasp declares.

### Options

**(i) The filesystem carries the taxonomy.** A profile is a directory:
`~/.ssh/work.foobarco/id_rsa`. Tools that hardcode `~/.ssh/id_rsa` are satisfied by a mechanism
already ratified — aliases ([D5](decision-D5.md)) are symlinks, so the top-level name remains and points
into the profile directory.

This completes a pleasing symmetry with [D9](decision-D9.md): **directory = profile, file = host group,
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

**(iv) Metadata in a hasp-owned region.** *Added by [D7](decision-D7.md), after this entry was written.*
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
- Under (i) or (ii), where does a profile's non-SSH identity live when [J9](decision-D11.md) arrives —
  Git identity has no natural home in a key directory.

### Decision

**Option (i): the filesystem carries the taxonomy.** A profile is a directory under the key
directory, and a key's profile membership is the directory it lives in.

**Directories nest; `.hasp` decides.** Profile name segments map to path segments, so
`work.foobarco` is `~/.ssh/work/foobarco/`. A directory is a profile **only if it carries a `.hasp`
marker**, which means `work` can be a profile in its own right, or merely a container that
groups others, and the marker settles which. The marker file was originally required to stay
**empty**, to keep it from becoming a manifest — option (iii) through the back door.
**[D15](decision-D15.md) restates that rule** at the right altitude: what matters is that hasp never
*reads* the file, not that the file has no bytes.

### The general rule this establishes — promoted to P9

The reasoning generalizes well beyond profile membership, so it is promoted to a principle
rather than left as a local answer:

> **P9 — Taxonomy rides on existing structure.** Whenever possible, organizing facts are carried
> by natural structures that already exist — filesystem layout, or the configuration data of the
> tool being managed. Where that is impossible, fall back to private metadata alongside the
> thing. Only as a last resort, a hasp settings file. No case has yet required the last resort.

P9 will govern the Git and GPG expansion ([J9](decision-D11.md)) as much as it governs SSH: the natural
structures there are `gitconfig` conditional includes and repository layout, not a hasp-owned
registry. Recording it as a principle means that question is pre-answered.

### Rationale

Option (i) completes the symmetry begun by [D9](decision-D9.md): **directory = profile, file = host group,
symlink = alias, key file = key, config stanza = host, and nothing else exists.** The filesystem
*is* the database. Everything is visible to `ls`, greppable, and version-controllable without
hasp — P6 satisfied not by choosing a legible format but by having no format at all.

Option (ii) was rejected because a key bound to no host would have no profile, and that is
precisely the key that matters most when offboarding ([J8](decision-D11.md)). Options (iii) and (iv)
were rejected as unnecessary once [D14](decision-D14.md) made (i)'s cost affordable.

### How (i)'s cost was paid

(i)'s stated cost was that hasp becomes opinionated about `~/.ssh` layout, and that adoption
would mean moving key files — awkward against J1's promise that adoption changes nothing.

**[D14](decision-D14.md) pays that cost** by distinguishing managed from unmanaged resources. Unmanaged
resources are reported read-only and are never moved, so the survey journey keeps its promise
exactly. Only a resource the user explicitly adopts is reorganized. Without D14 this decision
would not have been affordable.

### Consequence

- **Profile membership is derived from location.** It is not declared anywhere, which makes the
  model fully derivable — [D12](decision-D12.md)'s "no persisted state" is now literally true, with no
  asterisk.
- Moving a key between profiles is `mv`. The world remains the interface.
- **SSH's default identity probing breaks when a key moves.** `ssh` auto-tries
  `~/.ssh/id_ed25519`, `~/.ssh/id_rsa` and friends; a key that moves into a profile directory
  stops being found by anything that relied on that. Adoption must therefore leave a top-level
  alias symlink behind ([D5](decision-D5.md)) or write an explicit `IdentityFile`. This is a requirement of
  adoption, not a nicety.
- A key in no profile directory sits at the top level and belongs to no profile, which is a
  legitimate state and exactly what `check` should report.
