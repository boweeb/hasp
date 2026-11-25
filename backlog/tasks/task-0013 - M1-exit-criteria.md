---
id: TASK-0013
title: M1 exit criteria
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
updated_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - cli
  - check
milestone: m-2
dependencies: []
references:
  - docs/roadmap.md#3-m1--inventory
type: feature
ordinal: 13000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The exit criteria `docs/roadmap.md` §3 listed for M1, kept here as the completion record. Explicitly out of scope for M1 was every verb that writes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Pointed at the author's real `~/.ssh`, `hasp list key` returns an accurate inventory on one screen (P7).
- [x] #2 `hasp find key` identifies a key from a fingerprint fragment regardless of punctuation and case (J2).
- [x] #3 `hasp show key <name>` lists every host that binds it, with explicit and implicit-default bindings labelled distinctly ([T16](../decisions/decision-T16.md)).
- [x] #4 The write-nothing guard passes: a pre/post snapshot of the entire temp HOME tree is byte-identical after the whole M1 suite runs ([tdd §12](../../docs/tdd.md)).
- [x] #5 The CST fuzz test runs clean in CI over the committed seed corpus.
- [x] #6 `check` exits 1 with findings and 0 clean; the emitted finding ids match the golden list.
- [x] #7 Every read has a `--json` form, and `hasp <any read> --json | jq .version` returns 1.
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: released as v0.1.0 (merge `c595bc2`). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
