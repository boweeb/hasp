---
id: decision-D4
title: 'hasp forgets records; it never destroys key material'
date: '2026-08-26'
status: 'Accepted — mechanism amended by D12'
---
### Context

The abandoned implementation carried a full create/read/update/**delete** shape in its
scaffolding, but the command surface had no delete verb. The question of whether hasp destroys
things was never resolved — it was simply left in two states at once.

### Decision

**`forget` removes a record and never touches disk.** Adopted in the strong form:
**hasp has no operation that can destroy an irreplaceable secret.** This is canon.

### Rationale

`rm` already exists and is better understood. Reconciliation will notice the absence and update
the records on its own. A tool that has never destroyed anyone's private key has a perfect
safety record it can simply keep, and that record is worth more than the convenience of a
delete verb.

### Consequence

Dropping a record is cheap and fully reversible — reconciliation restores it if the material is
still there. Removing key material is the user's own act, performed with their own tools. This
is a permanent, load-bearing safety property, and it constrains every future operation: any
proposal that would give hasp the power to destroy key material contradicts ratified canon.

### Amended by D12

[D12](decision-D12.md) removed persisted state, which retires the `forget` **verb** — there are no records
to drop from. The **canon is unchanged and if anything is strengthened**: with nothing
persisted, forgetting is simply what happens when the material is gone and hasp next looks. The
safety property no longer depends on hasp declining to do something; it holds because hasp has
no mechanism through which it could.
