---
id: decision-T7
title: 'A settings file, read-only, reaching P9''s last rung'
date: '2026-08-28'
status: 'Accepted · Ratified upstream 2026-08-29 as D16'
---
### Context

P9's taxonomy ladder ends: *"Only as a last resort, a hasp settings file — and no case has yet
required one."* [T6](decision-T6.md)'s non-interactive default-passphrase-mode requirement is **the first
case that does.** This is not a detail to fold quietly into T6 — the project's own discipline
(stated identically in `design.md`'s own preamble) is that a gap in the ratified design gets
*amended*, not reinterpreted, and this is precisely that kind of gap.

*(That is P9 as it stood when this entry was written. [D16](decision-D16.md) — which this
entry prompted — has since amended its final clause, so `design.md` §4 no longer says "no case
has yet required one." The quotation above is preserved as the text this entry actually reasoned
against.)*

[D3](decision-D3.md) already anticipated this: *"hasp's own settings are settings, are not
part of the verb grid, and are edited directly."* This entry is walking through a door D3 already
named, not cutting a new one.

### Decision

- A settings file exists, in **TOML**, with `#` comments — the same convention D15 already
  argued for on the neighboring `.hasp` marker (*"the neighbouring file in that very directory —
  `ssh_config` — uses `#`, as do shell, `gitconfig`, TOML, and YAML"* — P9 rides the convention
  that already exists rather than inventing a second one).
- **Location:** `os.UserConfigDir()/hasp/settings.toml` — stdlib, no XDG library. On Linux this
  is `$XDG_CONFIG_HOME/hasp/settings.toml` or `~/.config/hasp/settings.toml`; on Darwin,
  `~/Library/Application Support/hasp/settings.toml`. Deliberately **outside** `~/.ssh` — it is
  not key material, not arrangement, and must never be swept into the profile scanner or the
  `~/.ssh/.hasp-backups/` rotation ([T8](decision-T8.md)).
- **hasp reads settings and never writes them.** There is no `hasp config set` and no
  `hasp settings set` — "config" is not a noun (D3), and neither is a settings-mutation verb
  added to the grid. The file is edited directly, with an editor, exactly as P6 already promises
  for everything hasp knows.
- **The admission rule**, stated because this file has the exact shape of the project's original
  failure mode: **settings may hold *only* user preferences that (a) cannot be derived, and
  (b) exist to enable non-interactive execution.** They may **never** hold facts about the
  machine (P1/D12 — that is precisely what the abandoned TOML state file held, per
  `docs/project-assessment-2026-08.md` §5.1's "TOML buys a human-editable... state file"), and
  they may **never** hold taxonomy (P9 rungs 1–2 — a profile assignment written into settings
  would be D13's rejected option (iii), readmitted through a different door than the one D13
  actually closed). Anything that fails this test stays a flag.
- **v1 keyset — deliberately minimal**, because nothing else has yet cleared the bar:

  ```toml
  # ~/.config/hasp/settings.toml
  # hasp reads this file. hasp never writes to it.
  [new_key]
  default_passphrase_mode = "prompt"   # "none" | "prompt" | "stdin"
  ```

- **Optional, throughout.** Absence of the file is not an error. Every settings key has a
  built-in default ([T6](decision-T6.md)'s "prompt if TTY, else refuse") that makes hasp fully useful with
  zero configuration, preserving [`design.md` §6.3](../../docs/design.md#63-cross-cutting-requirements)'s
  *"works on first run, on a machine it has never seen, with nothing configured"* exactly as
  written.

### Rationale

TOML reuses the same decoder [T10](decision-T10.md) already needs for metadata fragments — one parsing
dependency, not two. The admission rule is what prevents this from becoming "the TOML state file,
take two": the original sin, per D12's own rationale, was *persisting derivable facts* — this
file structurally cannot, because anything derivable fails the rule on sight and is refused a
place in it.

### Consequence

- **An upstream `D16` was recommended from here, and written there** — flagged rather than
  authored, per the ratifying author's own stated process. **Ratified 2026-08-29 as
  [D16](decision-D16.md)**, which records that P9's last rung was reached, by what case, and
  carries the admission rule as a permanent constraint on this file's growth rather than as this
  session's judgment call.
- Every future PR proposing a new settings key must justify it against the admission rule in its
  own right, not merely add a TOML field because it was convenient — that discipline is the
  entire point of writing the rule down here rather than letting the file grow the way the
  original state file did, one convenient field at a time.
- `hasp` never needs to create this file. `new key` and every other command that consults it
  treats a missing file identically to an empty one.
- **The admission rule is enforced mechanically, not left to review discipline** — a test
  asserts the exact field set of the Go struct settings decode into
  ([`tdd.md` §12](../../docs/tdd.md#12-testing-strategy--a-function-of-a-directory)). This is the same class
  of guard [T1](decision-T1.md) gets from the absence of a decrypt call site and [T13](decision-T13.md) gets from
  `go list -deps`: the rule most directly guarding against a repeat of the predecessor's TOML
  state file does not get to rely on a reviewer noticing a new field.
