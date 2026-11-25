---
id: decision-T52
title: '`render.marshalData`''s nil-slice normalization becomes a recursive walk, deep enough to cover every field, present and future'
date: '2026-09-11'
status: 'Accepted'
---
### Context

This entry closes the item [T51](decision-T51.md)'s own `### Consequence` left open — the maintainer's
decision, deferred there by name, on `"profiles": null` and `"hosts": null` both still reaching
`--json`'s wire form despite [T14](decision-T14.md)'s stated pipeline contract (*"`hasp list key --json | jq`
has to keep working release over release"*). This entry does not edit T51 — the logs are
append-only — it names T51's open item and answers it, the same mechanism [T49](decision-T49.md) uses against
[T39](decision-T39.md) and [T37](decision-T37.md) uses against [T29](decision-T29.md)/[T14](decision-T14.md).

`internal/cli/render/json.go`'s `marshalData` has, since its own introduction (pre-dating this
log's earliest entries), normalized a nil Go slice to JSON `"[]"` — but only when that slice *is*
the top-level `data` value `render.JSON` is handed, never when it is a field nested inside a
struct. Confirmed at the binary, as of this writing: `"profiles": null`
(`domain.Key.Profiles`) in `key.list`, `key.show`, and `key.find`; `"hosts": null`
(`app.KeyDetail.Hosts`) in `key.show`. `internal/app.originsForInvestigate`'s own doc comment (the
one T51's Consequence names) had already worked around one instance of this — constructing
`[]domain.Origin{}` by hand at the one call site M3.6.4 introduced — specifically *because*
`marshalData` was not recursive, and said so in as many words: "marshalData is deliberately NOT
made recursive to fix this class of defect generally... the fix belongs here, at the one call
site." That was a correct, scoped decision for M3.6.4's own boundaries (roadmap.md §5.6 exit
criterion 7 pinned `list key`'s default output byte-for-byte against a golden that still carried
the pre-existing `"profiles": null`, so widening the fix would have failed M3.6.4's own gate) — but
it left the actual defect in place at every other call site, present and future, each one relying
on its own author remembering to hand-construct an empty slice rather than writing the idiomatic
Go `return nil`.

### Decision

Replace `marshalData`'s top-level-only special case with a recursive walk,
`normalizeNilSlices(reflect.Value) reflect.Value`, that reaches every nil slice `data` contains at
any depth — through struct fields, slice elements, map values, pointers, and interfaces — and
rewrites each one to a non-nil, zero-length slice of the same type, so it marshals as `"[]"`
instead of `encoding/json`'s default `"null"`. This is now the one mechanism that discharges
[T14](decision-T14.md)'s array-typed-field guarantee everywhere at once, rather than one guarantee in the
renderer (the top-level case) plus a hand-maintained special case at each call site that happens to
need it (`originsForInvestigate`'s pre-T52 construction, and every future field a use case might
add) — the same "fix the class, not the instance" judgment this project already applies elsewhere
(T18's CST-defects-as-data, T30's witness-based race detection).

Four traps make the walk's correctness non-obvious, each guarded explicitly (the full argument
lives in `normalizeNilSlices`'s own doc comment, `internal/cli/render/json.go`):

1. **`json.RawMessage` never enters this walk.** `marshalData` only ever receives the payload
   `render.JSON`'s caller hands it — the value about to become `Envelope.Data` — never the
   `Envelope` itself, verified directly against `JSON`'s own body (`raw, err :=
   marshalData(data)`, with `Envelope.Data: raw` assigned only afterward). One less hazard to
   guard than a naive reading of "this walks arbitrary JSON-bound Go values" would suggest.
2. **`[]byte` is `encoding/json`'s one documented exception to "slices are JSON arrays"** — it
   base64-encodes to a string, and a nil one is `"null"`, correctly, not `"[]"`. The walk checks
   the slice's *element* Kind (`Uint8`), not its exact type, matching `encoding/json`'s own rule,
   and leaves such a slice — nil or not — completely untouched. This is the sharpest edge in the
   function: inverting the check (testing the slice's own Kind instead of its element's) would
   silently corrupt every `[]byte`-shaped field's wire *type*, not merely its value. No such field
   exists in this tree today; the guard is unconditional regardless.
3. **A type implementing `json.Marshaler` owns its output completely and is never rebuilt.** The
   walk checks both a value's own type and a pointer to it against `json.Marshaler` (a
   pointer-receiver method only attaches to the pointer type) and returns any match untouched
   rather than reconstructing it field-by-field — which would, among other things, silently drop
   its unexported state (see point 5). This covers every enum-shaped type in the tree as of this
   writing: `domain.KeyFormat`, `domain.BindingKind`, `domain.ProfilePath`, `domain.Confidence`,
   `domain.SchemeID`, `domain.KeyIdentity`'s `byFingerprint`/`byPath`, `app.Severity`, and
   `app.DiffKind`. `domain.Origin.Because` (`[]domain.ReasonToken`) stays in scope regardless:
   `ReasonToken` itself carries no `MarshalJSON` (it is a plain string on the wire), so it is the
   *slice's element type* owning a marshaler that disqualifies descending into that element, never
   merely the slice containing elements of some other, ordinary type.
4. **A nil map is left as-is, never rewritten to a non-nil empty map.** [T14](decision-T14.md)'s stated
   contract is about arrays (`jq '.data[]'` choking on `null`); nothing in it, or in any entry
   amending it, extends to JSON objects. The one map in this tree, `app.Finding.Detail`
   (`map[string]any`), is tagged `omitempty`, so this case is currently unreachable on the wire —
   recorded as a decision, not an oversight, precisely because a future non-`omitempty` map field
   would otherwise raise the same question again with no citation to answer it. A non-nil map's
   *values* are still walked (`Finding.Detail`'s own documented `{"paths": [...]}` shape, a
   `[]string` inside a `map[string]any` value, is exactly as normalized as any other reachable nil
   slice).

Unexported struct fields are skipped during the walk (`reflect.Value.CanSet` reports `false` for
one, by the `reflect` package's own visibility rule) rather than zero-valued by omission — recorded
explicitly as lossless, not overlooked, because `encoding/json` never marshals an unexported field
either, so the two facts compose to make skipping one free of any wire-visible effect. A nil
pointer and a nil interface both pass through unchanged (only a nil *slice* is ever rewritten; a
pointer or an interface names a single optional value, not a "no results" collection). The walk
never mutates the value it is handed — every slice, map, struct, pointer, and interface it
recurses into is copied fresh (`reflect.New`/`MakeSlice`/`MakeMapWithSize`) before being written
to, so a caller that reuses the same Go value after a `render.JSON` call observes it unchanged
(`TestMarshalData_DoesNotMutateInput`, `internal/cli/render/json_test.go`) — `marshalData`'s
contract with `internal/app` is to render what it is given, not to rewrite it out from under a
caller that might hold onto it.

`internal/app.originsForInvestigate`'s hand-maintained `[]domain.Origin{}` construction is retired
back to idiomatic `return nil` now that the guarantee it was standing in for is general — its own
doc comment is rewritten to cite this entry rather than argue for a special case that no longer
exists (`internal/app/investigate.go`).

### Rationale

P8 scopes hasp to one laptop and tens of keys, never a service processing bulk JSON at volume —
performance is explicitly not a design constraint this entry weighs against correctness, and the
walk's per-call allocation of a fresh copy at every level is a deliberate trade for the
trap-proof, no-mutation guarantee above, not an oversight to optimize later.

One mechanism, applied uniformly by the renderer, is more structurally consistent than the
alternative this project already tried and found wanting: a per-field discipline where each
producer in `internal/app` must remember, unprompted, to construct `[]T{}` instead of the
idiomatic `nil` its own use case would otherwise return. `originsForInvestigate` is the concrete
proof that discipline does not scale even across the *existing* surface — `domain.Key.Profiles`
and `app.KeyDetail.Hosts` sat unfixed in production output at the very moment T51 documented the
one field M3.6.4 *did* remember to fix — and every future field a use case adds would need the same
manual attention, forever, with no test able to catch a forgotten instance short of enumerating
every field by hand. A recursive walk in the one place every `--json` payload already passes
through closes the class of defect once, the same argument [T18](decision-T18.md) and [T30](decision-T30.md) already
settled for their own domains: fix the mechanism that produces every instance, not the instance
that happened to get noticed first.

### Consequence

- `internal/cli/render/json.go`'s `marshalData` and its new helper `normalizeNilSlices` implement
  the recursive walk described above; `internal/cli/render/json_test.go` gains one unit test per
  trap named in the Decision section, plus `TestMarshalData_DoesNotMutateInput` proving the
  no-mutation guarantee.
- `internal/app/investigate.go`'s `originsForInvestigate` returns idiomatic `return nil` on its
  non-RSA branch, and its doc comment cites this entry rather than arguing against recursion.
  `internal/app/investigate_test.go`'s `TestOriginsForInvestigate_JSON_NeverNull` — which asserted
  the pre-T52 workaround directly via a bare `json.Marshal`, bypassing `render.JSON` entirely — is
  replaced by `TestOriginsForInvestigate_NonRSA_ReturnsIdiomaticNil`, pinning the new, idiomatic
  app-layer shape; the wire-level guarantee it used to protect now lives, correctly, one layer up.
- `internal/cli/jsonnullguard_test.go` adds `TestJSONKinds_NeverEmitArrayTypedNull`, a mechanical
  end-to-end guard running every `--json` kind hasp exposes (`key.list`, `key.show`, `key.find`,
  `host.list`, `host.show`, `host.find`, `profile.list`, `profile.show`, `profile.find`,
  `check.report`, and the two `--investigate` variants) against a fixture built so every
  array-typed field each kind can carry comes back genuinely empty, asserting on raw response
  bytes (never round-tripped through `encoding/json`, which conflates `null` and `[]` on decode)
  that no array-typed field is ever rendered as JSON `null`.
- `internal/cli/testdata/golden/list-key.json.golden` is regenerated: `"profiles": null` becomes
  `"profiles": []` at both occurrences. `internal/cli/testdata/golden/list-key.human.golden` is
  byte-identical, unchanged — this fix is JSON-only, exactly as
  `TestDefaultRead_ListKey_Golden`'s own byte-for-byte comparison on both files already proves.
  Regenerating a golden to match a deliberately corrected structure, rather than preserving it
  byte-for-byte against the fix, is the maintainer's own stated stance for this project's
  pre-`v1.0.0` window specifically: correcting structure here is free, and test data — goldens
  included — is expected to track the correction, not pin the defect. That posture is explicitly
  temporary: the byte-frozen-artifact discipline `roadmap.md` §5.6 exit criterion 7 and
  [T31](decision-T31.md)/[T37](decision-T37.md)'s compatibility-surface freeze describe begins **at** the `v1.0.0` tag,
  not before it — this entry lands deliberately ahead of that tag so the fix is free to make now
  and impossible to make later without a major-version bump.
- `tdd.md` §10 states the corrected contract plainly, citing this entry: every array-typed field
  in `--json` output is always an array, never `null`, at every nesting depth, enforced centrally
  in the renderer.
- `roadmap.md` §5.6's exit criterion 7 and its M3.6.5 close-out note are updated to state honestly
  that the golden this criterion is asserted against was deliberately regenerated by this entry,
  and why that does not weaken the criterion's own intent (the default read still never diverges
  from `--investigate`'s absence, proven by the regenerated golden plus
  `TestDefaultRead_ListKey_Investigate_DiffersFromGolden`, unchanged).
