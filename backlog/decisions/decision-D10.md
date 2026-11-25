---
id: decision-D10
title: 'Nouns have two classes: first-class and second-class'
date: '2026-08-26'
status: 'Accepted · Refines: D3'
---
### Context

[D3](decision-D3.md) settled that the nouns are key, host, and profile. But the domain plainly contains
more nouns than three — alias, binding, host group — and it was unclear whether each needed a
place in the verb grid, which would multiply the command surface for concepts that are not
independently meaningful.

### Decision

Nouns come in two classes.

- **First-class nouns** — **key**, **host**, **profile**. These are the entities that exist in
  their own right. The verb + noun grammar applies to these and only these.
- **Second-class nouns** — alias, binding, host group, and others yet to be identified. These
  belong *to* a first-class noun and are addressed as modifiers of an operation on their owner,
  not as operations in their own right.

Illustratively: `hasp edit key "foo-key" --add-alias="bar-key"`, rather than a top-level
`alias` noun with its own verbs.

### Rationale

A second-class noun has no independent existence — an alias without a key is meaningless, a
binding without a host and a key is nothing. Promoting such concepts to the verb grid would
grow the command surface combinatorially while offering operations that are either nonsensical
or duplicative.

Keeping the first-class set at three preserves the property that made the original verb×noun
grid the best idea in the abandoned implementation: learning one noun teaches you the others,
and adding a noun is mechanical rather than inventive.

### Consequence

- The verb grid does not grow as the domain gets richer. (It was 3 × 8 when written; [D12](decision-D12.md)
  retired two verbs and [D14](decision-D14.md) added two, so it is 3 × 8 again. The *count* moves with the
  verb set; the point of this decision is that it does not move with the number of concepts.)
- Every new concept must be classified on arrival. "Is this first-class?" becomes a standing
  design question, and the bar is independent existence.
- Second-class nouns surface as named arguments. Their exact spelling is an interface concern
  and remains deferred; the *classification* is design and is settled here.
