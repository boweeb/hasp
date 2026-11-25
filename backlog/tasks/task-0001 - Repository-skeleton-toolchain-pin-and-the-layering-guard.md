---
id: TASK-0001
title: Repository skeleton, toolchain pin, and the layering guard
status: Done
assignee: []
created_date: '2026-11-25T15:44:00Z'
labels:
  - backfill
  - ci
milestone: m-1
dependencies: []
references:
  - docs/roadmap.md#2-m0--bootstrap
type: chore
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
`go.mod` and `mise.toml` pinning the toolchain ([tdd §2](../../docs/tdd.md)). The four-layer package skeleton: `cmd/hasp`, `internal/domain`, `internal/app`, `internal/adapter/*`, `internal/cli` ([tdd §3](../../docs/tdd.md)). The layering guard, wired before there was anything to violate it: `go list -deps` asserting `internal/domain` imports nothing outside the standard library ([T13](../decisions/decision-T13.md)).
<!-- SECTION:DESCRIPTION:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Landed: `04d6b8d` (Implement M0 — Bootstrap). Recorded retroactively from git history; the roadmap section named in the references carries the scope narrative.
<!-- SECTION:FINAL_SUMMARY:END -->
