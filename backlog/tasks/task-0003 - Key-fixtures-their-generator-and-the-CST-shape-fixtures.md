---
id: TASK-0003
title: Key fixtures, their generator, and the CST shape fixtures
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - keys
  - ci
milestone: m-1
dependencies: []
references:
  - docs/roadmap.md#2-m0--bootstrap
type: chore
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The full key-fixture set and the dev-only script that generates it ([tdd §12](../../docs/tdd.md)): the algorithm by format by encryption by public-half matrix, plus the hand-constructed DSA fixtures ([T23](../decisions/decision-T23.md)). These gated M1's entire read path. The CST shape fixtures: CRLF, no-trailing-newline, and a 0-byte file.
<!-- SECTION:DESCRIPTION:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: `04d6b8d` (Implement M0 — Bootstrap). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
