---
id: DRAFT-0006
title: J9 user guide section, and the v1.1.0 release
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
Document J9 in `docs/user-guide.md` once it exists and ship it as v1.1.0, additive over the v1.0.0 surface ([T31](../decisions/decision-T31.md), [T37](../decisions/decision-T37.md)). Gated: not before M3.6 closes with v1.0.0 cut ([D8](../decisions/decision-D8.md)). A draft idea only; the roadmap deliberately commits no M4 work breakdown. Constraints: [tdd §14](../../docs/tdd.md).
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 `go run mage.go ci` passes
- [ ] #2 Docs updated where affected (`go run mage.go docs` clean)
- [ ] #3 New choices recorded as a hand-authored decision-Dn/decision-Tn in backlog/decisions/
- [ ] #4 Follow-ups filed as tasks or drafts
- [ ] #5 Verification evidence in notes or final summary
<!-- DOD:END -->
