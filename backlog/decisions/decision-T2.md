---
id: decision-T2
title: 'A hand-rolled, lossless `ssh_config` CST, in place of any existing library'
date: '2026-08-28'
status: 'Accepted — amended by T17, T18, T24, T25'
---
### Context

[D7](decision-D7.md) requires two things of the same subsystem simultaneously: outside a
hasp-owned marked region, the human's bytes survive **byte for byte** — comments, ordering,
whitespace, unrecognized directives, all of it (this *is* P2, narrowed but not softened); inside
a marked region, hasp is authoritative and may rewrite freely. That is a fidelity contract, not
a best-effort parser feature.

### Decision

hasp does not depend on an existing Go `ssh_config` library. It owns a small, line-oriented
Concrete Syntax Tree in `internal/adapter/sshconfig`, whose defining, fuzz-tested contract is:

```text
Render(Parse(b)) == b
```

for arbitrary input bytes `b`, outside marked regions. Inside a marked region the contract is
weaker by design (D7 elaboration 1: hasp may reformat what it owns) but still idempotent:
`Render(Parse(Render(r))) == Render(r)` for any region body `r` hasp produced.

Surveyed and rejected, this session:

| Library | Verdict |
| --- | --- |
| `kevinburke/ssh_config` (the de-facto standard; used by `go-git`) | Its own README states `Match` is **unsupported**, and promises only that it *"attempts to preserve comments."* "Attempts" is not a contract P2 can be built on. |
| `k0sproject/rig/v2/sshconfig` | Reader with partial `Match` support; not a write-fidelity tool. |
| `patrikkj/sshconf`, `petems/go-sshconfig`, `mikkeloscar/sshconfig`, `soulteary/ssh-config` | Readers, or patchers with no stated fidelity contract at all. |

None offer round-trip fidelity as a *guarantee*. Adopting one and hoping is exactly the kind of
promise this project's own history (the round-tripping `ssh_config` parser investigation
`docs/project-assessment-2026-08.md` §6 names directly as unfinished, build-vs-buy business)
says not to make casually.

### Rationale

D7 elaboration 3 is what keeps this small rather than a full-format modeling project: hasp
models the **syntax** of the whole file completely (every byte parses into *some* node — a
recognized directive, an unrecognized line, or region markers) but the **semantics** only for
directives it manages. An unrecognized directive is carried as an opaque node and passed through
verbatim — never an error, per P6 ("nothing hasp does not understand is dropped"). The
"natively managed" directive set is the 21-directive list salvaged from the predecessor's
`README.rst` (§6 below closes this as scope, matching [`tdd.md` §6](../../docs/tdd.md#6-configuration-parsing--a-lossless-cst-for-a-file-hasp-does-not-fully-own)):
`HostName`, `IdentityFile`, `User`, `Port`, `ProxyCommand`, `ControlMaster`, `ControlPath`,
`ControlPersist`, `AddKeysToAgent`, `Ciphers`, `ForwardAgent`, `HashKnownHosts`,
`HostKeyAlgorithms`, `IdentitiesOnly`, `KexAlgorithms`, `LogLevel`, `MACs`,
`PasswordAuthentication`, `PubkeyAuthentication`, `StrictHostKeyChecking`,
`UserKnownHostsFile` — plus `Include`, which hasp itself writes ([T11](decision-T11.md)) and must
therefore understand semantically inside its own regions.

P8 also argues against a general-purpose dependency here: this is a personal tool, and the
libraries above carry surface area (fleet-scale config composition, `Match` blocks with
arbitrary criteria) hasp does not need and would rather not inherit the maintenance burden of.

### Consequence

- This is now hasp's largest first-party subsystem, and its correctness gate is a fuzz test
  (`FuzzSSHConfigRoundTrip`, [`tdd.md` §12](../../docs/tdd.md#12-testing-strategy--a-function-of-a-directory)),
  not a hand-written unit-test suite alone.
- Teaching hasp a new directive later — widening the 21-directive native set — is additive: the
  parser already carries it as an opaque `Directive` node today, so nothing needs to migrate on
  existing files. This is the direct payoff of D7 elaboration 3 the design doc predicted.
- Every future change to this package must run the fuzz corpus before merging; a regression here
  is a P2 violation with no smaller blast radius than "silently mangled the user's config."

**Four later entries harden this contract rather than change it.** The node sketch stated here
was necessary but not sufficient: [T17](decision-T17.md) fixes `Directive` so it actually satisfies
`Render(Parse(b)) == b` for every valid separator/quoting form `ssh_config(5)` allows, which the
original sketch's parsed-`Args`-only shape could not; [T18](decision-T18.md) gives "a marker or region is
malformed" — asserted elsewhere in this project but never defined here — concrete, detectable
shapes; [T24](decision-T24.md) closes a gap the contract implied but the node sketch didn't handle, CRLF and
other line-terminator variation; [T25](decision-T25.md) gives `#:hasp` metadata lines (§7) their own node
rather than leaving them to be re-scanned out of an opaque `Line`.
