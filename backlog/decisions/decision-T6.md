---
id: decision-T6
title: '`new key` passphrase handling: dual mode, `--passphrase-stdin`, fail-closed'
date: '2026-08-28'
status: 'Accepted · Ratified upstream 2026-08-29 as D17'
---
### Context

`new key` needs a way to support a passphrase-protected key, because non-interactive execution
(`--passphrase-stdin`, or a settings-file default, [T7](decision-T7.md)) requires the choice to be
resolvable without a human present to answer a prompt. The honest starting point is that
**nothing in `design.md` licenses hasp asking for a passphrase.** §3.2 states the opposite as one
of nine hard boundaries, each one described as turning hasp into *"a different and worse
project"* if crossed: *"Not a passphrase manager. hasp never asks for, stores, or transmits a
passphrase."* [`design.md` §5.1](../../docs/design.md#51-key)'s carve-out — *"hasp never writes to a private key file. The
single exception is generating a new key, where hasp authors the file outright"*, which
[D14](decision-D14.md) states in the same terms — licenses hasp **authoring** a key file. It
says nothing about
**asking** for a passphrase while doing so; those are different acts, and an earlier draft of
this entry conflated them, presenting the carve-out as if it settled the question it does not
address. That was a citation error, corrected here rather than left standing.

What follows is a deliberate, needed capability — not a reading of an existing exception. It
**narrows a stated hard boundary**, which this project's own discipline treats the same way as
any other gap in the ratified design: named plainly and flagged for ratification, not
reinterpreted into something `design.md` already permits when it does not.

### Decision

`new key` supports **both** answers — no passphrase, or a passphrase supplied at generation
time — and enforces a hard boundary at the type level: a passphrase is accepted **only** at
generation, **never** to decrypt an existing key, is **never persisted anywhere** (not in
settings, not in metadata, not logged), and its buffer is zeroed immediately after use.

Three mutually exclusive flags on `new key`:

| Flag | Effect |
| --- | --- |
| `--passphrase` | Prompt interactively via `golang.org/x/term.ReadPassword(int(os.Stdin.Fd()))` — no local echo |
| `--no-passphrase` | Generate without one, explicitly |
| `--passphrase-stdin` | Read the secret bytes from stdin — for non-interactive/scripted use |

Mode-selection precedence (which of the three applies when none is given as a flag):
**flag → env var (`HASP_NEW_KEY_PASSPHRASE_MODE`, one of `none`/`prompt`/`stdin`) → settings
file (`[new_key] default_passphrase_mode`, [T7](decision-T7.md)) → built-in default.**

The built-in default is **"prompt interactively if stdin is a TTY; otherwise, refuse."** This is
deliberately not "assume no passphrase" — silently generating an unprotected key because nobody
asked is exactly the failure P5/D6's non-interactive-consent language is written against.
**Non-interactive execution with no explicit mode chosen anywhere in the chain is fail-closed**:
`new key` exits with `ExitUsage` ([T14](decision-T14.md)) and a message naming the three flags, rather than
guessing.

Once a mode resolves to `stdin`, the actual secret bytes are **always** read from `os.Stdin` at
the moment of generation — never from the env var (which only ever carries the *mode name*, not
the secret — env vars are readable via `/proc/<pid>/environ` and process-listing tools on some
systems, which is a real leak path a flag argument or an env-carried secret would both have),
never from settings, never from a CLI argument (which would land in shell history and `ps`
output).

### Rationale

The type-level enforcement: nowhere in `internal/domain` or `internal/app` does a function accept
a passphrase alongside an *existing* `Key` value. The only function that accepts passphrase bytes
is the one building a `WriteKeyFile` change inside `NewKeyUseCase.Plan`, which calls
`ssh.MarshalPrivateKeyWithPassphrase(key, comment, passphrase)` and defers zeroing the slice. This
absence is itself the audit mechanism (P6: legible without hasp running — a reviewer reading a
diff that adds passphrase-accepting code anywhere else in the tree is reading a P3 violation on
sight, not a subtle one).

### Consequence

- **An upstream `D17` was recommended from here, and written there** — flagged rather than
  authored, per the ratifying author's own stated process. **Ratified 2026-08-29 as
  [D17](decision-D17.md)**, which records that §3.2's passphrase-manager boundary is narrowed
  by exactly this case, for exactly this reason, and that the mechanics above (generation-time
  only, never for decryption, never persisted, buffer zeroed) are the entire scope of the
  narrowing — not a general license to ask for a passphrase anywhere else.
- `new key`'s flag surface grows by three flags, mutually exclusive, and the full precedence
  resolver stays small — on the order of ~50 lines to walk four sources in order — which is
  exactly why hasp does not depend on `viper` for this (stack decision,
  [`tdd.md` §2](../../docs/tdd.md#2-stack--go-stdlib-first-and-a-dependency-budget)).
- A CI/scripted caller of `hasp new key` **must** pass `--no-passphrase` or `--passphrase-stdin`,
  or must have configured a default in settings ([T7](decision-T7.md)) — there is no path by which a
  script silently produces an unintentionally unprotected key, and no path by which it hangs
  waiting on a TTY prompt that will never come.
- `hasp new key`'s recorded facts (algorithm, format, fingerprint, encrypted-or-not) are derived
  by re-reading the file hasp just wrote through the same [T1](decision-T1.md) code path used for every
  other key — never asserted from the generation request — closing J4's requirement that a new
  key's recorded facts be *identical* to what a fresh inventory of the finished artifact reports.
