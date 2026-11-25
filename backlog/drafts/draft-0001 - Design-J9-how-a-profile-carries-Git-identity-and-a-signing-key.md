---
id: DRAFT-0001
title: 'Design J9: how a profile carries Git identity and a signing key'
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
Settle, as a numbered decision before any code, what a profile's Git identity and signing-key reference are, per P9 (natural structures first: `gitconfig` conditional includes and repository layout, not a hasp-owned registry). Gated: not before M3.6 closes with v1.0.0 cut ([D8](../decisions/decision-D8.md)). A draft idea only; the roadmap deliberately commits no M4 work breakdown. Constraints: [tdd §14](../../docs/tdd.md).
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 `go run mage.go ci` passes
- [ ] #2 Docs updated where affected (`go run mage.go docs` clean)
- [ ] #3 New choices recorded as a hand-authored decision-Dn/decision-Tn in backlog/decisions/
- [ ] #4 Follow-ups filed as tasks or drafts
- [ ] #5 Verification evidence in notes or final summary
<!-- DOD:END -->
