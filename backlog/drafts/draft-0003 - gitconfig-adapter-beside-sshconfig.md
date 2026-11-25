---
id: DRAFT-0003
title: gitconfig adapter beside sshconfig
status: Draft
assignee: []
created_date: '2026-11-25T15:44:00Z'
labels:
  - m4
milestone: m-7
dependencies: []
references:
  - docs/roadmap.md#6-m4--identity
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
A dedicated adapter for reading (and later writing) `gitconfig` conditional includes, keeping `internal/adapter/sshconfig` from being assumed the only config format ([T2](../decisions/decision-T2.md)). Gated: not before M3.6 closes with v1.0.0 cut ([D8](../decisions/decision-D8.md)). A draft idea only; the roadmap deliberately commits no M4 work breakdown. Constraints: [tdd §14](../../docs/tdd.md).
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 `go run mage.go ci` passes
- [ ] #2 Docs updated where affected (`go run mage.go docs` clean)
- [ ] #3 New choices recorded as a hand-authored decision-Dn/decision-Tn in backlog/decisions/
- [ ] #4 Follow-ups filed as tasks or drafts
- [ ] #5 Verification evidence in notes or final summary
<!-- DOD:END -->
