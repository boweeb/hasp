---
id: TASK-0008
title: >-
  sshconfig CST, read side: Parse, Render, marked-region recognition,
  MarkerDefect reporting
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - hosts
milestone: m-2
dependencies: []
references:
  - docs/roadmap.md#3-m1--inventory
type: feature
ordinal: 8000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The `sshconfig` CST, read side only ([T2](../decisions/decision-T2.md)): `Parse` plus `Render`, marked-region recognition, and `MarkerDefect` reporting. `Render` is in scope although nothing writes yet: `Render(Parse(b)) == b` is how `Parse` is proven lossless, and the fuzz test stands between M3 and a P2 violation.
<!-- SECTION:DESCRIPTION:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: released as v0.1.0 (merge `c595bc2`). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
