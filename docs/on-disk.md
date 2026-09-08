---
Status: New
DateCreated: 2026-09-06
Related:
  - "[`docs/README.md`](README.md)"
  - "[`docs/decision-log.md`](decision-log.md)"
  - "[`docs/tech-decision-log.md`](tech-decision-log.md)"
  - "[`docs/user-guide.md`](user-guide.md)"
---

# hasp — What hasp leaves on disk

[P6](design.md#4-principles) states the promise this page exists to make literal: *"Everything
hasp knows must be visible and repairable without hasp — with `ls`, an editor, and the tools that
were already there. If hasp breaks, or is abandoned again, the user loses a convenience and
nothing else."* Nothing hasp writes needs hasp to read it back. This page is the plain-language
inventory of every artifact hasp leaves behind, and, at the end, the complete path back to a
`~/.ssh` with no hasp in it at all.

## `.hasp` marker files

A **profile** is a directory under your key directory that carries a `.hasp` marker file
([D14](decision-log.md#d14)). The marker is what makes the difference between an unmanaged
directory (hasp only ever reports on it) and a managed one (hasp has full write access, and
features that depend on complete metadata become available).

On disk, it is exactly what it looks like: an empty-or-nearly-empty file named `.hasp` sitting in
the profile directory, e.g. `~/.ssh/work/foobarco/.hasp`. Run `ls -la` in a profile directory and
you'll see it sitting next to the keys it contains. hasp reads only whether the file
*exists* — never its contents ([D15](decision-log.md#d15)). That means:

- **You may write in it.** hasp never parses, validates, or reports on anything you put there.
  It's yours — a note to your future self is a perfectly good use of it.
- **Any line you add should start with `#`.** That's the one convention hasp itself follows when
  it creates the file — the same comment character `ssh_config`, shell, `gitconfig`, TOML, and
  YAML all already use. `adopt profile` writes a short `#`-prefixed header explaining what the
  file is and that its presence, not its contents, is the signal.
- **`release profile` shows you the file's contents before removing it**, specifically so that a
  note you wrote never disappears silently.

A **key**'s managed status has no marker of its own — it is managed if and only if it lives inside
a managed profile directory. hasp never writes to a private key file it did not itself create, so
there is nothing to mark on the key file safely; location is the only signal, and it is sufficient.
A **host**'s managed status is marked differently — see the next section.

## In-file marked regions

Configuration files like `~/.ssh/config` are different from a profile directory: they're **co-
owned**, written in by hand long before hasp ever saw them. hasp's rule for a co-owned file is
that it owns only the regions it explicitly marks as its own, and everything else is preserved
byte for byte ([D7](decision-log.md#d7)).

Two shapes appear on disk, depending on whether hasp created the file outright or is sharing it:

**A file hasp creates wholesale** — an explicit host group, e.g. `~/.ssh/work.sshconfig` — carries
a single header comment rather than a delimiter pair, because the entire file is hasp's region:

```text
# hasp:owned — this file is fully managed by hasp. Hand edits here are lost on the next write.
# See ~/.ssh/config for how it is included.
```

**The default `~/.ssh/config`**, which you likely already have content in, gets an explicit
begin/end pair instead, so the rest of the file stays untouched:

```text
# >>> hasp:managed >>>
Include ~/.ssh/work.sshconfig
Include ~/.ssh/personal.sshconfig
# <<< hasp:managed <<<
```

Everything between those two sentinel lines is hasp's: it will normalize, reorder, and rewrite
that span freely on every write, because it authored every byte of it. **A hand edit placed inside
the markers between hasp runs is intentionally lost on the next write** — this is a deliberate
trade, not a bug, and the marker lines themselves are the warning. Everything *outside* the
markers — every line above `# >>> hasp:managed >>>`, everything below `# <<< hasp:managed <<<`,
and the entire rest of the file — is untouched, exactly as [P2](design.md#4-principles) promises
for hand-written configuration in general.

A region may also carry structured facts as plain comments, in the form `#:hasp <key> =
<toml-value>` — this is how hasp records something the file format itself has no way to express
(like which profile a host group belongs to), without inventing a database. This only ever
*declares* facts a human decided; it never caches a fact the machine could look up itself.

## The settings file

hasp reads one optional settings file, at a location the Go standard library already defines —
`os.UserConfigDir()/hasp/settings.toml`, i.e. `~/.config/hasp/settings.toml` on Linux or
`~/Library/Application Support/hasp/settings.toml` on Darwin ([T7](tech-decision-log.md#t7)).
It lives deliberately outside `~/.ssh` — it is not key material and not arrangement, so it has no
business in the directory hasp otherwise confines itself to.

Today it holds exactly one thing: the default passphrase mode `new key` falls back to when you
don't pass `--passphrase`, `--no-passphrase`, or `--passphrase-stdin` explicitly
([D16](decision-log.md#d16)):

```toml
# ~/.config/hasp/settings.toml
# hasp reads this file. hasp never writes to it.
[new_key]
default_passphrase_mode = "prompt"   # "none" | "prompt" | "stdin"
```

**hasp only ever reads this file. There is no command that writes it.** If you want a different
default, edit it yourself with a text editor — exactly like every other file hasp knows about. Its
absence is not an error: every setting has a built-in default (`prompt`, if a real terminal is
attached), so hasp works fully with zero configuration.

## `~/.ssh/.hasp-backups/`

Before hasp modifies anything it did not create, it copies the current file into
`~/.ssh/.hasp-backups/`, timestamped — for example, `.hasp-backups/config.20260828T140501Z`
([T8](tech-decision-log.md#t8)). The directory inherits your key directory's own `0700`
permissions, and restoring a backup is a plain `mv` — no hasp command needed, no special tooling,
just the copy sitting there in a directory `ls` already shows you.

**hasp never prunes this directory. It grows without bound, and cleaning it up is entirely your
own responsibility.** Every qualifying write leaves a new timestamped copy behind and none is ever
deleted automatically — there is no retention policy, no age-based cleanup, and no command that
does it for you. If you don't want an ever-growing pile of backups, you need to delete old ones
yourself, the same way you'd clean up any other directory: `rm ~/.ssh/.hasp-backups/config.<old
timestamp>` or `rm -r ~/.ssh/.hasp-backups/` if you're confident you don't need any of them. A
backed-up key file is still key material, so hasp treats "when is it safe to delete" as a decision
only you get to make.

## The full withdrawal path

Because every managed resource got that way by explicit `adopt`, undoing hasp's presence
completely is a literal, mechanical process rather than an aspiration — this is
[D14](decision-log.md#d14)'s "two-way door" and [P6](design.md#4-principles)'s "the user loses a
convenience and nothing else," made concrete:

1. **Release every adopted resource.** For each managed key, host, and profile:
   ```sh
   hasp release key <name>
   hasp release host <pattern>
   hasp release profile <name>
   ```
   Each `release` is content-preserving and returns the resource to exactly the shape an
   unmanaged one has — a key back at a plain location, a host stanza re-inserted as ordinary text
   immediately after where hasp's markers were, a profile directory with its `.hasp` marker
   removed (contents shown to you first). This is the same byte-identical guarantee M2's own exit
   criteria hold `adopt`/`release` to.
2. **Delete `~/.ssh/.hasp-backups/`**, if you don't want to keep any of the backups it
   accumulated — see above; hasp will never do this for you.
3. **Remove any leftover `.hasp` markers by hand**, for any profile directory you chose not to
   release in step 1 (an empty marker file does nothing once nothing reads it).
4. **Uninstall the binary.** Nothing else on your machine refers to it — no daemon, no agent, no
   background process, no state outside `~/.ssh` and the optional settings file above.

At the end of that sequence, `~/.ssh` looks like a `~/.ssh` nobody ever ran hasp against — plain
key files, plain config, plain host groups — because every fact hasp ever recorded already lived
in the world, in a form legible to `ls` and a text editor, and not in some private store only hasp
could read.
