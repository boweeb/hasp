---
id: decision-T47
title: 'The fingerprint-scheme no-network guard is a direct-import assertion, not a `net.Dialer` `Control` hook'
date: '2026-09-09'
status: 'Accepted — amends T35'
---
### Context

`tdd.md` §12 states the no-network guard for [T35](decision-T35.md)'s fingerprint scheme registry in these
terms: *"A guard test runs every registered scheme's `Compute` inside a `net.Dialer` whose
`Control` callback fails any socket creation, and asserts no scheme ever trips it."* Implementing
`internal/adapter/fpscheme` for [M3.6.1](../../docs/roadmap.md#56-m36--investigation) this session, that
mechanism turned out not to work at all: a `net.Dialer` constructed inside a test has no effect on
code under test that never holds a reference to it. Every `Scheme.Compute` function in this
package operates purely on bytes already in memory (`crypto/x509`, `crypto/sha1`, `crypto/md5`,
and `golang.org/x/crypto/ssh`'s wire-format marshaling) — none of them accepts a `net.Dialer`, a
`context.Context`, or any other hook a test could use to intercept a dial that is never attempted.
A `Control`-callback guard would pass unconditionally regardless of whether the code under test
ever opened a socket, which makes it worthless as a guard: it cannot fail.

A second, unrelated problem surfaced while designing the replacement: this package's own required
dependency, `golang.org/x/crypto/ssh` (needed for `ssh.FingerprintSHA256` and SSH wire-format
marshaling — [T35](decision-T35.md)'s own table), itself directly imports `"net"` for its unrelated
`Dial`/`Listen`/`Conn` machinery elsewhere in that package. Verified this session:
`go list -deps golang.org/x/crypto/ssh` includes `net`, `net/netip`, and `net/url`. A guard built
the way [T13](decision-T13.md)'s `internal/domain` layering guard and [T32](decision-T32.md)'s `cmd/hasp` layering guard
both work — `go list -deps .`, the full transitive closure — would therefore fail permanently the
moment `fpscheme` imports `ssh` at all, for a reason completely unconnected to whether this
package's own code ever dials.

**What this guard actually stands in for.** [T35](decision-T35.md)'s "never a network call" boundary is not
incidental to the scheme registry — it is what keeps `design.md` §3.2's *"Not a key distribution
mechanism"* boundary intact even though matching a console fingerprint is, on its face, about an
external system. That boundary is itself [P3](../../docs/design.md#4-principles)'s scope half: hasp manages
arrangement, not secrets, and a scheme that phoned out to verify a fingerprint against a console
would be hasp acting as exactly the distribution mechanism §3.2 forbids. This entry exists only to
make that boundary mechanically checkable rather than a claim resting on review discipline, so it
cites [P3](../../docs/design.md#4-principles) directly, not only the T35/T13/T32 entries that shape its
mechanism.

### Decision

Replace the `net.Dialer`/`Control` mechanism with a `go list`-based guard, the same mechanical
shape [T13](decision-T13.md)'s and [T32](decision-T32.md)'s layering guards already use, but scoped to
**`internal/adapter/fpscheme`'s own direct imports** rather than its full transitive dependency
graph: `go list -f '{{ join .Imports "\n" }}' .` (non-transitive — direct imports only,
production code only, test files excluded), asserted to contain none of `net`, `net/http`, or
`os/exec`.

The direct-vs-transitive distinction is the load-bearing part of this decision, not an
implementation detail: a transitive check is factually impossible to pass while this package
depends on `golang.org/x/crypto/ssh` (Context above), so the only assertion that can ever be both
true and useful is "this package's own source code never imports a networking or subprocess
package" — which is exactly [T35](decision-T35.md)'s actual claim ("no scheme ever opens a socket").

### Rationale

An alternative considered: keep the `net.Dialer`/`Control` mechanism as a best-effort documentation
device even though it cannot fail. Rejected — a test that cannot fail is worse than no test, since
it reads as coverage in a diff and in `go test -v` output without providing any. [T13](decision-T13.md) and
[T32](decision-T32.md) already establish this codebase's preferred alternative to "review discipline" for a
layering claim: a `go list`-based mechanical assertion. Reusing that exact shape here, adjusted
only for the transitive/direct distinction the `x/crypto/ssh` dependency forces, keeps the guard
consistent with two precedents already in the tree rather than inventing a third pattern.

### Consequence

- `internal/adapter/fpscheme/layering_test.go`'s `TestNoNetworkGuard` implements this mechanism,
  with a doc comment explaining both why the `tdd.md` §12 mechanism cannot work and why the check
  is direct-import rather than transitive.
- `tdd.md` §12's no-network guard bullet is updated to describe this mechanism, citing this entry,
  per this log's own rule that `tdd.md` reflects every accepted entry as current truth.
- The assertion's scope is narrower than "nothing in this package's entire dependency tree can
  dial" — it is "this package's own code never asks to." That is deliberate: the broader claim was
  never true of `golang.org/x/crypto/ssh` even before this package existed, and pretending
  otherwise would make the guard fail for a fact this project has no intention of changing (T35's
  registry needs `x/crypto/ssh`, full stop).
