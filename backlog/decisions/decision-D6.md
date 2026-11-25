---
id: decision-D6
title: 'The write model'
date: '2026-08-26'
status: 'Accepted (2026-08-26) · Collapsed by: D12'
---
### Context

Every write being previewable and reversible is desirable, but applying that ceremony
uniformly would make cheap, safe operations tedious.

### Proposal (not yet ratified)

**Two tiers.** Changes to hasp's own records are immediate — they are reversible by
re-reconciling, since the machine is the source of truth. Changes to key files or configuration
go through preview → confirm → back up → write.

### Open questions

- Is the tier a property of the *operation* or of what it touches at runtime?
- Does preview need to be the default for the risky tier, or an available mode?
- Where do operations that touch both tiers at once land?

### Consequence if accepted

"What would this do" must be answerable *before* a change is applied, which shapes how change
is represented from the very beginning. This cannot be added later without rework.

### Reshaped by D12

[D12](decision-D12.md) removes persisted state entirely, which **collapses the two tiers into one**. If
hasp keeps no records, there is no such thing as a "record-only change" — every write touches
the world, and every write therefore gets the full preview → confirm → back up → write cycle.

The first two open questions above are answered by that collapse; the third dissolves, since no
operation can span tiers that no longer exist. What survives is narrower and should be settled
on its own terms: **is preview the default for every write, or an available mode?**

### Decision

**Every write.** Preview is the default, not a mode the user has to remember to ask for.

*Rationale:* the alternative puts the burden of caution on the person with the least context at
the moment they act. A default that is safe only when you remember to opt into it is not a safe
default. At this scale (P8) the extra step costs a keystroke and buys the guarantee P5 was
written to make.

*Consequence:* there is exactly one write model, applied uniformly — **preview → confirm →
back up → write** — with no operation-by-operation judgement about which changes are risky
enough to warrant it. Non-interactive use must therefore have an explicit way to consent in
advance; its spelling is an interface concern and remains deferred.
