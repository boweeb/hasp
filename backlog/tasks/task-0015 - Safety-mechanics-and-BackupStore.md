---
id: TASK-0015
title: Safety mechanics and BackupStore
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - write-path
milestone: m-3
dependencies: []
references:
  - docs/roadmap.md#4-m2--curation
type: feature
ordinal: 15000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The safety mechanics ([tdd §11](../../docs/tdd.md)): atomic write, symlink-through, mode preservation, D4's copy, verify, replace move rule, and the witness that catches the preview/apply race ([T30](../decisions/decision-T30.md)). `BackupStore` ([T8](../decisions/decision-T8.md)).
<!-- SECTION:DESCRIPTION:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: released as v0.2.0 (merge `cd2abce`). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
