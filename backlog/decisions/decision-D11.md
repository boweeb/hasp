---
id: decision-D11
title: 'Offboarding and unified identity are ratified journeys'
date: '2026-08-26'
status: 'Accepted'
---
### Context

Two journeys were proposed as extensions beyond the original vision rather than restatements
of it, and were flagged as such for explicit acceptance or rejection.

### Decision

Both are accepted as part of the vision.

- **J8 — Offboard.** *I'm leaving a context. Show me everything scoped to that profile — every
  key, every host that depends on it — so I know what to revoke and what will break.*
- **J9 — Unify identity.** *A profile carries not just SSH keys but the Git identity and
  signing key that go with the persona.* Post-v1, per [D8](decision-D8.md).

### Rationale

J8 is the practical payoff of profiles being first-class, and it is the strongest justification
for modelling bindings as a relation rather than a directive: the valuable question runs
*backwards* along the binding — which hosts use this key — and that is exactly what you need on
your last day somewhere.

J9 is the original "Future Ideas" section promoted from a list of features to the reason the
domain model has the shape it does.

### Consequence

- J8 makes the binding relation load-bearing rather than incidental, and it must work correctly
  for keys shared across profiles ([D2](decision-D2.md)).
- J9 is bound by [D8](decision-D8.md): it shapes the model, but no near-term milestone may include it.
