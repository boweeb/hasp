---
id: decision-T24
title: 'Line-terminator handling in the CST: preserved as trivia, excluded from `Args`'
date: '2026-08-28'
status: 'Accepted — amends T2'
---
### Context

[T2](decision-T2.md)'s contract is `Render(Parse(b)) == b`, fuzzed. That proves byte fidelity and nothing
else — and byte fidelity is not the property that breaks first.

A CRLF-terminated config parsed by a tokenizer that splits on space and tab would carry the
trailing `\r` into the last argument. `IdentityFile ~/.ssh/work/id_ed25519\r` renders back
byte-identically — the fuzz test passes — while `filepath.EvalSymlinks` fails on the path, and
[T16](decision-T16.md)'s binding resolution reports a dangling target for a key that is sitting right there.
A silent semantic corruption that passes the stated correctness gate.

### Decision

The line terminator is **trivia on the node** — `\n`, `\r\n`, or absent at EOF — preserved for
rendering and **excluded from `Args`**. §12 adds a **semantic** assertion alongside the byte
identity one: parsing a CRLF fixture must yield `Args` identical to parsing its LF twin.

Shape fixtures are committed for the cases that break naive tokenizers: CRLF throughout, mixed
terminators in one file, no trailing newline at EOF, and a zero-byte file.

### Rationale

The general lesson is worth stating once, here, because it applies to every future CST change: a
round-trip test proves you did not *lose* information, never that you *interpreted* it correctly.
The two properties need two tests. Dotfiles repositories shared across machines make CRLF a
realistic input rather than a hypothetical, and this document already acknowledges that workflow
in [T15](decision-T15.md)'s symlinked-config case.

### Consequence

- A zero-byte config is a valid parse producing an empty file with no nodes — not an error, and
  not a missing file. `hasp` must render an empty region into it correctly on first write.
- Rendering never *normalizes* terminators, including inside hasp-owned regions: a region written
  into a CRLF file matches the file it lives in. Rigid regeneration ([D7](decision-D7.md))
  governs a region's content, not the physical line endings of the file carrying it.
