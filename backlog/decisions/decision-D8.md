---
id: decision-D8
title: 'The Git/GPG identity direction stays, as direction'
date: '2026-08-26'
status: 'Accepted'
---
### Context

The original README listed GPG key awareness and Git identity management under "Future Ideas."
The question was whether to keep them in the vision at all, given that the project's history is
a case study in what happens when a second front opens before the first is finished.

### Decision

**Keep it in the vision; keep it out of the near-term work.**

### Rationale

It stays because it is *why* profiles are shaped as they are — a persona that carries a signing
key and a commit identity is the reason "profile" is the organizing concept rather than "key
group." Dropping it would reduce hasp to a key-inventory tool. It waits because SSH alone is
not yet done.

### Consequence

The domain model must not make [J9](decision-D11.md) impossible. No near-term milestone may include it.
Reaching for it early is the specific failure mode this project has already experienced once.
