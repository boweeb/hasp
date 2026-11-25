---
id: decision-T22
title: '`WriteKeyFile` fails closed on an existing target; a `Remove` change kind'
date: '2026-08-28'
status: 'Accepted — amends T4, T15; amended by T44'
---
### Context

Two gaps in [T4](decision-T4.md)'s `Change` set, both found by verification against
[D4](decision-D4.md).

First: `WriteKeyFile` is the sole operation that authors private key bytes, and [T15](decision-T15.md)'s
atomic-write mechanic ends in `os.Rename`, which Go documents as: *"If newpath already exists
and is not a directory, Rename replaces it."*
No pre-write check said the target must be empty. `hasp new key --name id_ed25519_foobarco` against
a path that already holds a key — a re-run, a typo, a reused name — would silently replace an
irreplaceable secret. D4 is canon: *"hasp has no operation that can destroy an irreplaceable
secret."* An unguarded rename is one.

Second: `release profile` must remove a `.hasp` marker, and no `Change` kind could delete
anything.

### Decision

**`WriteKeyFile` fails closed if its target path exists.** Refused before any byte is written,
with the existing path named in the error. There is no `--force`: an overwrite flag on the one
operation that authors secrets is exactly the affordance D4 forbids, and the user who genuinely
wants the path free has `rm`, which is their own act with their own tools (D4's own reasoning).

**A `Remove{ Path, Reason }` change kind** is added, restricted to artifacts hasp authored — a
`.hasp` marker, an alias symlink. It never applies to a key file. Per
[P4](../../docs/design.md#4-principles) it backs up the removed file first, and per
[D15](decision-D15.md) `release`'s preview shows a marker's contents before removal, so a
marker the user has written notes into is never a silent loss.

### Rationale

Failing closed costs a user who reuses a name one clear error message. Failing open costs a user
one key, permanently, with no backup to restore from — `WriteKeyFile` is a *create*, so there is
no prior version in the backup store to fall back on. The asymmetry is total and the choice is
not close.

`Remove` is deliberately narrow. Giving `Change` a general delete primitive would put the
mechanism for destroying key material into the codebase and rely on call sites never reaching for
it — the opposite of how [T1](decision-T1.md)'s no-decrypt boundary is enforced. Restricting it to
hasp-authored artifacts keeps D4 structural rather than disciplinary.

### Consequence

- §11's failure-stance table carries the row explicitly, and §9's `new` grid cell states the
  refusal, so an implementer meets it in both places.
- `edit key --replace-material` deliberately does **not** route through `WriteKeyFile` — replacing
  material means the target exists by definition. It uses [T15](decision-T15.md)'s copy → verify → unlink
  move rule against the incoming file, with the outgoing material backed up first.
- `Remove`'s restriction is a type-level obligation, not a comment: the constructor takes the
  artifact kind, and no code path constructs one for a key file.
