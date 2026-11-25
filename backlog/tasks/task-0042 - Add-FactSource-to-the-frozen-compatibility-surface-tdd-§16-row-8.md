---
id: TASK-0042
title: Add FactSource to the frozen compatibility surface (tdd §16 row 8)
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - docs
  - release
milestone: m-6
dependencies: []
references:
  - docs/roadmap.md#56-m36--investigation
type: docs
ordinal: 42000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The final review of M3.6 found `domain.FactSource` carried the same closed-set contract as `Confidence` while §16's table named only `Confidence`. The decision owed before the tag: add a row or record why not. Resolved by adding row 8 ([T31](../decisions/decision-T31.md), [T37](../decisions/decision-T37.md), [T38](../decisions/decision-T38.md)).
<!-- SECTION:DESCRIPTION:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: `6103c94`. Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
