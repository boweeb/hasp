---
id: decision-D3
title: 'The nouns are key, host, and profile; "config" is not a noun'
date: '2026-08-26'
status: 'Accepted — refined by D10'
---
### Context

"Config" meant two unrelated things: hasp's own settings, and the SSH configuration file. The
original implementation had both, under one word, and its verb grid carried a `config` noun
whose meaning was never pinned down — every one of its subcommands was an unimplemented stub,
which is itself evidence that nobody knew what it was for.

### Decision

**Drop `config` as a noun.** SSH configuration is the rendered consequence of hosts and
profiles; it is managed by managing hosts. hasp's own settings are *settings*, are not part of
the verb grid, and are edited directly.

### Rationale

An entity nobody can define is an entity that will be implemented inconsistently. Splitting the
overloaded word removes the ambiguity, and the remaining nouns each have an obvious referent.

### Consequence

Every scenario the old `config` noun was going to cover belongs to `host` or to `settings`.
[D10](decision-D10.md) refines this by establishing that the three nouns are the *first-class* set, and
that further nouns exist beneath them.
