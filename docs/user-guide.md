---
Status: New
DateCreated: 2026-09-06
Related:
  - "[`docs/README.md`](README.md)"
  - "[`docs/design.md`](design.md)"
  - "[`docs/decision-log.md`](decision-log.md)"
  - "[`docs/tech-decision-log.md`](tech-decision-log.md)"
  - "[`docs/on-disk.md`](on-disk.md)"
---

# hasp — User Guide

> **Scope of this document.** This is a hand-written narrative guide to *using* hasp, organized
> around the journeys [`design.md` §7](design.md#7-journeys) defines as the project's success
> stories. It is not a flag reference — every generated page under `docs/cli/` is regenerated
> from the binary itself and checked for staleness in CI, so it is the byte-exact source of truth
> for syntax. This guide exists to show *how* and *why*, in real invocations,
> and to point at the generated reference for the rest ([T34](tech-decision-log.md#t34)).

Every example below assumes a `~/.ssh` you want to inspect or manage. Pass `--key-dir` to point
hasp at a different directory (the flag defaults to `~/.ssh`); every command below also accepts
`--json` for machine-readable output, and every write accepts `--yes` to consent non-interactively
instead of confirming at a prompt.

This guide covers **J1–J8** and, as of [M3.6](tech-decision-log.md#t51), **J10**. J9 (Git/GPG
identity) is still a post-v1 direction gated behind M4 ([D8](decision-log.md#d8)) and has no
commands yet, so it stays out of this guide until it does.

## J1 — Survey

> I have a `~/.ssh` I did not curate. Point hasp at it and get a true inventory without changing
> anything. ([`design.md` §7](design.md#7-journeys))

This is the cold-start journey, and it works entirely read-only — hasp never writes to `~/.ssh`
until you explicitly `adopt` something.

```sh
hasp --key-dir ~/.ssh list key
hasp --key-dir ~/.ssh list host
hasp --key-dir ~/.ssh list profile
```

`list key` reports every key it finds, managed and unmanaged alike — name, kind, and location.
`list host` reports every `Host` stanza across `~/.ssh/config` and any host groups it includes;
scope it to one group with `--group <name>`. `list profile` reports every directory carrying a
`.hasp` marker, with key and host counts. Run all three and you have a complete, honest inventory
of a `~/.ssh` you've never touched with hasp before.

## J2 — Identify

> I have a fingerprint from somewhere else and I want to know which of my keys it is, if any.
> Punctuation and case shouldn't matter. ([`design.md` §7](design.md#7-journeys))

```sh
hasp find key SHA256:abcd1234...
hasp find key ab:cd:12:34...
```

`find key <fingerprint-fragment>` matches ignoring punctuation and case, and recognizes a key
under a fingerprint scheme other than the one hasp would natively compute for it. `find host
<clue>` and `find profile <clue>` do the analogous match-by-fragment for hosts (by pattern, or by
the name/fingerprint of a bound key) and profiles (by name fragment).

## J3 — Comprehend

> Show me every key I have, what kind it is, what it's called, what it's aliased to, which
> profile it belongs to, and what uses it — on one screen. ([`design.md` §7](design.md#7-journeys))

```sh
hasp show key id_ed25519_work
```

`show key <name>` gives full detail for one key: identity, every location it's known by
(including aliases), the profile it belongs to, and every host stanza that binds to it — all on
one screen, per [P7](design.md#4-principles). `show host <pattern>` and `show profile <name>` are
the same idea for the other two nouns; `show profile` aggregates a profile's keys and hosts across
its child profiles by default (pass `--no-recurse` to scope to the named profile alone).

## J4 — Create

> Make me a new key for a given persona, record it correctly, and have its recorded facts be
> *identical* to what an inventory of the finished artifact would report.
> ([`design.md` §7](design.md#7-journeys))

```sh
hasp new key id_ed25519_foobarco --profile work.foobarco --passphrase
```

`new key <name>` generates an ed25519 keypair and never overwrites an existing file. `--profile`
places it directly in that managed profile directory (dotted path, e.g. `work.foobarco`); omit it
for a top-level key. Passphrase handling is explicit: `--passphrase` prompts interactively with no
local echo, `--no-passphrase` generates without one, and `--passphrase-stdin` reads one line from
stdin for non-interactive/scripted use. If you don't pass one of these three, hasp falls back to
the default recorded in your settings file (see [`docs/on-disk.md`](on-disk.md#the-settings-file)),
which itself defaults to prompting.

## J5 — Tidy

> Tell me what's untidy — duplicates, missing public halves, orphans, dangling references — and
> help me fix each one deliberately. ([`design.md` §7](design.md#7-journeys))

```sh
hasp check
hasp check key
hasp check host
hasp check profile
```

`check` (and its per-noun forms `check key`, `check host`, `check profile`) reports untidiness
across managed *and* unmanaged resources alike — duplicate keys, keys missing a public half,
orphaned or unaffiliated keys, dangling or shadowed host references, and empty or unmarked profile
directories. Reporting works on anything it finds. Fixing an unmanaged resource requires adopting
it first, since unmanaged resources are read-only by design ([D14](decision-log.md#d14)):

```sh
hasp adopt key <name-or-clue> --profile work.foobarco
hasp adopt host <pattern>
hasp adopt profile <name>
```

Once a resource is managed, `edit` (J6) and `release` are available on it.

## J6 — Rearrange

> Rename a key without touching its material. Replace a key's material without disturbing its
> name, aliases, or profile membership. Add an alias. Move a key between profiles.
> ([`design.md` §7](design.md#7-journeys))

```sh
hasp edit key id_ed25519_foobarco --name id_ed25519_foobarco_2026
hasp edit key id_ed25519_foobarco --replace-material ./new_key
hasp edit key id_ed25519_foobarco --add-alias foobarco
hasp edit key id_ed25519_foobarco --profile work.other-client
```

`edit key <name-or-clue>` is the one command for all four moves, each gated behind its own flag:
`--name` renames the real file in place; `--replace-material <path>` swaps the private key bytes
without touching name, aliases, or profile; `--add-alias <profile>/<name>` (or a bare name for a
top-level alias) creates an alias, with `--remove-alias` as its inverse — removing an alias never
removes the real key file; `--profile <path>` moves the key's real file into a different,
already-existing managed profile.

## J7 — Configure safely

> Add or change a host entry and *know* that everything I had written in that file by hand is
> still exactly as I left it. ([`design.md` §7](design.md#7-journeys))

```sh
hasp new host foobarco.example.com --hostname 10.0.0.5 --user deploy --key id_ed25519_foobarco
hasp edit host foobarco.example.com --port 2222
hasp edit host foobarco.example.com --group work
hasp adopt host legacy-host.example.com
```

`new host <pattern...>` creates a `Host` stanza; `--group` places it in a named host group
instead of the default `~/.ssh/config` (omit it to write to the default file itself), and
`--hostname`/`--port`/`--user`/`--key` set the corresponding directives (`--key` binds the stanza
to a key by name or clue). `edit host <pattern>` changes an existing managed stanza's directives
(with `--remove-hostname`/`--remove-port`/`--remove-user`/`--unbind` as removals), moves it
between groups with `--group` (an explicitly passed empty string moves it back to the default
file), or rebinds it to a different key with `--key`. `adopt host <pattern>` wraps an existing
hand-written stanza in hasp's markers so it becomes editable this way — everything hasp did not
write, inside or outside its markers, is preserved exactly (see
[`docs/on-disk.md`](on-disk.md#in-file-marked-regions) for what those markers look like on disk).

## J8 — Offboard

> I'm leaving a context. Show me everything scoped to that profile — every key, every host that
> depends on it — so I know what to revoke and what will break.
> ([`design.md` §7](design.md#7-journeys))

```sh
hasp show profile work.foobarco
```

`show profile <name>` (aggregating child profiles by default) is the practical payoff here: one
command lists every key and every host bound to that profile, so you know exactly what to revoke
and what breaks before you touch anything. Once you've decided to withdraw, `release` undoes
`adopt` for each resource in turn — a key back out of its profile directory, a host stanza back
out of hasp's markers as plain text, a profile's `.hasp` marker removed (its contents shown first,
so a note you wrote there is never a silent loss):

```sh
hasp release key id_ed25519_foobarco
hasp release host foobarco.example.com
hasp release profile work.foobarco
```

See [`docs/on-disk.md`](on-disk.md#the-full-withdrawal-path) for the complete withdrawal path,
including what to clean up by hand afterward.

## J10 — Investigate

> I have a clue — a fingerprint, a partial memory, a signal I can't fully interpret — and I want
> hasp to surface everything it can about it, including facts a plain read would not show and
> inferences a plain read would never make, clearly marked with how sure hasp is.
> ([`design.md` §7](design.md#7-journeys))

```sh
hasp show key id_rsa --investigate
hasp show key id_rsa --investigate --json
hasp list key --investigate
```

`--investigate` is a local flag on `show key` and `list key` alone — it does not exist anywhere
else, and it never changes what either command reports without it. It is deliberately the
highest-friction thing hasp does: an investigation is something you ask for on purpose, and its
cost — more work per key, and possibly a passphrase prompt — is paid knowingly. `hasp list key`
and `hasp show key` without the flag are byte-identical to a plain read, exactly as if `--investigate`
never existed.

What the flag adds, per key, is:

- **Every registered fingerprint scheme's value.** hasp's own SHA256 fingerprint is one entry in
  an open registry, not the only one — a console you've never asked hasp about can compute a
  fingerprint under a completely different hash and encoding, and `--investigate` computes all of
  them so a value from anywhere else has something to compare against. On a real key that decision
  looked like this:

  ```
  Investigation: id_rsa
    SCHEME             VALUE                                                        CONFIDENCE  REASON
    aws-created-rsa    fb:9f:8c:50:18:2b:28:a0:ac:f9:53:03:21:89:12:73:16:36:8c:49  derived     -
    aws-imported-rsa   b2:ef:06:eb:b1:a0:df:44:30:66:80:33:bb:0c:25:33              derived     -
    ssh-native-sha256  SHA256:NguyN29o6xX3x9SnR9K0nN6BdEP6iRDuXGf06n5crS0           derived     -
    legacy-ssh-md5     9b:19:fa:eb:88:13:86:1a:0c:2c:08:7b:17:0c:75:a5              derived     -
    Origins:
      aws-ec2-created   possible
      aws-ec2-imported  possible
  ```

  `aws-created-rsa` hashes the *decrypted private* key (SHA-1); `aws-imported-rsa` hashes the
  public key a different way (MD5, PKIX/SPKI encoding); `ssh-native-sha256` is hasp's own scheme
  and also AWS's ED25519 fingerprint; `legacy-ssh-md5` is the older MD5-over-SSH-wire-format
  fingerprint some tools still print. If an AWS console, a colleague's terminal, or an old runbook
  hands you a fingerprint that doesn't match `hasp find key`'s own scheme, it may still match one
  of the other three — that mismatch is expected, not a sign either side is wrong.
- **A confidence-graded origin guess**, from a closed, permanent vocabulary: `derived` (read
  directly), `confirmed` (matched external evidence you supplied — only `find key` can reach this,
  since `--investigate` has no external clue to match against), `possible` (consistent with the
  evidence, not proven), or `unknown` (cannot be determined). An RSA key gets both
  `aws-ec2-created` and `aws-ec2-imported` at `possible`, because both are genuinely consistent
  with an RSA key's evidence and hasp has no way to prefer one over the other with no console
  fingerprint in hand. A non-RSA key (Ed25519, ECDSA, DSA) gets an empty `origins` array — neither
  AWS RSA scheme applies to it, so there is nothing to guess.
- **Whatever `ssh-agent` knows**, if `SSH_AUTH_SOCK` points at a running one: cross-referenced by
  public key against the key hasp is looking at, and — when it matches — the loaded key's
  *comment*, which is the one fact an encrypted OpenSSH-format key's own file can withhold even
  from hasp itself (`golang.org/x/crypto/ssh`'s own passphrase-decrypt path discards it). That fact
  is labelled `agent-sourced`, in `--json` as `"commentSource": "agent-sourced"` beside
  `"agentComment"`, never quietly folded into the key's ordinary `comment` field — it is real only
  for as long as that agent process keeps running, and the label says so. No agent, or nothing
  loaded for this key, degrades silently to whatever is derivable without it; it is never an error.
- **Whatever a passphrase unlocks**, if you have a TTY attached and one is still needed: exactly
  one registered scheme needs the decrypted private key (`aws-created-rsa`), and if nothing else —
  not a plain read, not the agent — has already produced it, hasp asks once, interactively
  (`Passphrase: `, no local echo), and tries that one answer against every key in the invocation
  that still needs it. It never asks per key. With no TTY attached — the default for a script or a
  pipeline — the read degrades instead of failing: whatever is derivable is reported, and the rest
  is marked `unknown` with the machine-readable reason `passphrase-required-no-tty`. The table
  above is the same key *after* a passphrase was supplied at a prompt; run non-interactively, its
  `aws-created-rsa` row instead reads `unknown` / `unknown` / `passphrase-required-no-tty` while
  the other three rows are unchanged, since those need only the public half. The command still
  exits `0` either way — an honest `unknown` is a better answer than refusing to run.

`--json` carries the identical facts under the same `key.list`/`key.show` envelope kinds a plain
read already uses — `--investigate` adds fields (`schemes`, `origins`, `agentComment`,
`commentSource`) rather than a different shape entirely, so a script that already parses
`hasp show key --json` keeps working unmodified if it never looks at the new fields.

None of this is quiet. A plain `hasp list key` or `hasp show key` stays exactly as fast and exactly
as untouched as it always was — `--investigate` is where the cost of a deeper answer lives, and it
lives nowhere else.
