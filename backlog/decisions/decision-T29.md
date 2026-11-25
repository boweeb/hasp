---
id: decision-T29
title: '`check` findings carry a stable `id` and a re-tunable `severity`'
date: '2026-08-29'
status: 'Accepted · Amends: T14'
---
**Closes:** [`docs/tdd.md`](../../docs/tdd.md) §15's severity-taxonomy question

### Context

[T14](decision-T14.md) makes the JSON envelope a public contract — `hasp list key --json | jq` has to keep
working release over release. `check`'s output is the part of that contract most likely to be
consumed by a program (a `check` in a dotfiles bootstrap, a pre-flight step in CI), and it was the
part left unspecified. `tdd.md` §15 asked whether findings need a `severity` field or whether
`kind` alone suffices, and — before this entry closed it — called the question *"an
interface-polish question for whoever implements `check`'s output schema."*

It is not polish. A schema consumers branch on is design, and settling it after the first consumer
exists means settling it too late.

### Decision

Every finding carries a **stable `id`** and a **`severity`**, and they do different jobs.

```json
{
  "id": "duplicate-key-unconfirmed",
  "severity": "info",
  "subject": {"kind": "key", "name": "id_rsa_old"},
  "message": "possible duplicate, cannot confirm: fingerprint is underivable",
  "detail": {"paths": ["/home/jesse/.ssh/id_rsa_old", "/home/jesse/.ssh/legacy/id_rsa"]}
}
```

**`id` is permanent.** kebab-case, never renamed, never reused for a different meaning — the same
guarantee a `Dn` or `Tn` ID carries. A new finding kind gets a new id; a finding that stops being
reported keeps its id retired rather than recycled.

The v1 set, one per finding `tdd.md` §9's grid already names:

| `id` | Severity | Noun |
| --- | --- | --- |
| `duplicate-key-confirmed` | `warning` | key |
| `duplicate-key-unconfirmed` | `info` | key ([T12](decision-T12.md)) |
| `key-missing-public-half` | `warning` | key |
| `key-no-profile` | `info` | key |
| `fingerprint-unknown` | `info` | key |
| `dangling-identityfile` | `error` | host |
| `unresolvable-token` | `warning` | host |
| `relative-identityfile` | `warning` | host ([T28](decision-T28.md)) |
| `shadowed-stanza` | `warning` | host ([T11](decision-T11.md)) |
| `host-no-binding` | `warning` | host ([T16](decision-T16.md)) |
| `stanza-in-multiple-groups` | `error` | host ([T13](decision-T13.md)'s partial-apply case) |
| `stanza-in-no-group` | `error` | host ([T13](decision-T13.md)'s partial-apply case) |
| `empty-profile-dir` | `info` | profile |
| `unmarked-profile-dir` | `info` | profile |
| `marker-defect` | `error` | host group ([T18](decision-T18.md)) |

**`severity` is advice, and may be re-tuned.** One of `error`, `warning`, `info`. `error` means
something is actually broken — a reference resolving to nothing, or a `Plan` that applied
partially. `warning` means it works but is probably not what was meant. `info` means it is worth
knowing and may be entirely deliberate.

**Severity never affects the exit code.** `check` exits `1` if there is any finding at all, of any
severity, and `0` if there are none — [T14](decision-T14.md)'s rule is unchanged. This is deliberate:
`design.md` §6.2 makes `check` **advisory** and its findings **non-suppressible**, so a severity
that gated the exit code would be a suppression mechanism arriving through the back door. A
consumer wanting to ignore `info` findings filters them in `jq`, visibly, in its own script —
where the decision to ignore something is legible to whoever reads that script.

### Rationale

Splitting the two fields is what allows one of them to be permanent. If consumers filter on
severity, and severity is also what hasp re-tunes as it learns which findings are noisy, then
every re-tuning is a breaking change. Giving `id` the permanence and `severity` the mutability
means hasp can decide next year that `key-no-profile` deserves `warning` without breaking a script
that filters on the id.

The human renderer uses the same two values — severity picks colour and sort order, and the `id`
is printed — so a user reading the terminal can find the same finding in `--json` output without
translating between two vocabularies.

### Consequence

- `tdd.md` §10 documents the finding shape, and §12 gains a guard test asserting the **exact id
  set**, the same mechanism the settings admission rule uses ([T7](decision-T7.md)). A new finding id is then
  visible in a diff rather than discovered by a consumer.
- **`check` remains non-suppressible.** Nothing here adds a way to silence a finding — only a way
  to sort them. If suppression is ever wanted, `design.md` §5.7 and §6.2 already say it needs its
  own decision.
- Adding a finding kind is additive for a consumer filtering positively and needs no envelope
  version bump; `Envelope.Version` ([T14](decision-T14.md)) moves only if the *shape* changes.
