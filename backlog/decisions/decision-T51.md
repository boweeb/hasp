---
id: decision-T51
title: 'Four M3.6.4 decisions: additive `--investigate` JSON, the no-clue origin rule, literal-value `because` tokens, and the agent-before-derive ordering fix'
date: '2026-09-10'
status: 'Accepted'
---
### Context

Chunk [M3.6.4](../../docs/roadmap.md#56-m36--investigation) (`internal/app/investigate.go`,
`internal/cli/key.go`) landed `--investigate` on `show key` and `list key` — `tdd.md` §18's and
[T35](decision-T35.md)–[T39](decision-T39.md)'s abstract design, wired into actual code and an actual wire format for
the first time. Wiring it forced four questions neither `tdd.md` §18 nor any prior `T`-log entry
had settled at the precision an implementation needs: what JSON shape the new facts take, what
`--investigate` may say about a key's provenance when the user has supplied no external clue at
all, what the evidence tokens behind that guess actually contain, and how the chunk's own
"agent tried before a passphrase is asked" ordering obligation (§18's `ssh-agent` subsection,
`roadmap.md` §5.6's M3.6.4 chunk row) is made a property of the code rather than a claim about it.
This entry is the permanent record of how each was settled, so the reasoning has one recorded home
rather than living only in a commit message (`9e51dc8`) and Go doc comments that could drift from
it unnoticed.

### Decision

**1. `--investigate`'s output is additive under the existing `key.list`/`key.show` kinds — no new
`kind` strings.** `app.InvestigatedKey` embeds `domain.Key` rather than duplicating its fields.
`domain.Key` carries no `MarshalJSON` of its own (only its field types do —
`KeyFormat`, `KeyIdentity`'s `byFingerprint`/`byPath`), so embedding marshals every existing field
flat, at its existing JSON path, unchanged: an existing `key.list`/`key.show` consumer sees nothing
new unless it also reads one of `InvestigatedKey`'s additive fields (`schemes`, `origins`,
`agentComment`, `commentSource`). The rejected alternative — `key.list.investigate` and
`key.show.investigate` as two new `kind` strings — was considered and set aside: it would add two
permanent entries to `tdd.md` §16 row 3's frozen `kind` surface, force every consumer of either
verb to branch on two kinds to answer what is still fundamentally "list/show a key," and buy
nothing a flag check on the existing kind does not already give a `--json` consumer for free. This
was the maintainer's own decision, made explicitly during M3.6.4, not a default arrived at by
omission.

**2. The no-clue origin rule: an RSA key yields both `aws-ec2-created` and `aws-ec2-imported` at
`possible`; a non-RSA key yields an empty array.** `internal/app.originsForInvestigate`'s own doc
comment states the reasoning this entry ratifies: `--investigate` has no external clue to match at
all (unlike `find`, T36), so it can never report `ConfidenceConfirmed` — the strongest claim
available is `possible`. An RSA key is genuinely consistent with *both* AWS RSA schemes — a
created key is PEM-encoded PKCS#8/PKCS1 and an imported key can be too ([T35](decision-T35.md)'s table) — so
reporting only one origin would falsely imply the other is inconsistent with the evidence, which is
not true; both are reported, each carrying the same `because` evidence for a consumer to weigh. A
non-RSA key has neither AWS RSA scheme apply to it at all
(`fpscheme.IsRSAAlgorithm`), so there is no provenance signal to grade at all, and the array is
empty — inventing one would be exactly the guess `design.md` §5.1 / [D12](decision-D12.md)
forbid. `tdd.md` §10's / [T37](decision-T37.md)'s worked example —
`{"id": "aws-ec2-created", "confidence": "possible", "because": [...]}` — shows the shape of
exactly one element of the two-element array this rule produces for an RSA key; that shape stays
exactly right under this decision.

**3. The `because` tokens carry the key's literal field values — `algorithm=ssh-rsa`,
`format=pkcs1` — not the simplified `algorithm=rsa`, `format=pem` of `tdd.md` §10's / T37's worked
example.** `domain.KeyFormat.String()` never returns `"pem"` at all — its closed set is
`openssh`/`pkcs1`/`pkcs8`/`sec1`/`dsa` (`internal/domain/key.go`) — and `domain.Key.Algorithm`
holds the SSH wire algorithm name hasp actually derives (`ssh-rsa`), never a bare `rsa`. The
worked example in `tdd.md` §10 and T37 was always illustrative of *shape* — an array of
machine-readable `key=value` tokens plus a fixed reason — not a promise of its literal string
content, and emitting the type system's real values is the honest reading of that shape: a
machine-readable token naming a value `domain.KeyFormat` cannot actually produce (`"pem"`) would be
worse than useless to a consumer that tries to match on it. **This is a precision on T37's example,
not a contradiction of it** — the example's citation in `tdd.md` §10 gains a note pointing here so
a reader does not mistake the literal strings shown there for a compatibility promise.

**4. The "agent before derive" ordering obligation is pinned by an observable consequence, not by
decomposition alone.** `tdd.md` §18's `ssh-agent` subsection and `roadmap.md` §5.6's M3.6.4 chunk
row both require the agent lookup to merge into a candidate key's material before
`PassphraseGate.Derive` runs for that key. The chunk's first implementation split this into
numbered steps (`internal/app.knownMaterialWithAgentFact` then `investigateKey`) that already
satisfied the ordering structurally — step 4 (`gate.Derive`) cannot compile without step 3's merged
`known` already in hand — but the reported `AgentComment` was read from
`knownMaterialWithAgentFact`'s own local return value, not from what `gate.Derive` actually
produced, so the output never changed even when a hypothetical call site quietly re-derived a
second, pre-merge `Material` and handed *that* to `Derive` instead: `TestInvestigateKey_AgentTriedBeforeDerive`'s
first draft passed against an inverted ordering, which the verification gate flagged as a guard
that proved nothing. The fix reads `AgentComment` from `derived.Material.Comment` instead — the
material that actually flowed through `gate.Derive` — guarded on `commentSource ==
domain.FactSourceAgent` so a `.pub` sidecar comment already present in `known.Comment` is never
mislabelled agent-sourced, per [T38](decision-T38.md)'s labelling rule. That source switch is what makes an
inverted call site produce a visibly wrong `AgentComment` (empty, or stale) rather than a silently
still-correct one.

Making that switch exposed a real bug it needed to be correct: `PassphraseGate.Derive`'s
successful-unlock path returned `openFn`'s freshly-opened `Material` verbatim, which erased a
caller-merged agent comment exactly when a correct passphrase was also supplied — the one branch
where the caller had done the most work to get there. This is load-bearing, not cosmetic:
[T49](decision-T49.md) established that `golang.org/x/crypto/ssh`'s passphrase-decrypt API discards an
OpenSSH-format key's on-wire comment even after a correct passphrase, so `ssh-agent` is the *only*
source for that fact once a key is encrypted with no `.pub` sidecar — a `Derive` that silently drops
a caller-merged agent comment on its one successful path would make the ordering obligation
worthless in exactly the case it exists to serve. `Derive` now carries `known.Comment` forward onto
the freshly-opened material whenever that material has no comment of its own (`mat.Comment == ""`),
and never overwrites a real, non-empty sidecar comment already present on `mat`.

### Rationale

All four decisions share one thread: an abstract correctness property (additive JSON, an honest
no-clue confidence ceiling, evidence tokens a consumer can actually rely on, an ordering obligation
that holds) is only as real as the mechanism that makes it observable. Decision 1 keeps `tdd.md`
§16's frozen `kind` surface frozen by construction (Go's own `MarshalJSON` resolution), not by
convention a future PR could violate unnoticed. Decision 2 keeps P10's confidence discipline
honest — `possible`, not `confirmed`, is the only claim the evidence supports with no external
clue — while still reporting real, weighable evidence rather than an empty gesture. Decision 3
chooses truthful machine-readable content over a documentation example's literal spelling, which is
the same judgment [D12](decision-D12.md) and `design.md` §5.1 already make against any kind of
invented fact. Decision 4 is this entry's most direct instance of the thread: a comment asserting
an ordering is not a test of that ordering, and [P3](../../docs/design.md#4-principles)/[D19](decision-D19.md)'s
consent-gated access rule is only as trustworthy as the guarantee that the fact a caller already
holds (an agent-sourced comment) survives the very call (`Derive`) that rule licenses.

### Consequence

- `internal/app.InvestigatedKey`, `internal/app.originsForInvestigate`, and
  `internal/app.investigateKey`/`knownMaterialWithAgentFact` (`internal/app/investigate.go`)
  implement decisions 1, 2, and 4 respectively, exactly as landed in `9e51dc8`.
- `internal/app.PassphraseGate.Derive` (`internal/app/passphrasegate.go`) carries a caller-merged
  `Material.Comment` forward on its successful-unlock path — the decision-4 bug fix — and its own
  doc comment states the contract in full, including the branch that used to violate it.
- `tdd.md` §9, §10, and §18 are updated to state these four shapes as current truth, each citing
  this entry.
- **A named, deliberately-deferred item.** `render.marshalData`
  (`internal/cli/render/json.go`) normalizes a nil slice to `[]` only at the top-level `data` value
  passed to `render.JSON`, never at a nested struct field. `origins` reaches `--json` as `[]` for a
  non-RSA key only because `originsForInvestigate` returns an empty slice at the source (decision 2
  above) — but `"profiles": null` (`domain.Key.Profiles`, visible in the committed pre-M3.6 golden
  `internal/cli/testdata/golden/list-key.json.golden`) and `"hosts": null`
  (`app.KeyDetail.Hosts`, on `show key --json`) both still reach the wire as JSON `null`, the same
  [T14](decision-T14.md) pipeline-contract wart `originsForInvestigate`'s own doc comment names and does not
  fix generally. Neither was touched in M3.6.4, deliberately: `roadmap.md` §5.6 exit criterion 7
  pins `list key`'s default output byte-for-byte against that golden, so changing `profiles` would
  fail the milestone's own gate, and fixing only `hosts` would leave the two inconsistent with each
  other for no principled reason. **This is a decision the maintainer must make before `v1.0.0` is
  tagged**, because the tag freezes it (`tdd.md` §16).
