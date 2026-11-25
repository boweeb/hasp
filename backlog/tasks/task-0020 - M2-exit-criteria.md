---
id: TASK-0020
title: M2 exit criteria
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
updated_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - write-path
milestone: m-3
dependencies: []
references:
  - docs/roadmap.md#4-m2--curation
type: feature
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The exit criteria `docs/roadmap.md` §4 listed for M2, kept here as the completion record. Explicitly out of scope for M2: anything touching a host stanza or host-group file, and every `WriteRegion` change.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 `adopt` a key, then `release` it, and the key directory is byte-identical to where it started: the two-way door proven, not asserted ([D14](../decisions/decision-D14.md)).
- [x] #2 `hasp new key`'s reported facts are identical to what a fresh `hasp show key` of the finished artifact reports (J4).
- [x] #3 Every write previews first; with no TTY and no `--yes`, every write fails closed.
- [x] #4 A backup exists for every change that modified something hasp did not create (P4).
- [x] #5 The injected-failure test: interrupt `adopt` at each step and assert a complete, readable key remains at its original path every time ([tdd §12](../../docs/tdd.md)).
- [x] #6 The settings field-set test passes, and `~/.config/hasp/settings.toml` is byte-identical before and after the entire suite.
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: released as v0.2.0 (merge `cd2abce`). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
