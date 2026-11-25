---
id: decision-D5
title: 'An alias is a name hasp materializes as a link'
date: '2026-08-26'
status: 'Accepted'
---
### Context

Aliases exist because something outside your control insists on a name you didn't choose — a
tool that hardcodes `id_rsa`, a script you can't edit. The original implementation discovered
them from symlinks and stored them with globally unique names, but never stated what an alias
*is*.

### Decision

An alias is an **additional name** for a key. hasp discovers aliases when adopting a machine
and materializes them in the world as links. Names and aliases share one flat namespace.

### Rationale

Uniqueness is inherited from the filesystem rather than invented by hasp: within a directory,
two things cannot share a name, and hasp should not pretend to a richer model than the thing
every other tool actually reads.

### Consequence

hasp's alias model can never be more expressive than a directory. This is correct, not
limiting — the directory is the interface every other tool consumes.
