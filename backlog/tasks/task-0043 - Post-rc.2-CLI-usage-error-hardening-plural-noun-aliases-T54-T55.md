---
id: TASK-0043
title: 'Post-rc.2 CLI usage-error hardening: T54, T55'
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
updated_date: '2026-11-25T16:00:00Z'
labels:
  - backfill
  - cli
milestone: m-6
dependencies: []
references:
  - docs/roadmap.md#56-m36--investigation
type: bug
ordinal: 43000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Unknown nouns and verbs are usage errors with exit 2 ([T54](../decisions/decision-T54.md), `b11ddf3`); flag parse errors and extra positionals on zero-argument leaves are usage errors ([T55](../decisions/decision-T55.md), `2013a73`). Both landed on main after v1.0.0-rc.2 and are unreleased. Context: the plural noun spelling on `list`, `check`, and `find` (`91d11bd`, `93a0c79`) shipped inside v1.0.0-rc.2 (`5a1cb76`).
<!-- SECTION:DESCRIPTION:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: `b11ddf3` and `2013a73` on main, after v1.0.0-rc.2 (`5a1cb76`). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
