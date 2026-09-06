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

This guide covers **J1–J8**. J9 (Git/GPG identity) is a post-v1 direction gated behind M4 and has
no commands yet; J10 (`--investigate`) belongs to the M3.6 milestone and is not yet built. Neither
is documented here because neither exists in the CLI today.

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
