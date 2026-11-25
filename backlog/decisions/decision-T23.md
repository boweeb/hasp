---
id: decision-T23
title: 'DSA is in scope for the read path; fixtures are PEM-only, hand-constructed'
date: '2026-08-28'
status: 'Accepted — extends T1'
---
### Context

`KeyFormat` carried a `FormatDSA` arm while §12's fixture matrix named only ed25519, RSA and
ECDSA — an untested enum value, which is either dead code or an untested real path and should not
ship as neither.

### Decision

**DSA is in scope for reading, and never for generating.** `x/crypto/ssh`'s `ParseRawPrivateKey`
handles a `DSA PRIVATE KEY` PEM block, so hasp can report such a key's facts; `new key` will
never produce one.

DSA fixtures are **PEM-only and hand-constructed**, because current OpenSSH will not generate them
— DSA support has been disabled and then removed from `ssh-keygen`. The fixtures are built once,
from Go, and committed.

### Rationale

[design.md §2](../../docs/design.md#2-the-problem) describes the actual artifact this tool exists for: a
decade of accumulated keys, *"a mix of RSA and Ed25519, PEM and OpenSSH formats"*, seven of
thirteen in PEM. A tool whose premise is telling the truth about an old `~/.ssh` cannot decline to
read the oldest thing it might find. Reporting a DSA key accurately costs one enum arm and four
fixtures.

Generating one is a different question and the answer is no — hasp should not mint keys with an
algorithm the ecosystem has removed.

### Consequence

- The fixture builder is Go, not a shell script calling `ssh-keygen`, for DSA specifically. This
  is worth noting because every other fixture *can* be produced by `ssh-keygen`, and a future
  contributor regenerating the corpus from a shell script will silently drop the DSA cases.
- A DSA key is reportable but unmanageable in one respect: it can be adopted, aliased and moved
  like any other key, since none of those operations touch its bytes ([D14](decision-D14.md)).
