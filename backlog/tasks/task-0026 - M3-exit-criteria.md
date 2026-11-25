---
id: TASK-0026
title: M3 exit criteria
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
updated_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - hosts
milestone: m-4
dependencies: []
references:
  - docs/roadmap.md#5-m3--configuration
type: feature
ordinal: 26000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The exit criteria `docs/roadmap.md` §5 listed for M3, kept here as the completion record.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Take a real `~/.ssh/config` with hand comments, odd whitespace, and unrecognized directives; have hasp add a host through it; every byte outside hasp's markers is unchanged (P2, J7).
- [x] #2 `adopt host` then `release host` returns the stanza to plain text with its content intact ([D18](../decisions/decision-D18.md)).
- [x] #3 A symlinked `~/.ssh/config` is written through, and the symlink's target inode is unchanged ([T15](../decisions/decision-T15.md)).
- [x] #4 A file with a malformed marker pair is readable (findings reported, defects flagged by kind and line) and unwritable (fail-closed) in one fixture ([T18](../decisions/decision-T18.md)).
- [x] #5 J8 answered end to end: `hasp show profile work` names every key and every host that would break if that profile went away.
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: released as v0.3.0 (merge `8b13b0a`). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
