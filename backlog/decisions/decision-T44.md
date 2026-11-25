---
id: decision-T44
title: '`edit key --replace-material` routes through `WriteKeyFile{AllowOverwrite: true}`, not a separate move rule'
date: '2026-09-08'
status: 'Accepted · Amends: T22'
---
### Context

[T22](decision-T22.md)'s Consequence section claims `edit key --replace-material` "deliberately does not
route through `WriteKeyFile`" and instead uses T15's copy → verify → unlink move rule. That is not
what shipped. `WriteKeyFile` (`internal/app/change_writekeyfile.go`) gained an `AllowOverwrite`
field — `false` for `new key`, always; `true` only for `--replace-material` — and
`EditKeyUseCase.planReplaceMaterial` (`internal/app/editkey.go`) builds `WriteKeyFile{...,
AllowOverwrite: true}` `Change`s directly against the key's existing real path.

### Decision

`--replace-material`'s `Plan` is: a `WriteKeyFile{AllowOverwrite: true}` `Change` against the
key's current path, plus a matched `.pub` sidecar `Change` — `WriteKeyFile{AllowOverwrite: true}`
if the incoming material has its own `.pub`, `Remove` if it doesn't but the old key did, and no
sidecar `Change` at all if neither has one. `RequiresBackup()` mirrors `AllowOverwrite` exactly, so
T22's safety property — the one legitimate overwrite is unconditionally backed up first — still
holds; only the mechanism changed.

Ordering matters here too: the `.pub` `Change` is planned *before* the private-key `WriteKeyFile`,
per T20's "any prefix leaves a working state" rule. A partial apply that stops after the `.pub`
step leaves the old, still-matching private key on disk — self-consistent either way. The reverse
order is worse: a successful private-key swap followed by a failed `.pub` cleanup would leave new
key bytes on disk with a stale `.pub` silently misreporting the old fingerprint, a P1 violation
with no `Preview` to have caught it first, since a partial-apply failure happens after `Preview`
already showed both changes as a pair. This reasoning lives in `planReplaceMaterial`'s own comment
in `internal/app/editkey.go`.

### Rationale

T22's stated reason for wanting a *separate* mechanism — this is the one path allowed to overwrite
key material, and it must be unconditionally backed up first — is satisfied directly by
`AllowOverwrite` and `RequiresBackup()`'s coupling on `WriteKeyFile` itself. There is no need for a
second `Change` kind that duplicates `WriteKeyFile`'s own write mechanics for the sake of one
caller. This keeps [D4](decision-D4.md)'s no-operation-destroys-an-irreplaceable-secret
guarantee, backup-first included, expressed in one place instead of two.

### Consequence

- `docs/tdd.md` §4's `WriteKeyFile` type sketch already documents `AllowOverwrite` and cites T22
  for it, so no `tdd.md` change and no code change accompanies this entry — it corrects
  `tech-decision-log.md` alone.
- T22's fail-closed default (`AllowOverwrite: false`) for `new key` is unaffected; only
  `--replace-material`'s own record is corrected.
- The `.pub`-before-private-key ordering is specific to `--replace-material`'s own two-`Change`
  shape, not a restatement of T20's general rule — it exists because this particular pair of
  `Change`s has a genuine ordering question, unlike [T43](decision-T43.md)'s single-`Change` case.
