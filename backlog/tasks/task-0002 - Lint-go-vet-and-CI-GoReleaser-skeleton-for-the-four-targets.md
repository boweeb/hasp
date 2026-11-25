---
id: TASK-0002
title: Lint, go vet, and CI; GoReleaser skeleton for the four targets
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - ci
  - release
milestone: m-1
dependencies: []
references:
  - docs/roadmap.md#2-m0--bootstrap
type: chore
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Lint, `go vet`, and CI running both plus `go test ./...`. A GoReleaser skeleton that builds the four targets ([T9](../decisions/decision-T9.md)), releasing nothing yet, proving `CGO_ENABLED=0` holds from the first commit.
<!-- SECTION:DESCRIPTION:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: `04d6b8d` (Implement M0 — Bootstrap). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
