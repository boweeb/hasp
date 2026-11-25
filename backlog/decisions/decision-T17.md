---
id: decision-T17
title: '`Directive` preserves its exact separator and spacing; quote-aware tokenizing'
date: '2026-08-28'
status: 'Accepted — amends T2'
---
### Context

[T2](decision-T2.md) commits to a lossless CST whose contract is `Render(Parse(b)) == b`. The first
`Directive` sketch was `{ Keyword string; Args []string; Trivia []byte }` — which cannot satisfy
that contract. `ssh_config(5)`: *"Configuration directives are separated from their values by
whitespace or exactly one '=' character (which may be surrounded by whitespace)."* So `Port 22`,
`Port = 22` and `Port=22` are semantically identical and byte-different, and a node holding only
`Keyword` and `Args` has nowhere to record which was written.

Because recognized keywords parse into `Directive` even inside ordinary hand-written stanzas —
not only inside hasp's own regions — reconstruction would normalize `Port=22` to `Port 22` and
break [P2](../../docs/design.md#4-principles) on the first real config using that style. A guarantee that
fails on a legal spelling of the most common directive form is not a guarantee.

### Decision

`Directive` retains the raw bytes of everything between and around its tokens: the
keyword-to-value separator verbatim, inter-argument spacing, leading indentation, and any trailing
comment. `Keyword` and `Args` are **derived views** for hasp's own logic; rendering emits the
original bytes unless the directive sits inside a hasp-owned region, where [D7](decision-D7.md)
licenses rigid regeneration.

Tokenizing is **quote-aware**. `ssh_config(5)`: *"'#' outside of a quoted string may be used to
add a comment to the end of a line"* — so a `#` inside a double-quoted value is a value byte, not
a comment boundary, and `Values may optionally be enclosed in double quotes (")`.

### Rationale

The alternative — keep byte-exact `Line` nodes everywhere outside a marked region and parse into
`Directive` only inside regions hasp owns — also satisfies P2, and is simpler. It was rejected
because hasp must *read* semantics from directives it does not own: [T16](decision-T16.md)'s binding
resolution reads `IdentityFile` out of stanzas hasp will never write. Parsing everywhere and
rendering from retained bytes gives both properties at once; parsing only inside regions would
force a second, parallel read-only parser for the outside case.

### Consequence

- `Render` is not "reconstruct from fields" but "emit retained bytes, except where regenerating."
  A future contributor who adds a field to `Directive` and renders from it reintroduces exactly
  the bug this entry exists to prevent — the fuzz test in §12 is what catches them.
- Quote-awareness means the comment scanner is a small state machine, not a `strings.IndexByte`.
  A `ProxyCommand` value containing `#` inside quotes is the fixture that proves it.
