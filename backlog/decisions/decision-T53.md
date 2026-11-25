---
id: decision-T53
title: 'T52''s end-to-end guard overstated its own coverage; corrected, not merely noted'
date: '2026-09-11'
status: 'Accepted'
---
### Context

A v1.0.0-gate code review (revision cycle 1) of [T52](decision-T52.md) found that
`internal/cli/jsonnullguard_test.go`'s `TestJSONKinds_NeverEmitArrayTypedNull` — [T52](decision-T52.md)'s own
Consequence section describes it as "a mechanical end-to-end guard running every `--json` kind hasp
exposes ... against a fixture built so every array-typed field each kind can carry comes back
genuinely empty" — did not, in fact, do that for three of the twelve kinds it enumerates.
`profile.list` and `profile.find` carry `app.ProfileSummary` (`internal/app/profile.go`) and
`domain.Profile` (`internal/domain/profile.go`) respectively, and neither type has a nested
array-typed field at all: the only array either kind's payload can ever carry is the top-level
`data` envelope array, already covered by `marshalData`'s pre-[T52](decision-T52.md) top-level-only special
case. Those two subtests cannot fail for the defect class [T52](decision-T52.md) exists to catch, no matter
what `normalizeNilSlices` does — they were true before [T52](decision-T52.md)'s recursive walk existed and
remain true regardless of it.

`check.report` is the more consequential gap. `app.Finding.Detail` (`map[string]any`) is the one
genuine map-nested-slice case in this tree — [T52](decision-T52.md)'s own Decision section cites it by name as
proof the map-values-are-walked guarantee matters (`internal/app/check_key.go` populates
`Detail["paths"]` with a `[]string` on a duplicate-key finding) — but
`jsonNullGuardArrayFields` never listed `"paths"`, and the guard's shared fixture (one key, one
bare host stanza, one empty profile) never produced a duplicate-key finding in the first place, so
nothing about that map-nested case was ever exercised end-to-end. `check.report`'s subtest was a
bare `assertNoArrayTypedNull` call with no fixture-specific positive assertion, unlike every other
subtest in the same test function.

The identical overstated claim — that the guard proves the guarantee "for every kind" — also
appears in two other places written at the same time as [T52](decision-T52.md): this log's own [T52](decision-T52.md)
Consequence section (quoted above) and `internal/app/investigate.go`'s `originsForInvestigate` doc
comment, which cites the guard as proving the wire guarantee "through the full --json pipeline for
every kind including this one."

### Decision

**1. `check.report`'s subtest now genuinely exercises the map-nested-slice path.**
`TestJSONKinds_NeverEmitArrayTypedNull`'s shared fixture gains a second, independent real file
(`custom_ed25519_dup`) carrying the exact same key bytes as the existing `custom_ed25519` fixture
key. `internal/app.deriveKeys` (`internal/app/pipeline.go`) groups the two by fingerprint into one
`domain.Key` with two non-alias `Locations` — `internal/app/check_key.go`'s own
`detectDuplicateKeyConfirmed` trigger ("two or more non-alias Locations") — so `check --json` now
raises a real `duplicate-key-confirmed` finding whose `Detail["paths"]` carries both files' paths.
`jsonNullGuardArrayFields` gains `"paths"`, and `check.report`'s subtest gains a fixture-specific
positive assertion (`"paths": [` present, and the second file's name present inside it) matching
the pattern every other subtest already follows.

**2. The three overstated-coverage sites are corrected, not merely footnoted.** `jsonnullguard_test.go`'s
own doc comment now states plainly that `profile.list` and `profile.find` are **structural no-ops**
for this regression class — their payload types carry no nested array field, so neither subtest can
fail for a [T52](decision-T52.md)-class defect — and that they are enumerated anyway so a future field added to
either payload type is caught the moment it starts carrying a nested array, not silently. This
log's [T52](decision-T52.md) entry is append-only and is not edited; its Index-row Status gains a `refined by
[T53](decision-T53.md)` relation clause instead, the same mechanism [T26](decision-T26.md)'s row already uses. `internal/app/investigate.go`'s
doc comment is reworded to the same accurate claim: the guard proves the wire guarantee for
`--investigate`'s own `origins` field specifically, and for every other kind's own genuinely
nested array fields, while naming `profile.list`/`profile.find` as this entry's recorded, deliberate
exception rather than leaving the prior, broader claim standing uncorrected.

**3. The strengthened guard's actual bite is recorded honestly, not assumed.** Verification for this
revision required neutering `normalizeNilSlices` to `return v` and confirming `check.report`'s
subtest newly fails. It does not. Both `Detail["paths"]`-producing detectors in
`internal/app/check_key.go` — `detectDuplicateKeyConfirmed` and `detectDuplicateKeyUnconfirmed` —
only ever construct their `paths`/`allPaths` slice, and only ever raise their finding at all, after
first establishing `len(paths) >= 2` (each detector's own firing precondition, guarding against
reporting a "duplicate" of one). That precondition makes `Detail["paths"]` structurally
non-nil at the moment either finding carries it, in the codebase as it stands today — so a fully
neutered `normalizeNilSlices` still renders `"paths": [...]` correctly for this fixture, by
accident of the source value never being nil, not because the walk did anything. `check.report`'s
subtest therefore adds genuine end-to-end coverage of the map-nested-slice shape reaching the wire
through the real detector pipeline (proving the walk does not corrupt a real, non-synthetic
map-nested value, and guarding against a future detector change that could produce a nil `paths`),
but it is not, today, a guard that would have caught [T52](decision-T52.md)'s original defect for this specific
field — that proof remains `TestNormalizeNilSlices_MapValueNilSliceNormalized`
(`internal/cli/render/json_test.go`), which constructs the nil case directly rather than relying on
a detector that cannot produce one.

### Rationale

A coverage claim that reads as broader than what the code actually proves is a liability at exactly
the moment [D12](decision-D12.md)/`design.md` §5.1 already treat as universal: an assertion
should never claim more than its evidence supports, the identical discipline [P10](../../docs/design.md#4-principles)'s
closed confidence vocabulary enforces for a domain fact, applied here to a test's own claim about
itself. A maintainer auditing test coverage before cutting `v1.0.0` needs `profile.list`/`profile.find`'s
enumeration to read as the recorded, intentional no-op it is — not as evidence of something the code
cannot demonstrate — and needs `check.report`'s subtest to actually exercise the one map-nested case
[T52](decision-T52.md) itself cited as load-bearing, not merely gesture at it. Both halves of this entry serve
that one principle: say only what is true, and make what should be true actually exercised where
the class of defect could hide.

### Consequence

- `internal/cli/jsonnullguard_test.go`: the shared fixture gains `custom_ed25519_dup`;
  `jsonNullGuardArrayFields` gains `"paths"`; `check.report`'s subtest gains a fixture-specific
  positive assertion; the function's own doc comment, and the `profile.list`/`profile.find`
  subtests' comments, state the structural-no-op fact plainly instead of implying uniform coverage.
- `internal/app/investigate.go`'s `originsForInvestigate` doc comment no longer claims the guard
  covers "every kind including this one" without qualification; it names the recorded exception.
- This log's [T52](decision-T52.md) Index row gains a `refined by [T53](decision-T53.md)` Status clause; [T52](decision-T52.md)'s own
  entry prose is unedited, per this log's append-only rule.
- No production code changes: this entry is test-and-documentation-only, correcting what the guard
  proves and what three doc comments claimed it proves, and closing the one genuine coverage gap
  (`check.report`'s map-nested slice) the correction surfaced.
