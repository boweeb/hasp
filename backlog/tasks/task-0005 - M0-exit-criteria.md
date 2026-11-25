---
id: TASK-0005
title: M0 exit criteria
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
updated_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - ci
milestone: m-1
dependencies: []
references:
  - docs/roadmap.md#2-m0--bootstrap
type: chore
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The exit criteria `docs/roadmap.md` §2 listed for M0, kept here as the completion record.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 `go build ./...` and `go test ./...` pass in CI on a clean checkout.
- [x] #2 The layering guard fails when someone adds a third-party import to `internal/domain`, verified by trying it once, deliberately.
- [x] #3 All key fixtures exist, are committed, and each parses to the algorithm, format, and encryption state its filename claims.
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: `04d6b8d` (Implement M0 — Bootstrap). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
