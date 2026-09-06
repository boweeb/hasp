# Security

This document restates guarantees hasp's design already makes — it does not create new policy.
The authoritative sources are [`docs/design.md`](docs/design.md) (see especially §3.2 "What hasp
is not" and principle P3 in §4) and [`docs/decision-log.md`](docs/decision-log.md) (D17 and D19).
If this file and those disagree, the docs under `docs/` are the source of truth.

## The custody rule: absolute

Private key bytes never enter hasp's own records, no matter how or why hasp touched them. hasp
reads and records *metadata about* key material — fingerprints, algorithms, comments, formats,
locations — and it writes *arrangement* — names, aliases, membership, references. It never takes
custody of the secret itself.

## The access rule: consent-gated

hasp may read private key material only when you have explicitly asked for an operation that
requires reading it, only for that operation's duration, held in memory and zeroed after use.
hasp never reads private key bytes automatically, and never as a side effect of an operation that
did not ask for it. (hasp also never *writes* to a private key file it did not itself create,
independent of consent — consent gates reading, not writing.)

## What hasp is not

These boundaries are hard, and none of them is new here — they're restated from
[`docs/design.md` §3.2](docs/design.md#32-what-hasp-is-not):

- **Not a secret store or vault.** hasp never takes custody of private key material. Key material
  lives where it already lives; hasp records facts *about* it.
- **Not an agent.** hasp does not hold, cache, forward, or broker credentials at connection time.
  `ssh-agent` exists and is not the problem.
- **Not a passphrase manager.** hasp never asks for, stores, or transmits a passphrase — with two
  narrow, consent-based exceptions, detailed below. Only one of them exists in the shipped CLI
  today.
- **Not a keyring replacement.** For GPG specifically, hasp is an overlay that is aware of keys,
  never an authority over them.
- **Not configuration management, not a multi-user or team tool, not a key distribution
  mechanism, not a connection tool, and not a destroyer of key material.** hasp manages one
  laptop, for one person, and has no operation that can destroy an irreplaceable secret — removing
  key material is your own act, with your own tools.

## The two access exceptions, and their constraints

Both exceptions are bound by the same four constraints — [D17](docs/decision-log.md#d17)'s,
generalized by [D19](docs/decision-log.md#d19) from a single write path to every read:

1. **Explicit request only.** hasp never prompts or reads as a side effect of an operation that
   didn't ask for it.
2. **Never persisted.** Not in settings, not in metadata, not in a log, not in a backup.
3. **Never transmitted.** The passphrase (or, for the second exception, the key material read)
   never leaves the process.
4. **Zeroed after use.** The buffer does not outlive the operation it was collected or read for.

**Shipped today:** `new key` is the only place hasp asks for a passphrase, and only at the moment
it is authoring a brand-new key file — never to decrypt, unlock, or re-encrypt an existing one
(D17).

**Designed, not yet built:** `--investigate` will read an existing key's material with your
explicit consent, for that operation's duration only (D19). It is [M3.6 —
Investigation](docs/roadmap.md#56-m36--investigation) scope, shown as "Next" in the Status table
at [`roadmap.md` §1](docs/roadmap.md#1-how-this-document-sequences-work) — it does not exist in
the CLI yet, and this document will be updated once it ships.

## Reporting a vulnerability

hasp is a solo, one-laptop-scale project with no formal security team, so keep this simple and
honest rather than expecting a formal process: please open an issue at
[github.com/boweeb/hasp](https://github.com/boweeb/hasp). If what you've found is sensitive, say
so in the issue and avoid posting exploit details or proof-of-concept code publicly — there's no
dedicated security contact or PGP key at this time, just say what you need to say and it will get
attention.
