---
id: decision-D1
title: '"Profile" is the single organizing concept for personas'
date: '2026-08-26'
status: 'Accepted — consequence amended by D9'
---
### Context

The original README promised "profiles and profile groups," with groups for `personal` and
`work` and multiple profiles per group. The original implementation instead built "key groups"
with one level of nesting, which it then flattened into dotted names like `work.foobarco`. Two
vocabularies, one underlying idea, and no statement of which was correct.

### Decision

**One concept, hierarchically named.** A profile is a persona. Profile names are paths:
`work.foobarco` is a profile, and `work` is its parent and also a profile. A "profile group" is
simply a profile that has children. Nesting is a naming convention, not a second type.

### Rationale

Collapsing the two satisfies the original requirement — multiple profiles per context
(`work.foobarco`, `work.acme`) — without a second entity. It also matches what the original code
*actually did* once it flattened its configuration, meaning the implementation had already
discovered this answer without anyone writing it down.

### Consequence

The words "key group" and "profile group" leave the project's vocabulary permanently.

The original form of this consequence also retired "host group." **[D9](decision-D9.md) reverses that
specific part**: host groups exist, but as a container concept on a different axis, not as
persona taxonomy. Profiles remain the sole *organizing* concept.
