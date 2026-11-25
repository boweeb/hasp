---
id: TASK-0024
title: The metadata channel
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - hosts
milestone: m-4
dependencies: []
references:
  - docs/roadmap.md#5-m3--configuration
type: feature
ordinal: 24000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The metadata channel ([T10](../decisions/decision-T10.md), [T25](../decisions/decision-T25.md)). The mechanism is built because [D7](../decisions/decision-D7.md) requires it be available; its keyset is empty, and adding a field needs its own decision.
<!-- SECTION:DESCRIPTION:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: released as v0.3.0 (merge `8b13b0a`). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
