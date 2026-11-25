---
id: TASK-0022
title: Marker syntax and host-group composition
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
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Marker syntax for both cases: a wholly-owned group file, and marked regions inside the co-owned `~/.ssh/config` ([D7](../decisions/decision-D7.md)). Host-group composition and `Include` ordering ([T11](../decisions/decision-T11.md)), plus the shadowed-stanza finding that falls out of it.
<!-- SECTION:DESCRIPTION:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: released as v0.3.0 (merge `8b13b0a`). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
