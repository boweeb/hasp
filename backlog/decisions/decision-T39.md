---
id: decision-T39
title: 'Passphrase-gated derivation: explicit, lazy, one passphrase per invocation, degrades without a TTY'
date: '2026-09-03'
status: 'Accepted · Amends: T6 · Amended by: T49'
---
### Context

[D19](decision-D19.md) generalizes [D17](decision-D17.md)'s four constraints from `new
key`'s write path to any read: hasp may read private key material only when the user explicitly
asked for an operation that needs it, only for that operation's duration, held in memory and
zeroed after, never automatically. Two concrete facts need exactly this: [T35](decision-T35.md)'s created-RSA
scheme, which hashes the *decrypted private key* and is therefore `unknown` for every encrypted
key regardless of format unless the user consents to unlocking it; and an OpenSSH-format
encrypted key's comment, when [T38](decision-T38.md)'s agent path finds nothing loaded.

### Decision

**Prompt only under `--investigate`**, and only for a key where a passphrase would unlock
something otherwise completely `unknown` — the created-RSA scheme is the case that matters, since
it is `unknown` in its entirety without it; an OpenSSH-format key's comment alone is often not
worth interrupting the user for, since the fingerprint and algorithm are already derivable without
asking. **Try the agent first** ([T38](decision-T38.md)) — a key already loaded needs no prompt at all.

**One passphrase, tried across every candidate key in the invocation** — D19's ladder, stated
plainly: not one prompt per key, one prompt per run, held in memory and zeroed after, using the
same `x/term.ReadPassword` path [T6](decision-T6.md) already established for `new key`. No new dependency.

**Declared failure stance, and it is the important one: no TTY (for example, `--investigate
--json` in a pipe) DEGRADES — it does not fail closed.** hasp reports what is derivable without
the passphrase and marks the rest `unknown` with a machine-readable reason, e.g.
`passphrase-required-no-tty`, rather than refusing to run at all.

### Rationale

Writes fail closed ([T6](decision-T6.md), [T22](decision-T22.md)) because a wrong write is destructive and irreversible
in the way P4's backups do not fully undo for key material. A read has no such asymmetry — the
worst outcome of proceeding without a passphrase is an honest `unknown`, which is exactly what
[D12](decision-D12.md) already treats as the correct answer when a fact cannot be derived. A
read that exits non-zero because nobody was present to type a passphrase would make
`--investigate` useless in precisely the scripted, non-interactive workflows it exists to serve —
the opposite failure from the one `new key`'s fail-closed default guards against.

**This is a deliberately different stance from `new key`'s fail-closed non-interactive path
([T6](decision-T6.md)), and the asymmetry is the point, not an inconsistency.** `new key` fails closed because
silently generating an unprotected secret is worse than refusing. `--investigate` degrades because
silently under-reporting a fact you already told hasp to go looking for is a worse failure than
`new key`'s would be, in the opposite direction — a read that refuses to run at all is *less*
honest than one that runs and says `unknown`, not more.

### Consequence

- [T6](decision-T6.md)'s index is annotated as amended by this entry: T6's own fail-closed default remains
  correct and unchanged for `new key`'s **write** path; this entry is the citation that keeps a
  future reader from assuming T6's stance generalizes to every non-interactive case, including
  reads, which it explicitly does not.
- `tdd.md` §9's `show key --investigate` grid cell inherits this mechanic; no new flag is needed
  beyond `--investigate` itself — the mode's default resolves to "prompt if a TTY is attached,
  else degrade," the mirror image of `new key`'s "prompt if a TTY is attached, else fail closed."
- `tdd.md` §11 gains a fail-open (degrade) row for this case, stated beside T6's fail-closed row so
  the read/write asymmetry is visible in the same table rather than scattered across two.
- A guard test drives `--investigate --json` with stdin redirected from `/dev/null` and asserts
  exit `0`, every derivable fact present, and the undecidable ones marked `unknown` with
  `passphrase-required-no-tty` — never a hang, never a non-zero exit.
