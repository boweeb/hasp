---
id: decision-D2
title: 'A key may belong to more than one profile'
date: '2026-08-26'
status: 'Accepted'
---
### Context

Whether key↔profile membership is one-to-many or many-to-many. One-to-many is tidier and
produces simpler output.

### Decision

**Many-to-many.** A key belongs to zero or more profiles.

### Rationale

A key genuinely shared between `personal` and `work.foobarco` is a real situation, not a modelling
error. Forcing a single choice would make the tool assert something false about the machine,
which violates the premise that hasp tells the truth about what is there.

### Consequence

"Which profile is this key in" is a list, not a value. [J8](decision-D11.md) (offboard) must handle the
case of a key scoped to a departing profile *and* to something being kept — which is precisely
the case where getting it wrong is most expensive.
