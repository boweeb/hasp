---
id: decision-D14
title: 'Managed vs. unmanaged resources; `adopt` and `release`'
date: '2026-08-27'
status: 'Accepted · Amends: D10, J1 · Enables: D13'
---
**Carve-out narrowed by:** [D17](decision-D17.md)

### Context

[D13](decision-D13.md) put profile membership in the filesystem, which requires moving key files — the
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

- **Profile** — a directory carrying a `.hasp` marker file ([D13](decision-D13.md), [D15](decision-D15.md)).
- **Host** — a stanza inside hasp's markers ([D7](decision-D7.md)).
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
file**, which is the closest hasp would ever come to [D4](decision-D4.md)'s line. Not destruction, but the
one operation where a bad moment costs an irreplaceable secret.

**Location is sufficient anyway.** Under [D13](decision-D13.md) location *is* the taxonomy, so an intrinsic
marker is redundant with [P1](decision-D12.md) — if a key moves, it moved, and the world is the truth. It
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
contradicts the spirit of [D4](decision-D4.md) and [P6](decision-D12.md) — nothing should be trapped inside hasp.
Mechanically the inverse is cheap: move back, remove the marker.

### Consequence

- **[D10](decision-D10.md)'s grid grows from 3 × 6 to 3 × 8.** `adopt` and `release` are first-class verbs
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
