---
Status: APPROVED
DateCreated: 2026-08-28
DateApproved: 2026-08-29
DateLastReviewed: 2026-09-04
Related:
  - "[`docs/README.md`](README.md)"
  - "[`docs/design.md`](design.md)"
  - "[`docs/decision-log.md`](decision-log.md)"
  - "[`docs/tdd.md`](tdd.md)"
  - "[`docs/roadmap.md`](roadmap.md)"
---

# hasp — Technical Decision Log

**Purpose.** The permanent record of *technical* design questions — language, storage
mechanics, parsing strategy, dependencies, distribution — settled downstream of
[`docs/design.md`](design.md). That document is upstream and says nothing about implementation
by design; this file is where implementation questions get settled, the same way
[`docs/decision-log.md`](decision-log.md) settles domain questions. [`docs/tdd.md`](tdd.md) is
the *current* statement of the technical design; this file is *why* it says that, and what was
rejected.

**How to use this file.**

- Entries are append-only. A decision is never edited to say something different — it is
  superseded or amended by a **later** entry that names it.
- IDs are permanent and stable. `T3` means `T3` forever. `T` is a separate namespace from `D` —
  no `T`-ID collides with a `D`-ID, and neither log amends the other's numbering.
- `docs/tdd.md` is the current statement of the technical design and always reflects every
  accepted entry here. When the two disagree, this log is the history and the TDD is the truth.
- An entry needs a **Consequence** to be complete. A decision whose cost nobody wrote down is a
  decision nobody actually made — the rule `docs/decision-log.md` established, kept here without
  modification.
- Every entry cites the principle (P1–P10), decision (D1–D22), or journey (J1–J10) it serves.
  This project settles arguments by appeal to `docs/design.md`; a decision with no citation is
  a decision that gets re-litigated.

**Status values:** `Accepted` · `Open` · `Rejected` — optionally followed by one or more
relation clauses naming another entry: `amends`/`amended by`, `extends`/`extended by`,
`refines`/`refined by`, `supersedes`/`superseded by`, `ratified upstream as` (a `D`-log entry),
`staged by`, `gap closed by`. The three base values are closed; relation clauses are open-ended
and grow as entries need them — the Index table below is the authoritative list of which ones
are actually in use, not this line.

---

## Index

| ID | Decision | Status |
| --- | --- | --- |
| [T1](#t1) | Pure-Go SSH key derivation via `x/crypto/ssh`; no `ssh-keygen`, no `libmagic` | Accepted — extended by [T23](#t23) |
| [T2](#t2) | A hand-rolled, lossless `ssh_config` CST, in place of any existing library | Accepted — amended by [T17](#t17), [T18](#t18), [T24](#t24), [T25](#t25) |
| [T3](#t3) | `spf13/cobra` for the CLI, commands registered from a table | Accepted |
| [T4](#t4) | The `Plan` type: change is represented as data before it is applied | Accepted — amended by [T20](#t20), [T22](#t22), [T26](#t26), [T30](#t30) |
| [T5](#t5) | A host's profile membership is derived from its key bindings | Accepted — amended by [T16](#t16), [T28](#t28) |
| [T6](#t6) | `new key` passphrase handling: dual mode, `--passphrase-stdin`, fail-closed | Accepted — ratified upstream as [D17](decision-log.md#d17); amended by [T39](#t39) |
| [T7](#t7) | A settings file, read-only, reaching P9's last rung | Accepted — ratified upstream as [D16](decision-log.md#d16) |
| [T8](#t8) | Backups live in `~/.ssh/.hasp-backups/`, timestamped, never pruned by hasp | Accepted |
| [T9](#t9) | GoReleaser v2 idioms: `ko`, `nfpms`, `aur`, `homebrew_casks`, `sboms`, `signs` | Accepted — staged by [T33](#t33) |
| [T10](#t10) | Metadata format: sentinel-prefixed TOML fragments as comments | Accepted — amended by [T25](#t25) |
| [T11](#t11) | Host-group composition: one `Include` line per group, hasp orders them | Accepted |
| [T12](#t12) | Key identity: fingerprint when derivable, else canonical path | Accepted |
| [T13](#t13) | DDD adapted to Go: no Unit of Work, no message bus, split aggregate boundary | Accepted |
| [T14](#t14) | Output contract: 4 exit codes, a versioned JSON envelope, silent stdout | Accepted — amended by [T29](#t29), [T31](#t31), [T37](#t37) |
| [T15](#t15) | Safety mechanics: atomic write, symlink-through, mode preservation, D4's move rule | Accepted — refined by [T20](#t20), [T22](#t22); amended by [T30](#t30) |
| [T16](#t16) | Implicit default-identity probing is a distinct, labelled binding kind | Accepted — amends [T5](#t5); amended by [T28](#t28) |
| [T17](#t17) | `Directive` preserves its exact separator and spacing; quote-aware tokenizing | Accepted — amends [T2](#t2) |
| [T18](#t18) | CST marker defects are represented as data, not parse errors | Accepted — amends [T2](#t2) |
| [T19](#t19) | Alias location is the mechanism for cross-profile key membership — D2's write path | Accepted |
| [T20](#t20) | Plan ordering: any prefix leaves a working state; `adopt`'s alias-preserving move | Accepted — amends [T4](#t4), [T15](#t15); amended by [T43](#t43) |
| [T21](#t21) | `show profile` aggregates descendants by default, `--no-recurse` escape | Accepted |
| [T22](#t22) | `WriteKeyFile` fails closed on an existing target; a `Remove` change kind | Accepted — amends [T4](#t4), [T15](#t15); amended by [T44](#t44) |
| [T23](#t23) | DSA is in scope for the read path; fixtures are PEM-only, hand-constructed | Accepted — extends [T1](#t1) |
| [T24](#t24) | Line-terminator handling in the CST: preserved as trivia, excluded from `Args` | Accepted — amends [T2](#t2) |
| [T25](#t25) | `MetadataLine` is its own CST node type for `#:hasp` lines | Accepted — amends [T2](#t2), [T10](#t10) |
| [T26](#t26) | `WriteRegion` previews carry a real diff; key material is never diffed | Accepted — amends [T4](#t4); gap closed by [T30](#t30) |
| [T27](#t27) | Go, and a fresh implementation rather than a repair of the predecessor | Accepted — closes `design.md` §10's stack item |
| [T28](#t28) | Relative `IdentityFile` resolves against the key directory, and says so | Accepted — amends [T5](#t5), [T16](#t16) |
| [T29](#t29) | `check` findings carry a stable `id` and a re-tunable `severity` | Accepted — amends [T14](#t14); extended by [T31](#t31); amended by [T37](#t37) |
| [T30](#t30) | The preview/apply race is detected by a witness, and fails closed | Accepted — amends [T4](#t4), [T15](#t15) |
| [T31](#t31) | Semantic versioning, and the compatibility surface v1.0.0 freezes | Accepted — amends [T14](#t14), [T29](#t29); amended by [T37](#t37), [T40](#t40) |
| [T32](#t32) | Mage is the build/CI contract; platform workflows are thin shims | Accepted — amended by [T40](#t40), [T41](#t41) |
| [T33](#t33) | Distribution is staged: self-hosted now, public channels blocked on one missing fact | Accepted — amends [T9](#t9); amended by [T40](#t40) |
| [T34](#t34) | User-facing documentation is generated wherever it can drift | Accepted |
| [T35](#t35) | The fingerprint scheme registry: open, pure-Go, no subprocess, no network | Accepted |
| [T36](#t36) | `find` matches across every registered scheme; the clue's shape routes the search | Accepted — amends `tdd.md` §9 |
| [T37](#t37) | Confidence is a closed, permanent vocabulary in the output contract | Accepted — amends [T14](#t14), [T29](#t29), [T31](#t31) |
| [T38](#t38) | `ssh-agent` is a derivation source for public facts, and its contribution is labelled | Accepted |
| [T39](#t39) | Passphrase-gated derivation: explicit, lazy, one passphrase per invocation, degrades without a TTY | Accepted — amends [T6](#t6); amended by [T49](#t49) |
| [T40](#t40) | The public origin is GitHub; T33's shared blocker resolves and distribution restages | Accepted — amends [T31](#t31), [T32](#t32), [T33](#t33) |
| [T41](#t41) | `CI`'s `mg.Deps` aggregate excludes `Fixtures`: non-deterministic fixtures race `Test` and break `TestFullInventory` | Accepted — amends [T32](#t32) |
| [T42](#t42) | Signing covers both the release checksum and the `kos`-built container image, via two GoReleaser sections | Accepted — amends [T9](#t9) |
| [T43](#t43) | `adopt key`'s alias-preserving move is one atomic `Change`, not an ordered two-`Change` pair | Accepted — amends [T20](#t20) |
| [T44](#t44) | `edit key --replace-material` routes through `WriteKeyFile{AllowOverwrite: true}`, not a separate move rule | Accepted — amends [T22](#t22) |
| [T45](#t45) | The clean-room test: `tools/cleanroom`, a `CleanRoom` Mage target, a post-release CI job, and a shared `internal/snapshot` package | Accepted |
| [T46](#t46) | Container base image moves from Chainguard to Google's distroless (`cgr.dev/chainguard/static` → `gcr.io/distroless/static:nonroot`) after Chainguard gated free registry access behind a business-email requirement | Accepted |
| [T47](#t47) | The fingerprint-scheme no-network guard is a `go list` direct-import assertion, not the `net.Dialer`-`Control` mechanism `tdd.md` §12 originally named | Accepted — amends [T35](#t35) |
| [T48](#t48) | `find key` never prompts for a passphrase; an inapplicable-for-that-reason scheme is an explicit warning, not a silent miss | Accepted |
| [T49](#t49) | Narrowing §18/T39's comment-recovery claim: a passphrase prompt cannot recover an OpenSSH-format key's comment; only `ssh-agent` can | Accepted — amends [T39](#t39) |
| [T50](#t50) | `find`'s comparison folds case and separator punctuation at compare time; `fpscheme.Normalize` stays exactly as it is | Accepted |

---

<a id="t1"></a>
## T1 — Pure-Go SSH key derivation via `x/crypto/ssh`; no `ssh-keygen`, no `libmagic`

**Date:** 2026-08-28 · **Status:** Accepted

### Context

[`design.md` §5.1](design.md#51-key) defines the derivation gap: a three-row truth table for
whether a fingerprint is derivable, closed with the requirement that hasp report the undecidable
case as `unknown` rather than prompt for a passphrase (P3) or persist a guess (D12). [D12](decision-log.md#d12)
requires every fact be recomputed on demand; P8 requires it be cheap enough to do so on every
invocation. The read path must determine fingerprint, algorithm, format, encryption state, and
comment for the full key matrix — ed25519, RSA, ECDSA, in both OpenSSH and legacy PEM framing,
encrypted or not, with or without a `.pub` file — from raw bytes, without ever decrypting
anything.

The predecessor project needed two external dependencies to do this: an `ssh-keygen`
subprocess and `libmagic` (via `python-magic`) for format sniffing — both named in
[`docs/project-assessment-2026-08.md` §4](project-assessment-2026-08.md) as toolchain weight,
and the former as a source of `Cannot load private key: incorrect passphrase`-shaped failures
that D14 later used as evidence against ever touching an encrypted private key file at all.

### Decision

The entire read path is pure Go, using `golang.org/x/crypto/ssh`. No subprocess is exec'd, no
cgo dependency is introduced. Confirmed against the installed module (`v0.55.0`) this session:

```go
// x/crypto/ssh/keys.go
type PassphraseMissingError struct {
    // PublicKey will be set if the private key format includes an unencrypted
    // public key along with the encrypted private key.
    PublicKey PublicKey
}

func ParseRawPrivateKey(pemBytes []byte) (interface{}, error)
```

`ParseRawPrivateKey` switches on the PEM block's `Type` field — `"OPENSSH PRIVATE KEY"` versus
`"RSA PRIVATE KEY"` / `"EC PRIVATE KEY"` / `"DSA PRIVATE KEY"` / bare `"PRIVATE KEY"` (PKCS#8) —
which gives `format` for free, unencrypted, no decryption attempted. Whether the key is
encrypted is likewise readable without decrypting it: `Proc-Type: 4,ENCRYPTED` /
`DEK-Info` headers for legacy PEM, the cipher name field inside the OpenSSH wire format for the
modern one.

For the undecidable case, the function returns `*PassphraseMissingError` and the struct's own
`PublicKey` field is the typed discriminator design.md's three-row table needs:

| Case | What `ParseRawPrivateKey` does |
| --- | --- |
| Public half present | Never reaches this path — hasp parses the `.pub` via `ParseAuthorizedKey`, then `FingerprintSHA256` |
| OpenSSH format, encrypted, no `.pub` | Returns `*PassphraseMissingError` with **`.PublicKey` populated** — `parseOpenSSHPrivateKey` reads it from the embedded, unencrypted `w.PubKey` field before it ever touches the ciphertext |
| Legacy PEM, encrypted, no `.pub` | Returns `&PassphraseMissingError{}` with **`.PublicKey == nil`** — the `encryptedBlock(block)` check fires before any parsing that could populate it |

Reading `err.PublicKey != nil` after a type assertion on `*ssh.PassphraseMissingError` *is* the
derivation-gap check. No stderr scraping, no exit-code interpretation, no format guess.

### Rationale

This directly satisfies P3 on the read path — encryption state and format are readable without
ever asking for, or attempting, decryption — and it satisfies P8: no process fork, no cgo build
step, per key read. It removes both of the predecessor's external dependencies at once, which
also feeds [T9](#t9): a statically linked, `CGO_ENABLED=0` binary is only genuinely
`FROM scratch`-capable if nothing in its call graph shells out to a tool that may not exist in
the container.

Also available on the same package, used elsewhere in the read/write path: `ParseAuthorizedKey`
(parses a `.pub` line), `FingerprintSHA256` (the `SHA256:`-prefixed, unpadded-base64 form
OpenSSH ≥6.8 uses — this is the fingerprint format hasp reports, never the legacy MD5 colon-hex
form, though `FingerprintLegacyMD5` exists if a future `find key` clue needs to match one),
`NewPublicKey`, `MarshalAuthorizedKey`, `MarshalPrivateKey`, and
`MarshalPrivateKeyWithPassphrase` — the last two are how `new key` ([T6](#t6)) authors a file,
the one write path D14 permits.

### Consequence

- Zero runtime dependency on `PATH` containing `ssh-keygen`, and zero cgo — `CGO_ENABLED=0`
  build in [T9](#t9) is unconditionally achievable, not merely likely.
- The derivation-gap table stops being documentation asserted by hand and becomes a fact a test
  suite checks directly: construct a fixture in each of the three rows, assert the concrete
  return type and the `PublicKey` field's nilness. This is the seed of the fixture matrix in
  [`tdd.md` §12](tdd.md#12-testing-strategy--a-function-of-a-directory).
- No code path in `internal/adapter/keyfile` may call anything that supplies a passphrase to
  *decrypt* an existing key — there is no `ParsePrivateKeyWithPassphrase` call for reading, and
  a future contributor adding one to "helpfully" resolve an unknown fingerprint is adding a P3
  violation, not a feature. This absence is the enforcement mechanism, not a comment asking
  someone not to.

---

<a id="t2"></a>
## T2 — A hand-rolled, lossless `ssh_config` CST, in place of any existing library

**Date:** 2026-08-28 · **Status:** Accepted — amended by [T17](#t17), [T18](#t18),
[T24](#t24), [T25](#t25)

### Context

[D7](decision-log.md#d7) requires two things of the same subsystem simultaneously: outside a
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
`README.rst` (§6 below closes this as scope, matching [`tdd.md` §6](tdd.md#6-configuration-parsing--a-lossless-cst-for-a-file-hasp-does-not-fully-own)):
`HostName`, `IdentityFile`, `User`, `Port`, `ProxyCommand`, `ControlMaster`, `ControlPath`,
`ControlPersist`, `AddKeysToAgent`, `Ciphers`, `ForwardAgent`, `HashKnownHosts`,
`HostKeyAlgorithms`, `IdentitiesOnly`, `KexAlgorithms`, `LogLevel`, `MACs`,
`PasswordAuthentication`, `PubkeyAuthentication`, `StrictHostKeyChecking`,
`UserKnownHostsFile` — plus `Include`, which hasp itself writes ([T11](#t11)) and must
therefore understand semantically inside its own regions.

P8 also argues against a general-purpose dependency here: this is a personal tool, and the
libraries above carry surface area (fleet-scale config composition, `Match` blocks with
arbitrary criteria) hasp does not need and would rather not inherit the maintenance burden of.

### Consequence

- This is now hasp's largest first-party subsystem, and its correctness gate is a fuzz test
  (`FuzzSSHConfigRoundTrip`, [`tdd.md` §12](tdd.md#12-testing-strategy--a-function-of-a-directory)),
  not a hand-written unit-test suite alone.
- Teaching hasp a new directive later — widening the 21-directive native set — is additive: the
  parser already carries it as an opaque `Directive` node today, so nothing needs to migrate on
  existing files. This is the direct payoff of D7 elaboration 3 the design doc predicted.
- Every future change to this package must run the fuzz corpus before merging; a regression here
  is a P2 violation with no smaller blast radius than "silently mangled the user's config."

**Four later entries harden this contract rather than change it.** The node sketch stated here
was necessary but not sufficient: [T17](#t17) fixes `Directive` so it actually satisfies
`Render(Parse(b)) == b` for every valid separator/quoting form `ssh_config(5)` allows, which the
original sketch's parsed-`Args`-only shape could not; [T18](#t18) gives "a marker or region is
malformed" — asserted elsewhere in this project but never defined here — concrete, detectable
shapes; [T24](#t24) closes a gap the contract implied but the node sketch didn't handle, CRLF and
other line-terminator variation; [T25](#t25) gives `#:hasp` metadata lines (§7) their own node
rather than leaving them to be re-scanned out of an opaque `Line`.

---

<a id="t3"></a>
## T3 — `spf13/cobra` for the CLI, commands registered from a table

**Date:** 2026-08-28 · **Status:** Accepted

### Context

The interface layer needs a command framework. [D10](decision-log.md#d10) fixes the grid at
3 nouns × 8 verbs (24 first-class combinations) with second-class nouns surfacing as flags; the
predecessor's one genuinely good idea, per
`docs/project-assessment-2026-08.md` §2.1, was that this grid was **mechanically** registered —
`cli/verb/<verb>.py` imported the same-named function from each `cli/resource/*.py` — so adding
a noun cost nothing structurally. That property is worth preserving in Go.

### Decision

**`spf13/cobra`**, over `alecthomas/kong`, which was seriously considered.

Commands are registered from a single Go table rather than built up ad hoc, so the grid's
uniformity survives the language change:

```go
type CommandSpec struct {
    Verb, Noun string
    Build      func(deps Deps) *cobra.Command
}

var commands = []CommandSpec{
    {Verb: "list", Noun: "key", Build: buildListKey},
    {Verb: "show", Noun: "key", Build: buildShowKey},
    // ... 24 entries, one per cell in tdd.md §9
}
```

### Rationale

`kong` is arguably the better technical fit for a rigid, statically-known grid — struct-tag
declaration is terser than cobra's imperative `*cobra.Command` construction, and hasp's grid is
about as static as a CLI's surface gets. It loses on two points that matter here: cobra
generates shell completions and man pages as first-class subcommands
(`cobra.Command.GenManTree`, the built-in `completion` command), both of which
[T9](#t9)'s GoReleaser packaging consumes directly (nfpms/completions archive entries); and
cobra is the ecosystem default, which matters for a project whose entire prior failure mode was
unfinished infrastructure choices (`docs/project-assessment-2026-08.md` §5, "Verb×resource
matrix or `factory/`? ... you never picked one"). This call is routine enough not to need the
user's sign-off, and consequential enough to name so it can be overridden.

### Consequence

- Adding a noun means adding entries to `commands` and to `internal/app`'s use-case set; the CLI
  tree itself never needs hand-editing beyond the table.
- If the table ever fights cobra's construction model badly enough to erase this benefit, `kong`
  remains the documented alternative — this entry is the record of why it was not chosen first,
  not a claim that it never could be.
- Generated completions and man pages are a packaging concern owned entirely by
  [T9](#t9)/[`tdd.md` §13](tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought);
  this entry only establishes that cobra makes them available.

---

<a id="t4"></a>
## T4 — The `Plan` type: change is represented as data before it is applied

**Date:** 2026-08-28 · **Status:** Accepted — amended by [T20](#t20), [T22](#t22), [T26](#t26)

### Context

[D6](decision-log.md#d6) makes preview the default for **every** write, with no
operation-by-operation judgment call about which changes are risky enough to deserve it. P5
states this explicitly as a modeling constraint, not an interface nicety: *"the ability to
preview is not a convenience flag bolted on later; it constrains how change is modeled from the
beginning."* A design where a use case performs I/O directly and merely *logs what it did*
cannot satisfy this — preview must exist **before** anything is touched, which means the thing
previewed must be a value, not a side effect already in flight.

### Decision

Every write-verb use case returns a `Plan` — an ordered list of typed `Change` values — instead
of performing I/O. A single `Applier` is the only thing in the codebase permitted to touch the
filesystem for a write.

```go
type Plan struct {
    Summary string
    Changes []Change
}

type Change interface {
    Describe() string        // one-line preview text, rendered by both the human and JSON renderers
    RequiresBackup() bool
    Apply(fsys WriteFS) error
}

type MoveFile struct{ From, To, Reason string }
type WriteRegion struct{ File string; Marker RegionID; Body []byte }
type CreateSymlink struct{ Path, Target string }
type CreateMarker struct{ Dir string; Header []byte }
type WriteKeyFile struct{ Path string; Contents []byte; Mode fs.FileMode } // the D14 exception

type Applier struct {
    FS      WriteFS
    Backups BackupStore
}

func (a *Applier) Apply(p Plan) (Result, error) {
    for _, c := range p.Changes {
        if c.RequiresBackup() {
            if err := a.Backups.Snapshot(c); err != nil {
                return Result{}, fmt.Errorf("backup failed, aborting before write: %w", err)
            }
        }
        if err := c.Apply(a.FS); err != nil {
            return Result{}, err
        }
    }
    return Result{Applied: p.Changes}, nil
}
```

A use case's method signature is `Plan(req Request) (Plan, error)` — it may perform *reads* to
compute the plan (P5: reads are always safe), but returns before anything is written.

**Three later entries refine this type sketch rather than replace it: [T20](#t20)** states the
general ordering rule for `Changes` and adds no new type; **[T22](#t22)** adds an
`AllowOverwrite` field to `WriteKeyFile` and a `Remove` change kind; **[T26](#t26)** replaces
`Describe() string` with `Preview() Preview`, a structured value carrying an optional diff,
because `Describe`'s single line could not represent what a `WriteRegion` rewrite actually
changes.

### Rationale

This is the architectural keystone the plan for this document names explicitly: D6 is only
structural, rather than a flag bolted onto each command, if change is data first and an
executable action second. It is the Go-appropriate stand-in for the Unit-of-Work pattern Cosmic
Python uses for a different purpose ([T13](#t13) states why UoW itself does not port).

**The resulting inversion is worth stating in exactly these terms: `--dry-run` does not exist,
because preview *is* the default.** The flag that exists is `--yes` — its absence is what
triggers the preview-only path, not its presence.

### Consequence

- Every use case in `internal/app` is testable without touching a filesystem: assert on the
  returned `Plan` value. This is the design that makes the guard test in
  [`tdd.md` §12](tdd.md#12-testing-strategy--a-function-of-a-directory) ("hasp writes nothing
  outside the key directory") tractable to write per-use-case, not just end-to-end — the
  `testscript` `.txtar` cases in that same section cover the full end-to-end path.
- New `Change` kinds are additive and carry their own `RequiresBackup()` answer — nothing about
  the safety machinery depends on a call site remembering to ask for a backup; it is a property
  of the change, not of the caller's diligence.
- A `Plan` with zero `Changes` is a legitimate, renderable value ("nothing to do") — every
  write-verb use case can report "no change needed" through the same preview path used for a
  real change, so there is no separate no-op code path to keep in sync.

---

<a id="t5"></a>
## T5 — A host's profile membership is derived from its key bindings

**Date:** 2026-08-28 · **Status:** Accepted — amended by [T16](#t16)

### Context

[D13](decision-log.md#d13) settled how a **key's** profile membership is expressed — the
directory it lives in — but [`design.md` §5.4](design.md#54-host) only asserts that a host has
*"two independent affiliations... who it belongs to (a profile) and where it physically lives (a
host group)"* without saying how the first one is computed. Left unresolved, this is exactly the
kind of gap D13's own preamble warns about: something hasp would have to be *told*, reopening
the closed §5.7 "Intent" category.

### Decision

**A host's profile set is the union of the profile sets of the keys its `IdentityFile` lines
resolve to**, via the binding resolution mechanism in
[`tdd.md` §5](tdd.md#5-derivation-pipeline--scan-classify-resolve-project). No metadata is
declared for this. A host resolves to zero or more keys (D2 permits a key to belong to many
profiles; nothing prevents a host from listing more than one `IdentityFile`), and its profile
set is exactly the union across all of them.

Because a key's own profile membership already accounts for [§5.3](design.md#53-profile)'s
`.hasp`-marker rule (a directory is a profile only if marked), this derivation needs no separate
marker check for hosts — it reuses the already-computed key-profile-set verbatim.

**[T16](#t16) adds a second binding source.** This entry, as written, resolved only *explicit*
`IdentityFile` lines. A stanza with none is not a stanza with zero bindings — `ssh` itself still
tries its own default identity files — and T16 extends "the keys its bindings resolve to" to
cover both.

### Rationale

This keeps [D12](decision-log.md#d12) literally true with **no new asterisk**: nothing is
declared for hosts that the filesystem plus the config file don't already carry. It works on
**unmanaged** hosts too, since the computation only needs key *location*, not host management
state — a stanza hasp has never wrapped in markers still gets a computed profile, which is what
lets M1's read-only survey (D14) report host-profile membership before `adopt` exists at all,
strengthening J1 and J3.

The rejected alternative — declaring host→profile membership as metadata inside a marked region,
which D7's mechanism ([T10](#t10)) makes possible — was rejected on P9's own ladder: it would
duplicate a fact fully computable from structure that already exists, and P9 requires showing
existing structure genuinely *cannot* carry the fact before falling back to declaration. It can.

### Consequence

- A host with no `IdentityFile` line belongs to no profile — the same legitimate state as a
  top-level key ([§5.1](design.md#51-key)) — and it is a `check` finding, mirroring the language
  D13 already uses for the key case.
- A host bound to a key shared across profiles (D2) inherits **every one of them**. This is
  exactly what J8 (offboard) needs — *"show me everything scoped to that profile"* — walked
  backwards from key to host along the binding relation ([§5.6](design.md#56-binding)), and is
  the concrete reason D11 called binding-as-a-relation load-bearing.
- If a resolved `IdentityFile` target sits in a directory *without* a `.hasp` marker, it
  contributes nothing to the host's profile set — a plain unmarked directory carries no
  taxonomy, by [§5.3](design.md#53-profile)'s own rule, and the host derivation must not invent
  an exception for itself.

---

<a id="t6"></a>
## T6 — `new key` passphrase handling: dual mode, `--passphrase-stdin`, fail-closed

**Date:** 2026-08-28 · **Status:** Accepted · **Ratified upstream 2026-08-29 as**
[D17](decision-log.md#d17)

### Context

`new key` needs a way to support a passphrase-protected key, because non-interactive execution
(`--passphrase-stdin`, or a settings-file default, [T7](#t7)) requires the choice to be
resolvable without a human present to answer a prompt. The honest starting point is that
**nothing in `design.md` licenses hasp asking for a passphrase.** §3.2 states the opposite as one
of nine hard boundaries, each one described as turning hasp into *"a different and worse
project"* if crossed: *"Not a passphrase manager. hasp never asks for, stores, or transmits a
passphrase."* [`design.md` §5.1](design.md#51-key)'s carve-out — *"hasp never writes to a private key file. The
single exception is generating a new key, where hasp authors the file outright"*, which
[D14](decision-log.md#d14) states in the same terms — licenses hasp **authoring** a key file. It
says nothing about
**asking** for a passphrase while doing so; those are different acts, and an earlier draft of
this entry conflated them, presenting the carve-out as if it settled the question it does not
address. That was a citation error, corrected here rather than left standing.

What follows is a deliberate, needed capability — not a reading of an existing exception. It
**narrows a stated hard boundary**, which this project's own discipline treats the same way as
any other gap in the ratified design: named plainly and flagged for ratification, not
reinterpreted into something `design.md` already permits when it does not.

### Decision

`new key` supports **both** answers — no passphrase, or a passphrase supplied at generation
time — and enforces a hard boundary at the type level: a passphrase is accepted **only** at
generation, **never** to decrypt an existing key, is **never persisted anywhere** (not in
settings, not in metadata, not logged), and its buffer is zeroed immediately after use.

Three mutually exclusive flags on `new key`:

| Flag | Effect |
| --- | --- |
| `--passphrase` | Prompt interactively via `golang.org/x/term.ReadPassword(int(os.Stdin.Fd()))` — no local echo |
| `--no-passphrase` | Generate without one, explicitly |
| `--passphrase-stdin` | Read the secret bytes from stdin — for non-interactive/scripted use |

Mode-selection precedence (which of the three applies when none is given as a flag):
**flag → env var (`HASP_NEW_KEY_PASSPHRASE_MODE`, one of `none`/`prompt`/`stdin`) → settings
file (`[new_key] default_passphrase_mode`, [T7](#t7)) → built-in default.**

The built-in default is **"prompt interactively if stdin is a TTY; otherwise, refuse."** This is
deliberately not "assume no passphrase" — silently generating an unprotected key because nobody
asked is exactly the failure P5/D6's non-interactive-consent language is written against.
**Non-interactive execution with no explicit mode chosen anywhere in the chain is fail-closed**:
`new key` exits with `ExitUsage` ([T14](#t14)) and a message naming the three flags, rather than
guessing.

Once a mode resolves to `stdin`, the actual secret bytes are **always** read from `os.Stdin` at
the moment of generation — never from the env var (which only ever carries the *mode name*, not
the secret — env vars are readable via `/proc/<pid>/environ` and process-listing tools on some
systems, which is a real leak path a flag argument or an env-carried secret would both have),
never from settings, never from a CLI argument (which would land in shell history and `ps`
output).

### Rationale

The type-level enforcement: nowhere in `internal/domain` or `internal/app` does a function accept
a passphrase alongside an *existing* `Key` value. The only function that accepts passphrase bytes
is the one building a `WriteKeyFile` change inside `NewKeyUseCase.Plan`, which calls
`ssh.MarshalPrivateKeyWithPassphrase(key, comment, passphrase)` and defers zeroing the slice. This
absence is itself the audit mechanism (P6: legible without hasp running — a reviewer reading a
diff that adds passphrase-accepting code anywhere else in the tree is reading a P3 violation on
sight, not a subtle one).

### Consequence

- **An upstream `D17` was recommended from here, and written there** — flagged rather than
  authored, per the ratifying author's own stated process. **Ratified 2026-08-29 as
  [D17](decision-log.md#d17)**, which records that §3.2's passphrase-manager boundary is narrowed
  by exactly this case, for exactly this reason, and that the mechanics above (generation-time
  only, never for decryption, never persisted, buffer zeroed) are the entire scope of the
  narrowing — not a general license to ask for a passphrase anywhere else.
- `new key`'s flag surface grows by three flags, mutually exclusive, and the full precedence
  resolver stays small — on the order of ~50 lines to walk four sources in order — which is
  exactly why hasp does not depend on `viper` for this (stack decision,
  [`tdd.md` §2](tdd.md#2-stack--go-stdlib-first-and-a-dependency-budget)).
- A CI/scripted caller of `hasp new key` **must** pass `--no-passphrase` or `--passphrase-stdin`,
  or must have configured a default in settings ([T7](#t7)) — there is no path by which a
  script silently produces an unintentionally unprotected key, and no path by which it hangs
  waiting on a TTY prompt that will never come.
- `hasp new key`'s recorded facts (algorithm, format, fingerprint, encrypted-or-not) are derived
  by re-reading the file hasp just wrote through the same [T1](#t1) code path used for every
  other key — never asserted from the generation request — closing J4's requirement that a new
  key's recorded facts be *identical* to what a fresh inventory of the finished artifact reports.

---

<a id="t7"></a>
## T7 — A settings file, read-only, reaching P9's last rung

**Date:** 2026-08-28 · **Status:** Accepted · **Ratified upstream 2026-08-29 as**
[D16](decision-log.md#d16)

### Context

P9's taxonomy ladder ends: *"Only as a last resort, a hasp settings file — and no case has yet
required one."* [T6](#t6)'s non-interactive default-passphrase-mode requirement is **the first
case that does.** This is not a detail to fold quietly into T6 — the project's own discipline
(stated identically in `design.md`'s own preamble) is that a gap in the ratified design gets
*amended*, not reinterpreted, and this is precisely that kind of gap.

*(That is P9 as it stood when this entry was written. [D16](decision-log.md#d16) — which this
entry prompted — has since amended its final clause, so `design.md` §4 no longer says "no case
has yet required one." The quotation above is preserved as the text this entry actually reasoned
against.)*

[D3](decision-log.md#d3) already anticipated this: *"hasp's own settings are settings, are not
part of the verb grid, and are edited directly."* This entry is walking through a door D3 already
named, not cutting a new one.

### Decision

- A settings file exists, in **TOML**, with `#` comments — the same convention D15 already
  argued for on the neighboring `.hasp` marker (*"the neighbouring file in that very directory —
  `ssh_config` — uses `#`, as do shell, `gitconfig`, TOML, and YAML"* — P9 rides the convention
  that already exists rather than inventing a second one).
- **Location:** `os.UserConfigDir()/hasp/settings.toml` — stdlib, no XDG library. On Linux this
  is `$XDG_CONFIG_HOME/hasp/settings.toml` or `~/.config/hasp/settings.toml`; on Darwin,
  `~/Library/Application Support/hasp/settings.toml`. Deliberately **outside** `~/.ssh` — it is
  not key material, not arrangement, and must never be swept into the profile scanner or the
  `~/.ssh/.hasp-backups/` rotation ([T8](#t8)).
- **hasp reads settings and never writes them.** There is no `hasp config set` and no
  `hasp settings set` — "config" is not a noun (D3), and neither is a settings-mutation verb
  added to the grid. The file is edited directly, with an editor, exactly as P6 already promises
  for everything hasp knows.
- **The admission rule**, stated because this file has the exact shape of the project's original
  failure mode: **settings may hold *only* user preferences that (a) cannot be derived, and
  (b) exist to enable non-interactive execution.** They may **never** hold facts about the
  machine (P1/D12 — that is precisely what the abandoned TOML state file held, per
  `docs/project-assessment-2026-08.md` §5.1's "TOML buys a human-editable... state file"), and
  they may **never** hold taxonomy (P9 rungs 1–2 — a profile assignment written into settings
  would be D13's rejected option (iii), readmitted through a different door than the one D13
  actually closed). Anything that fails this test stays a flag.
- **v1 keyset — deliberately minimal**, because nothing else has yet cleared the bar:

  ```toml
  # ~/.config/hasp/settings.toml
  # hasp reads this file. hasp never writes to it.
  [new_key]
  default_passphrase_mode = "prompt"   # "none" | "prompt" | "stdin"
  ```

- **Optional, throughout.** Absence of the file is not an error. Every settings key has a
  built-in default ([T6](#t6)'s "prompt if TTY, else refuse") that makes hasp fully useful with
  zero configuration, preserving [`design.md` §6.3](design.md#63-cross-cutting-requirements)'s
  *"works on first run, on a machine it has never seen, with nothing configured"* exactly as
  written.

### Rationale

TOML reuses the same decoder [T10](#t10) already needs for metadata fragments — one parsing
dependency, not two. The admission rule is what prevents this from becoming "the TOML state file,
take two": the original sin, per D12's own rationale, was *persisting derivable facts* — this
file structurally cannot, because anything derivable fails the rule on sight and is refused a
place in it.

### Consequence

- **An upstream `D16` was recommended from here, and written there** — flagged rather than
  authored, per the ratifying author's own stated process. **Ratified 2026-08-29 as
  [D16](decision-log.md#d16)**, which records that P9's last rung was reached, by what case, and
  carries the admission rule as a permanent constraint on this file's growth rather than as this
  session's judgment call.
- Every future PR proposing a new settings key must justify it against the admission rule in its
  own right, not merely add a TOML field because it was convenient — that discipline is the
  entire point of writing the rule down here rather than letting the file grow the way the
  original state file did, one convenient field at a time.
- `hasp` never needs to create this file. `new key` and every other command that consults it
  treats a missing file identically to an empty one.
- **The admission rule is enforced mechanically, not left to review discipline** — a test
  asserts the exact field set of the Go struct settings decode into
  ([`tdd.md` §12](tdd.md#12-testing-strategy--a-function-of-a-directory)). This is the same class
  of guard [T1](#t1) gets from the absence of a decrypt call site and [T13](#t13) gets from
  `go list -deps`: the rule most directly guarding against a repeat of the predecessor's TOML
  state file does not get to rely on a reviewer noticing a new field.

---

<a id="t8"></a>
## T8 — Backups live in `~/.ssh/.hasp-backups/`, timestamped, never pruned by hasp

**Date:** 2026-08-28 · **Status:** Accepted

### Context

P4: *"Before hasp modifies anything it did not create, the prior state is preserved... the user
should never need to have thought ahead."* D6's preview → confirm → back up → write cycle
requires a concrete location and mechanism, and [§5.10](design.md#510-what-is-deliberately-absent)
is explicit that hasp is not to grow a state file or database in the process of providing it.

### Decision

Backups live at `~/.ssh/.hasp-backups/`, as timestamped copies —
`.hasp-backups/config.20260828T140501Z` — inheriting the key directory's own `0700` mode,
restorable with a plain `mv`, and excluded from the profile scanner via the same
dotfile/marker-exclusion rule that already skips `.hasp` markers themselves (this is a reused
exclusion, not a new one).

### Rationale

P4 requires the backup to be automatic on **every** qualifying write, not opt-in — a rolling
backup directory inside the same tree the write touches satisfies that with no separate
subsystem. It sits **inside** `~/.ssh`, deliberately, rather than beside settings
([T7](#t7))'s `os.UserConfigDir()` location: a backup is a copy of *this directory's own
content* and belongs next to it, restorable with the same tools — `ls`, `mv`, `cp` — a human
already has for everything else here (P6).

### Consequence

This precisely completes hasp's write-surface invariant, now stated exactly:
**hasp writes to exactly one directory tree — the key directory, including its
`.hasp-backups/` subtree — and reads at most one optional settings file it never writes.** This
is the literal assertion the guard test in
[`tdd.md` §12](tdd.md#12-testing-strategy--a-function-of-a-directory) checks: no file is ever
created, modified, or deleted outside that one tree, for any operation, in any test run.

A backed-up key file is itself key material under D4's canon — hasp therefore never prunes,
rotates, or garbage-collects `.hasp-backups/` automatically. Retention is the user's own act,
with their own tools (`rm`), matching D4's philosophy exactly: hasp's job is to never be the
reason a copy of a secret disappears, not to decide when enough copies exist.

---

<a id="t9"></a>
## T9 — GoReleaser v2 idioms: `ko`, `nfpms`, `aur`, `homebrew_casks`, `sboms`, `signs`

**Date:** 2026-08-28 · **Status:** Accepted

### Context

GoReleaser is a stated constraint on this project, to be embraced rather than worked around.
`design.md` §13 (via [`tdd.md`](tdd.md)) needs: `CGO_ENABLED=0` (made unconditionally achievable
by [T1](#t1)), a container image path, and package-manager coverage that includes the Arch User
Repository, since the dev host is Arch Linux.

### Decision

Verified against the installed `2.14.0` this session, with docs current at v2.18:

| Section | Status | hasp uses it for |
| --- | --- | --- |
| `ko` | Current, recommended | The container image — no `Dockerfile`, `FROM scratch`-capable because of [T1](#t1) |
| `dockers` / `docker_manifests` | **Deprecated** | Not used |
| `dockers_v2` | Current (the Docker-specific route) | Not used — `ko` is preferred over hand-writing a Dockerfile for a statically linked binary |
| `nfpms` | Current | `.deb` / `.rpm` / `.apk` packages |
| `aur` | Current | Arch User Repository — the dev host's own package manager |
| `brews` | **Deprecated** | Not used |
| `homebrew_casks` | Current (`brews`'s replacement) | macOS distribution |
| `sboms` | Current | Supply-chain attestation |
| `signs` | Current | Artifact signing |

Four OS/arch build targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` —
matching the "macOS + Linux long-term" scope, dev host x86 Arch Linux via `mise` (`go = "latest"`,
currently `go1.27.1`).

Generated shell completions and the man tree (from [T3](#t3)'s cobra, via
`cobra.Command.GenManTree` and the built-in `completion` subcommand) are packaged into the
release archives, not left to `go install` users to generate themselves.

### Rationale

Embracing GoReleaser's idioms rather than working around them means letting its recommended path
(`ko`) determine a hasp design constraint, not the reverse: `ko`'s default posture assumes a
statically linked binary with no runtime dependency on anything not already in a distroless base
image. That assumption only holds because [T1](#t1) removed the `ssh-keygen` subprocess and
`libmagic` cgo dependency the predecessor needed — had either survived, `ko`'s scratch-based
image would silently break at the first key read, which is exactly why T1 and T9 are one linked
decision, not two independent ones, despite being numbered separately.

### Consequence

- `dockers`, `docker_manifests`, and `brews` are explicit **non-choices**: a future contributor
  reaching for any of them is reaching for a deprecated GoReleaser v2 section, and this entry is
  the record of why.
- Build info (`-X`-injected via ldflags: version, commit, build date) is consumed by
  `hasp version`; `runtime/debug.ReadBuildInfo()` (`func ReadBuildInfo() (info *BuildInfo, ok
  bool)`) is the fallback for a `go install`-built binary that never went through GoReleaser at
  all, so `hasp version` never reports "unknown" for a build that has *any* module information
  available.
- **Container caveat, stated plainly:** hasp manages the machine it runs *on* — §3.2 is explicit
  that hasp is *"not configuration management... does not reach across a fleet."* A container
  image is therefore for CI/scripted use, with the key directory bind-mounted in, and UID /
  `0700`-mode mapping across that boundary is the sharp edge: a mismatched UID inside the
  container silently produces a `~/.ssh` the host user cannot read. This is named here so it does
  not surface as a support question later.

---

<a id="t10"></a>
## T10 — Metadata format: sentinel-prefixed TOML fragments as comments

**Date:** 2026-08-28 · **Status:** Accepted — amended by [T25](#t25)

### Context

[D7](decision-log.md#d7) elaboration 3 gives hasp a general mechanism: *"anything hasp needs to
record that the format cannot express natively is written into its own region in comment form...
metadata may record what the machine cannot tell you; it may never cache what the machine can."*
This closes [`design.md` §10](design.md#10-explicitly-deferred)'s "the metadata format" item —
load-bearing, and newly created by D7 — but the format itself was never specified.

### Decision

**Sentinel-prefixed TOML.** Inside a hasp-owned marked region, a metadata line has the exact
shape:

```text
#:hasp <key> = <toml-value>
```

Example, inside a marked region hasp owns wholesale (an explicit host group,
[D9](decision-log.md#d9)):

```text
# >>> hasp:managed >>>
Host foobarco-prod
    #:hasp created_by = "new host"
    HostName prod.foobarco.internal
    User jesse
    IdentityFile ~/.ssh/work/foobarco/id_ed25519
# <<< hasp:managed <<<
```

Stripping the `#:hasp` sentinel from a metadata line yields a syntactically valid single-line
TOML fragment (`key = value`) — a metadata reader concatenates every stripped line in a region
and feeds the result to an ordinary TOML decoder ([T7](#t7)'s dependency, reused, not
duplicated). Non-sentinel `#` comments inside a managed region (section banners hasp itself
writes, for instance) are ordinary trivia and are never parsed as data — **sentinel presence,
not comment syntax, is what marks a line as metadata**, mirroring D15's presence-not-contents
discipline for the `.hasp` marker exactly, rather than inventing a second convention beside it.

### Rationale

The D7 line is enforced by construction, not by review discipline: a fingerprint is derivable
([T1](#t1)) and therefore may never appear on the right-hand side of a `#:hasp` line — writing
one would be a D12 violation the same way it would be in the marker file D15 governs. A profile
assignment, by contrast, would be legitimate content for this channel in principle — except that
[T5](#t5) already derives host-profile membership from key location, which **removes the single
largest anticipated consumer** of this mechanism before it ever needed a value.

### Consequence

At the milestone scope this document covers (through M3), **hasp writes no field through this
channel.** That is deliberate, not an oversight: the mechanism exists because D7 requires it be
available, and its keyset is empty because T5 closed the one gap it was invented to fill. Any
future proposal to populate a field here must independently clear the same P9 bar every taxonomy
proposal must — *"show that the existing structure genuinely cannot carry the fact"* — the same
test [`design.md` §5.7](design.md#57-intent--the-category-that-closed) already stands ready to
apply.

Because the grammar is "TOML after stripping a fixed string prefix," no bespoke parser is needed
beyond a string match plus the existing TOML decoder — no new dependency, and no new grammar for
[T2](#t2)'s fuzz test to cover beyond the surrounding region structure it already fuzzes. A
future field costs one more `#:hasp key = value` line; no schema migration and no version field
are needed while the keyset is empty, and `#:hasp _schema = N` remains available, at zero present
cost, if a real one is ever needed.

**[T25](#t25) gives this line its own place in the CST** — a `MetadataLine` node, recognized by
the sentinel at parse time rather than left inside an opaque `Line` for a metadata reader to
re-scan. The keyset staying empty through M3 is unaffected; this is a representation fix, not a
new field.

---

<a id="t11"></a>
## T11 — Host-group composition: one `Include` line per group, hasp orders them

**Date:** 2026-08-28 · **Status:** Accepted

### Context

[D9](decision-log.md#d9) makes host groups real files (`~/.ssh/<group>.sshconfig`), but
`design.md` §10 leaves *"the composition mechanism behind host groups (§5.5)"* explicitly
deferred. OpenSSH's own `ssh_config(5)` semantics are **first-obtained-value-wins** per
parameter — the first matching value for a given keyword wins, later ones are ignored — which
makes the *order* group files are pulled into `~/.ssh/config` a fact with real behavioral
consequences, not a cosmetic one.

### Decision

hasp writes **one `Include` directive per host group it manages**, inside its own marked region
of `~/.ssh/config` — never a single glob (`Include ~/.ssh/*.sshconfig`) — so that **the position
of each `Include` line in the marked region is the precedence order**, controlled by hasp
directly. This is not a defense against nondeterminism: `ssh_config(5)` states that `Include`'s
wildcards *"will be expanded and processed in lexical order,"* which is deterministic. It is a
defense against the wrong *deterministic* order — lexical filename order is not necessarily the
order the user wants, and a glob would force naming discipline on group files (`0-personal`,
`1-work`) purely to win a precedence fight that explicit ordering avoids entirely.

```text
# >>> hasp:managed >>>
Include ~/.ssh/work.sshconfig
Include ~/.ssh/personal.sshconfig
# <<< hasp:managed <<<
```

### Rationale

D9's own rationale — *"SSH configuration is already multi-file in practice; the format supports
composition natively"* — is only fully honored if the composition is legible and hasp-controlled;
a glob makes the *effective* precedence order a function of how the group files happen to be
named, which is a fact the user would have to manage by naming discipline rather than one hasp
states directly (P1 — hasp should report the order, not make the user reverse-engineer it from
filenames). Because this block sits inside hasp's own marked region (D7), reordering host groups
is an ordinary `Plan`/`Change` ([T4](#t4)) — never a hand edit outside markers, keeping P2 intact
for the block's neighbors.

### Consequence

- Adding or removing a managed host group is always a `WriteRegion` change to the *same* marked
  region (append or remove one `Include` line) — never a separate edit outside markers.
- [T2](#t2)'s CST must treat `Include` as **semantically understood**, not opaque trivia, when it
  appears inside a marked region — it is part of the natively-managed directive set precisely
  because hasp needs to reason about its argument and its position, not merely preserve it.
  Outside a marked region, a human-written `Include` line round-trips like any other unrecognized
  construct — hasp reads it (to know what it must not shadow) but never rewrites it.
- `check` gains a finding category from first-obtained-value-wins directly: a later `Include`d
  group defining a `Host` pattern already fully shadowed by an earlier group or stanza is dead
  configuration — reachable by nothing — and is exactly the kind of silent bite this entry exists
  to name and catch rather than let a human discover the hard way.

---

<a id="t12"></a>
## T12 — Key identity: fingerprint when derivable, else canonical path

**Date:** 2026-08-28 · **Status:** Accepted

### Context

[`design.md` §5.1](design.md#51-key) states a key's *"intrinsic identity is its fingerprint...
two files with the same fingerprint are the same key,"* but does not say what identity **means**
in the derivation-gap case ([T1](#t1)'s third row), where no fingerprint can be computed at all.
Code has to answer this: what makes two files "the same key" when that cannot be proven?

### Decision

```go
type KeyIdentity interface{ identityKey() string }

type byFingerprint Fingerprint          // Fingerprint = ssh.FingerprintSHA256 output
func (f byFingerprint) identityKey() string { return "fp:" + string(f) }

type byPath string                      // resolved (symlink-followed), absolute path
func (p byPath) identityKey() string    { return "path:" + string(p) }
```

**Identity is the fingerprint when derivable; otherwise it is the resolved canonical path.**
Two files with the same fingerprint are one key with two locations, exactly as design.md states.
A key identified by path (the undecidable case) can **never** be deduplicated against another
path-identified key, even one that a human strongly suspects is a copy of the same secret — hasp
has no way to prove it, and P1 forbids reporting a fact hasp cannot verify.

### Rationale

[T1](#t1)'s derivation-gap table row 3 (legacy PEM, encrypted, no `.pub`) is the concrete case
this answers: seven of the thirteen keys in the design doc's own motivating inventory
([`design.md` §2](design.md#2-the-problem)) are PEM format. [D12](decision-log.md#d12) requires
this be a pure function of what is readable **this run** — no cross-scan memory of "I previously
decided these two paths are the same key" — so path-based identity being recomputed fresh every
scan, and sensitive to `mv`/symlink changes exactly as expected, is not a limitation of this rule
but a direct consequence of P1 correctly applied.

### Consequence

- `check` reports two same-format, same-size, undecidable files as a distinct finding category —
  *"possible duplicate, cannot confirm"* — never silently merged and never silently treated as
  unrelated. This sits beside, but is not the same finding as, a *confirmed* duplicate (two
  fingerprint-identified files sharing a fingerprint).
- Renaming an undecidable key's file changes its path-based identity by definition — this is a
  sharp edge worth flagging explicitly against J6's promise (*"rename a key without touching its
  material"*): rename must be implemented as a `MoveFile` change addressed by the key's stable
  `KeyName` handle, never by re-deriving identity after the move and hoping it still matches.
  `KeyName` is the stable handle across a rename; `KeyIdentity` is expected, correctly, to change
  for the `byPath` case as a direct consequence of the move — not a bug to guard against.

---

<a id="t13"></a>
## T13 — DDD adapted to Go: no Unit of Work, no message bus, split aggregate boundary

**Date:** 2026-08-28 · **Status:** Accepted

### Context

The architectural style for this project is DDD. Its most available reference material —
Cosmic Python — is Python-flavored: its patterns (a Unit of Work wrapping a database session, a
Repository pattern backed by that session, a message bus dispatching domain events to multiple
handlers, ports collected in a separate `abstractions` module) assume there is state to
transact over and more than one process or consumer in the picture. [D12](decision-log.md#d12)
(no persisted state, hasp is a pure function of the machine) and P8 (one laptop, one process, one
invocation) both remove exactly the conditions those patterns exist to serve.

### Decision

Four layers, dependency direction strictly inward:

```text
internal/domain     Key, Host, Profile, HostGroup, Binding, value objects
                     (Fingerprint, ProfilePath, KeyName, HostPattern) — zero third-party imports
internal/app         one use case per verb×noun cell; owns the Plan type (T4);
                     declares ports as interfaces at the point of consumption
internal/adapter     scan, keyfile (T1), sshconfig (T2), settings (T7), backup (T8), fswrite (T15)
internal/cli         cobra wiring (T3), flags, human + JSON renderers (T14)
```

**Two deliberate deviations from Cosmic Python, each justified against a specific principle:**

1. **No Unit of Work, no repository-with-a-session.** Nothing is persisted (D12), so there is no
   transaction to own. `Applier` ([T4](#t4)) is not a UoW in disguise — it has no rollback across
   multiple aggregates on partial failure, only a per-`Change` backup-then-write guarantee.
2. **No message bus, no domain events.** P8: *"a single human operating interactively"* — there
   is no second consumer for an event to reach. A cross-cutting effect (adopting a key must also
   ensure a top-level alias exists, per D13) is an ordinary function call inside the use case
   building the `Plan`, not a published event with a handler registered elsewhere.

**Ports are declared at the point of consumption**, in `internal/app`, per-use-case — the Go
idiom of "accept interfaces, return structs" — rather than collected in a separate abstractions
module the way Cosmic Python centralizes them.

**The aggregate boundary differs by direction, deliberately:**

- **Reads use one read-model aggregate** — a single derived snapshot of "the machine," covering
  keys, hosts, profiles, and bindings together, computed once per invocation. Justified by P8
  (cheap at this scale) and *required* by P7: answering "which hosts use this key" in one screen
  needs the join computed once, not stitched together in the CLI layer from several independent
  repository calls.
- **Writes are file-scoped** — one aggregate boundary per file being changed. The file is hasp's
  actual unit of atomic replacement ([T15](#t15)) and of backup ([T8](#t8)); two files changed in
  one `Plan` are two independent `Change` entries, each independently backed up. There is no
  cross-file transaction, because POSIX offers no way to rename two files atomically as one
  operation — stated plainly here rather than implied away.

### Rationale

Every deviation above traces to a specific principle already in force (D12, P8, P7) rather than
to "DDD is inconvenient here" — the whole discipline of this project is that an argument gets
settled by citation, and an architectural deviation is exactly the kind of claim that needs one.

### Consequence

- A use case touching two files (for example, `edit host --group` moving a stanza between host
  groups) produces a `Plan` with two `Change` entries and **no atomicity guarantee stronger than
  "each individual file write is atomic"** ([T15](#t15)). If the second write fails after the
  first succeeds, the result is a `check`-detectable partial application — the stanza present in
  both groups, or in neither — and `check` must carry a finding for exactly this shape. This is
  documented here so a future contributor does not assume `Plan` grants all-or-nothing atomicity
  across files when it only grants it per file.
- A guard test enforces `internal/domain` has zero third-party imports — the domain layer is pure
  Go standard library plus its own types, checkable mechanically (`go list -deps` on the package,
  asserted in CI-equivalent tooling), not merely a convention someone has to remember.

---

<a id="t14"></a>
## T14 — Output contract: 4 exit codes, a versioned JSON envelope, silent stdout

**Date:** 2026-08-28 · **Status:** Accepted

### Context

[`design.md` §6.3](design.md#63-cross-cutting-requirements) requires *"every read has a
machine-readable form"* — `design.md` does not use the word "contract," but this is what a
requirement worded that way means in practice: `hasp list key --json | jq` must keep working
release over release, not merely work once. `check` ([§6.2](design.md#62-the-verbs)) is
explicitly advisory and must be usable in a pipeline, which means "found issues" and "failed to
run" have to be distinguishable by exit code alone.

### Decision

Four exit codes, and only four:

| Code | Meaning | Who can return it |
| --- | --- | --- |
| `0` | Clean — no findings, or a write completed | Any command |
| `1` | Findings — `check` surfaced issues; not a hasp failure | `check`, exclusively |
| `2` | Usage — bad flags, bad arguments, or an unresolvable non-interactive precondition ([T6](#t6)'s fail-closed case) | Any command |
| `3` | Error — hasp could not complete the requested operation (I/O failure, a fail-closed guard tripped mid-write, backup failed) | Any command |

Cobra itself never calls `os.Exit`; `cmd/hasp`'s `main()` maps the error returned from
`rootCmd.Execute()` to one of the four codes via a small sentinel-error taxonomy
(`errors.Is(err, app.ErrFindings)`, `errors.Is(err, app.ErrUsage)`, else `3`).

`log/slog` writes to stderr, **off by default**, enabled only by `--verbose` — stdout carries
data and nothing else, which is what makes the pipeline promise literal rather than aspirational.

JSON output is wrapped in a versioned envelope:

```go
type Envelope struct {
    Version  int             `json:"version"`  // envelope schema version — independent of hasp's own release version
    Kind     string          `json:"kind"`      // "key.list", "host.show", "check.report", ...
    Data     json.RawMessage `json:"data"`
    Warnings []string        `json:"warnings,omitempty"`
}
```

### Rationale

P7 (*"answer the question in one screen"*) governs the human renderer; §6.3's machine-readable-form
requirement governs the JSON one — the two are explicitly allowed to diverge in **form** but must
never diverge in **content**, which [T13](#t13)'s single read-model aggregate guarantees by
construction: there is exactly one code path that computes an answer, and two that display it.

A versioned envelope from the first release, not retrofitted after the first breaking change, is
what makes the machine-readable form a contract in practice rather than an accident of the
current implementation — a consumer piping `hasp ... --json` through `jq` can branch on
`.version` whenever it needs to, rather than only after hasp breaks it without warning.

### Consequence

- `check` is the **only** use case permitted to resolve to exit code `1` — no other verb ever
  does, which keeps the meaning of "1" stable across the entire surface: a scripted
  `if hasp check ...; then` reads unambiguously, forever.
- `--verbose`'s diagnostic stream and `--json`'s data stream can never collide, because they are
  different file descriptors by construction, not by convention that a future change could erode.

---

<a id="t15"></a>
## T15 — Safety mechanics: atomic write, symlink-through, mode preservation, D4's move rule

**Date:** 2026-08-28 · **Status:** Accepted — refined by [T20](#t20), [T22](#t22)

### Context

P4 (every write reversible) and [D4](decision-log.md#d4) (*"hasp has no operation that can
destroy an irreplaceable secret... this is canon"*) both need a concrete write-time mechanism, not
just a policy. Four distinct sharp edges live here, and each has a specific way to get it wrong.

### Decision

**1. Atomic write.** Write to a temporary file in the **same directory** as the target (so the
final replace is same-filesystem and therefore eligible to be atomic on POSIX), `fsync` the temp
file, then `os.Rename` (`func Rename(oldpath, newpath string) error` — Go documents only that
*"if newpath already exists and is not a directory, Rename replaces it"*; the atomicity comes
from POSIX `rename(2)` on the platforms hasp targets, not from Go, which notes rename is *"not an
atomic operation"* on non-Unix platforms) the temp file onto the target, then `fsync` the
containing directory. The directory `fsync` is easy to skip and looks correct until a crash —
without it, the rename itself is not guaranteed durable on Linux.

**2. A symlinked `~/.ssh/config` is resolved and written through — the symlink itself is never
replaced.** Detected via `os.Lstat` distinguishing a symlink from a regular file, with
`filepath.EvalSymlinks` resolving the real target; the atomic-write dance in point 1 runs against
the **resolved** target's directory, not `~/.ssh`. Named explicitly because dotfiles repositories
make this layout common (`~/.ssh/config -> ~/dotfiles/ssh/config`), and a tool that transparently
removes and recreates the symlink as a plain file silently converts the user's dotfiles-managed
config into an orphaned copy — destructive by omission, not by design, which is exactly the class
of bug D4's spirit exists to prevent even though D4's letter is about key material specifically.

**3. Mode and ownership are preserved across a write.** `os.Stat` the original file before
writing, then apply the same mode (and ownership, where the process has permission to) to the
temp file before the rename in point 1 — so a `0600` config file does not silently become
umask-determined (commonly `0644`) after hasp's first touch.

**4. D4's move rule for key material: copy → verify → unlink source. Never unlink first.**
Verification is fingerprint re-derivation and comparison when the key is fingerprint-derivable
([T1](#t1)/[T12](#t12)); a byte-for-byte comparison for the undecidable case. This is the
mechanism behind `edit key --profile`, a plain relocation with no alias required — no failure
path between the copy and the verified unlink leaves zero readable copies of the secret.

**Refined for `adopt`, which must also leave an alias behind — see [T20](#t20).** `new key` and
`edit key --replace-material` are covered by the same atomic-write discipline (point 1) for a
different reason — they write to a path that may already hold a key — and are refined
in [T22](#t22).

### Rationale

Each point traces to a named principle: 1 and 3 to P4 (a reversible write is only meaningful if
the write itself cannot corrupt the file mid-flight, or silently loosen its permissions); 2 to P6
(nothing hasp does should require the user to have anticipated hasp's own implementation detail —
a dotfiles-managed symlink is the user's own arrangement, and hasp is a guest in it); 4 is D4
applied literally, at the level of the exact sequence of syscalls, not merely as a policy
statement.

### Consequence

- The guard test for point 4 injects a write failure partway through `adopt` (for example, a
  read-only destination directory) and asserts the **original** file still exists afterward — a
  test that fails loudly if a future refactor ever reorders copy/verify/unlink.
- The guard test for point 2 constructs a symlinked `~/.ssh/config` fixture and asserts, after a
  write, that the symlink's **target inode** is unchanged — not merely that the file's content is
  correct. Content-correct-but-inode-replaced is exactly the silent-breakage failure mode this
  entry exists to catch, and a content-only assertion would not catch it.
- These four mechanics are the concrete implementation behind every `RequiresBackup() == true`
  `Change.Apply` in [T4](#t4)'s `Applier` — this entry is where "how" lives; T4 is where "when."

---

<a id="t16"></a>
## T16 — Implicit default-identity probing is a distinct, labelled binding kind

**Date:** 2026-08-28 · **Status:** Accepted — amends [T5](#t5)

### Context

[T5](#t5) derives a host's profile membership from the keys its `IdentityFile` lines resolve to.
That rule is correct and stays — but as first written it modelled only **explicit** `IdentityFile`
directives, and `ssh` does not require one. `ssh_config(5)` states the fallback plainly: *"The
default is ~/.ssh/id_rsa, ~/.ssh/id_ecdsa, ~/.ssh/id_ecdsa_sk, ~/.ssh/id_ed25519,
~/.ssh/id_ed25519_sk and ~/.ssh/id_mldsa44_ed25519."*

A stanza with no `IdentityFile` is not an exotic case; it is the ordinary shape of a hand-written
config, which is precisely the artifact [J1](design.md#7-journeys) points hasp at. The omission
was self-contradictory as well as wrong: `adopt`'s own rationale ([D13](decision-log.md#d13))
turns on leaving a top-level alias *because* `ssh` probes those names, so the write path already
depended on a behaviour the read path did not model.

### Decision

Model **two binding kinds**, carried as `Binding.Kind` through `internal/domain` and into the JSON
envelope: `Explicit` (an `IdentityFile` directive resolved per §5) and `ImplicitDefault` (a member
of the default set above that exists in the key directory).

Exactly one thing suppresses the implicit set: **`IdentityFile none`**. `IdentitiesOnly yes` does
**not** — it constrains which identities an *agent* may offer beyond *"the configured
authentication identity and certificate files (either the default files, or those explicitly
configured...)"*, and the default files are named there as still in scope.

### Rationale

Merging the two kinds into one undifferentiated set would answer [J8](design.md#7-journeys)
correctly but report it dishonestly — "you wrote this binding" and "`ssh` would fall back to this
binding" are different facts about the machine, and P1's whole claim is that hasp reports what is
actually there. Keeping them labelled costs one enum field and preserves the distinction for any
consumer that needs it.

Not modelling implicit bindings at all was the alternative, and it fails twice over: `check`'s
"host bound to no key" finding would fire on nearly every stanza in an uncurated `~/.ssh` —
turning J1's first impression into a false-positive flood — and J8 would silently omit hosts that
genuinely depend on a departing profile, which is the one journey where an omission is expensive.

### Consequence

- `check`'s **"host bound to no key"** finding fires only when *neither* kind resolves to
  anything. A stanza relying on default probing is correctly reported as bound.
- A key at the top level of the key directory can now confer profile membership on a host without
  any directive naming it. This is correct — it is what `ssh` actually does — but it means
  `adopt`'s alias symlink is load-bearing for *reporting*, not only for connectivity.
- The default list is version-sensitive: OpenSSH has both added (`id_mldsa44_ed25519`) and removed
  (`id_dsa`) entries over time. It is recorded here as read from `ssh_config(5)` on the
  development host and must be re-checked against the man page rather than from memory.

---

<a id="t17"></a>
## T17 — `Directive` preserves its exact separator and spacing; quote-aware tokenizing

**Date:** 2026-08-28 · **Status:** Accepted — amends [T2](#t2)

### Context

[T2](#t2) commits to a lossless CST whose contract is `Render(Parse(b)) == b`. The first
`Directive` sketch was `{ Keyword string; Args []string; Trivia []byte }` — which cannot satisfy
that contract. `ssh_config(5)`: *"Configuration directives are separated from their values by
whitespace or exactly one '=' character (which may be surrounded by whitespace)."* So `Port 22`,
`Port = 22` and `Port=22` are semantically identical and byte-different, and a node holding only
`Keyword` and `Args` has nowhere to record which was written.

Because recognized keywords parse into `Directive` even inside ordinary hand-written stanzas —
not only inside hasp's own regions — reconstruction would normalize `Port=22` to `Port 22` and
break [P2](design.md#4-principles) on the first real config using that style. A guarantee that
fails on a legal spelling of the most common directive form is not a guarantee.

### Decision

`Directive` retains the raw bytes of everything between and around its tokens: the
keyword-to-value separator verbatim, inter-argument spacing, leading indentation, and any trailing
comment. `Keyword` and `Args` are **derived views** for hasp's own logic; rendering emits the
original bytes unless the directive sits inside a hasp-owned region, where [D7](decision-log.md#d7)
licenses rigid regeneration.

Tokenizing is **quote-aware**. `ssh_config(5)`: *"'#' outside of a quoted string may be used to
add a comment to the end of a line"* — so a `#` inside a double-quoted value is a value byte, not
a comment boundary, and `Values may optionally be enclosed in double quotes (")`.

### Rationale

The alternative — keep byte-exact `Line` nodes everywhere outside a marked region and parse into
`Directive` only inside regions hasp owns — also satisfies P2, and is simpler. It was rejected
because hasp must *read* semantics from directives it does not own: [T16](#t16)'s binding
resolution reads `IdentityFile` out of stanzas hasp will never write. Parsing everywhere and
rendering from retained bytes gives both properties at once; parsing only inside regions would
force a second, parallel read-only parser for the outside case.

### Consequence

- `Render` is not "reconstruct from fields" but "emit retained bytes, except where regenerating."
  A future contributor who adds a field to `Directive` and renders from it reintroduces exactly
  the bug this entry exists to prevent — the fuzz test in §12 is what catches them.
- Quote-awareness means the comment scanner is a small state machine, not a `strings.IndexByte`.
  A `ProxyCommand` value containing `#` inside quotes is the fixture that proves it.

---

<a id="t18"></a>
## T18 — CST marker defects are represented as data, not parse errors

**Date:** 2026-08-28 · **Status:** Accepted — amends [T2](#t2)

### Context

[T2](#t2) states that parsing never fails — every byte becomes some node, which is what makes
`Render(Parse(b)) == b` total. §11's safety table then posits *"a marker or region is malformed"*
as a fail-closed guard for writes. Both cannot be true as originally written: if parsing never
errors, nothing carries the signal the guard keys off.

### Decision

Marker defects are **data on the parsed file**, not errors from parsing. `File.MarkerDefects
[]MarkerDefect` is populated during the parse, enumerating the detectable conditions: an
unmatched begin, an unmatched end, a duplicate begin for the same region id, an end preceding its
begin, and a marker nested inside a `HostBlock`.

A non-empty `MarkerDefects` makes writes **to that file** fail closed. Reads are unaffected and
report what parsed, per §11's fail-open rule for reads and [P5](design.md#4-principles)'s
guarantee that reads are always safe.

### Rationale

Representing defects as data preserves T2's totality — the contract that a file hasp does not
understand still round-trips is what [P2](design.md#4-principles) rests on, and an error return
would break it for exactly the damaged files most in need of being left alone.

It also puts the two stances in the right places. A damaged marker means hasp cannot know where
its own territory ends, so writing would risk destroying human bytes — fail closed. But refusing
to *report* on the file would break [J1](design.md#7-journeys), whose promise is that surveying
any machine works. The asymmetry is deliberate and follows the shape §11 already uses everywhere.

### Consequence

- `check` gains a finding for each `MarkerDefect` kind — a damaged region is exactly the sort of
  untidiness [J5](design.md#7-journeys) exists to surface, and the user repairs it with an editor
  ([P6](design.md#4-principles)), since hasp will not write to that file until it is sound.
- A hand-mangled region is recoverable rather than fatal: because hasp owns the region wholesale
  ([D7](decision-log.md#d7)), the repair is to rewrite it from scratch once the markers are
  balanced again.

---

<a id="t19"></a>
## T19 — Alias location is the mechanism for cross-profile key membership — D2's write path

**Date:** 2026-08-28 · **Status:** Accepted

### Context

[D2](decision-log.md#d2) states a key may belong to more than one profile, and calls a key shared
between `personal` and `work.foobarco` *"a real situation, not a modelling error."*
[D13](decision-log.md#d13) then made a profile a **directory** and membership a fact of location.
A file lives in exactly one directory. Nothing in the design says how a user is supposed to
*create* the state D2 declares legitimate.

`--add-alias` existed, but framed purely as a naming convenience for tools that hardcode `id_rsa`
([D5](decision-log.md#d5), §5.2) — never as the mechanism that realizes D2.

### Decision

**An alias symlink placed inside a second profile's directory is what makes a key a member of
that profile.** Membership derivation unions over **all** of a key's locations — every path at
which its material is reachable, canonical or symlinked — not only the canonical one.

`edit key --add-alias=<path>` therefore does double duty: it is both D5's naming escape hatch and
D2's write path, depending on where the alias is placed. §5 and §9 say so explicitly rather than
leaving the user to infer it.

### Rationale

This needed no new mechanism, which is the point. [P9](design.md#4-principles) says taxonomy rides
on structures that already exist, and the symlink was already ratified as the materialization of a
name (D5). Reading membership from *all* locations rather than one is a one-line change to the
derivation and keeps [D12](decision-log.md#d12) literally true — nothing is declared, and the
second membership is as visible to `ls` as the first.

The alternative was a metadata field recording extra profiles, which [D7](decision-log.md#d7)
would have permitted. It was rejected on P9's ladder: the filesystem can carry this fact, so the
metadata channel must not.

### Consequence

- A key's `Locations` is a set, and its `Profiles` is the union over that set. Any code path that
  reasons from "the" location of a key is wrong by construction.
- `release` on one profile must not remove material still reachable from another. The safe form
  is removing the alias in the released profile, never the canonical file —
  [D4](decision-log.md#d4) is unaffected because no operation removes the last copy.
- [J8](design.md#7-journeys) inherits D2's hard case exactly as [D2](decision-log.md#d2) predicted:
  offboarding `work.foobarco` must report a key also reachable from `personal` as *kept, not
  revoked*, and `check` must be able to tell the two apart.

---

<a id="t20"></a>
## T20 — Plan ordering: any prefix leaves a working state; `adopt`'s alias-preserving move

**Date:** 2026-08-28 · **Status:** Accepted — amends [T4](#t4), [T15](#t15); amended by
[T43](#t43)

### Context

[T13](#t13) accepts that a `Plan` touching several files has no cross-file atomicity, because
POSIX offers no way to rename two files as one operation, and routes the resulting inconsistency
into a `check` finding. That is honest, but incomplete: `Applier` walks `Changes` in slice order
and nothing said how the slice should be **built**.

Ordering is not cosmetic here. `adopt key` naturally plans `[MoveFile, CreateSymlink]`. If the
move succeeds and the symlink does not, the key sits in its profile directory with no top-level
alias — and `ssh` stops finding it. That is [D13](decision-log.md#d13)'s stated consequence
arriving as a silent failure: the user's next push fails with nothing pointing at hasp.

### Decision

**A `Plan`'s `Changes` are ordered so that any prefix of them leaves the machine in a working
state.** The least recoverable change goes last.

Applied to `adopt key`, this inverts the natural order to `[CreateSymlink, MoveFile]`: the symlink
is created first, pointing at a destination that does not yet exist. A dangling symlink is inert —
`ssh` simply does not find it, which is the pre-adopt status quo — and the subsequent move makes
it valid.

### Rationale

This converts an unavoidable weakness into a bounded one. Cross-file atomicity is not available
(T13), so the remaining lever is *which* half-applied states are reachable. Ordering by
recoverability means every reachable partial state is either the status quo or an improvement on
it, and the worst outcome is a stray artifact `check` can name rather than a silently broken
authentication path.

It also composes with [P4](design.md#4-principles) rather than duplicating it. Backups make a
partial application *recoverable*; ordering makes it *non-damaging in the first place*. The second
is worth more, because it does not require the user to notice anything went wrong.

### Consequence

- Ordering is a property of each use case's `Plan` construction, and it needs a test per
  write verb: inject a failure at each index and assert the machine still works.
  [T15](#t15)'s injected-failure guard test for `adopt` is the template.
- A dangling symlink is now a legitimate transient state, so `check`'s dangling-alias finding must
  distinguish "left over from a failed `adopt`" from "the target was deleted" — or at minimum
  report it in terms that make the remedy obvious.
- Any future `Change` kind arrives with an ordering obligation, not just an `Apply`. The rule is
  stated in §4 so it is visible where plans are built, not only here.

---

<a id="t21"></a>
## T21 — `show profile` aggregates descendants by default, `--no-recurse` escape

**Date:** 2026-08-28 · **Status:** Accepted

### Context

[D1](decision-log.md#d1) makes profile names hierarchical paths — `work.foobarco` is a profile and
`work` is its parent and also a profile — and explicitly retires "profile group" in favour of "a
profile with children." `show profile` originally listed child profiles as one more field
alongside keys and hosts, which reads as enumeration rather than aggregation.

That is too weak for [J8](design.md#7-journeys), whose whole framing is *"show me everything
scoped to that profile — every key, every host that depends on it — so I know what to revoke and
what will break."* Offboarding `work` while `work.foobarco` and `work.acme` hold the actual keys
would answer with an almost-empty screen, and the user would have to walk each child and union the
results by hand — precisely what [P7](design.md#4-principles) says the tool exists to prevent.

### Decision

`show profile <name>` **aggregates the full subtree by default**: keys and hosts from the named
profile and every descendant, with each row attributed to the profile it actually came from.
`--no-recurse` restricts the view to the named profile alone.

### Rationale

The default follows P7 directly. The question a human asks about `work` on their last day is
about everything under `work`, and a tool that answers a narrower question than the one asked has
failed at comprehension even if every fact it prints is true.

Attribution-per-row is what keeps this honest under [D2](decision-log.md#d2) and
[T19](#t19): a key reachable from both `work.foobarco` and `personal` appears in the `work` subtree
*and* is visibly also `personal`, which is the distinction that decides whether it gets revoked.
Aggregating without attribution would produce a correct list that supports an incorrect decision.

### Consequence

- `--no-recurse` exists because "what is *directly* in this profile" is a real question when
  deciding whether a parent is a profile in its own right or merely a container
  ([D13](decision-log.md#d13)'s `.hasp` distinction).
- The JSON envelope carries the owning profile on every aggregated row, so a machine consumer can
  regroup without re-querying.
- A container directory with no `.hasp` marker is not a profile and cannot be the *subject* of
  `show profile`, but its descendants still aggregate under the nearest marked ancestor.

---

<a id="t22"></a>
## T22 — `WriteKeyFile` fails closed on an existing target; a `Remove` change kind

**Date:** 2026-08-28 · **Status:** Accepted — amends [T4](#t4), [T15](#t15); amended by
[T44](#t44)

### Context

Two gaps in [T4](#t4)'s `Change` set, both found by verification against
[D4](decision-log.md#d4).

First: `WriteKeyFile` is the sole operation that authors private key bytes, and [T15](#t15)'s
atomic-write mechanic ends in `os.Rename`, which Go documents as: *"If newpath already exists
and is not a directory, Rename replaces it."*
No pre-write check said the target must be empty. `hasp new key --name id_ed25519_foobarco` against
a path that already holds a key — a re-run, a typo, a reused name — would silently replace an
irreplaceable secret. D4 is canon: *"hasp has no operation that can destroy an irreplaceable
secret."* An unguarded rename is one.

Second: `release profile` must remove a `.hasp` marker, and no `Change` kind could delete
anything.

### Decision

**`WriteKeyFile` fails closed if its target path exists.** Refused before any byte is written,
with the existing path named in the error. There is no `--force`: an overwrite flag on the one
operation that authors secrets is exactly the affordance D4 forbids, and the user who genuinely
wants the path free has `rm`, which is their own act with their own tools (D4's own reasoning).

**A `Remove{ Path, Reason }` change kind** is added, restricted to artifacts hasp authored — a
`.hasp` marker, an alias symlink. It never applies to a key file. Per
[P4](design.md#4-principles) it backs up the removed file first, and per
[D15](decision-log.md#d15) `release`'s preview shows a marker's contents before removal, so a
marker the user has written notes into is never a silent loss.

### Rationale

Failing closed costs a user who reuses a name one clear error message. Failing open costs a user
one key, permanently, with no backup to restore from — `WriteKeyFile` is a *create*, so there is
no prior version in the backup store to fall back on. The asymmetry is total and the choice is
not close.

`Remove` is deliberately narrow. Giving `Change` a general delete primitive would put the
mechanism for destroying key material into the codebase and rely on call sites never reaching for
it — the opposite of how [T1](#t1)'s no-decrypt boundary is enforced. Restricting it to
hasp-authored artifacts keeps D4 structural rather than disciplinary.

### Consequence

- §11's failure-stance table carries the row explicitly, and §9's `new` grid cell states the
  refusal, so an implementer meets it in both places.
- `edit key --replace-material` deliberately does **not** route through `WriteKeyFile` — replacing
  material means the target exists by definition. It uses [T15](#t15)'s copy → verify → unlink
  move rule against the incoming file, with the outgoing material backed up first.
- `Remove`'s restriction is a type-level obligation, not a comment: the constructor takes the
  artifact kind, and no code path constructs one for a key file.

---

<a id="t23"></a>
## T23 — DSA is in scope for the read path; fixtures are PEM-only, hand-constructed

**Date:** 2026-08-28 · **Status:** Accepted — extends [T1](#t1)

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

[design.md §2](design.md#2-the-problem) describes the actual artifact this tool exists for: a
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
  like any other key, since none of those operations touch its bytes ([D14](decision-log.md#d14)).

---

<a id="t24"></a>
## T24 — Line-terminator handling in the CST: preserved as trivia, excluded from `Args`

**Date:** 2026-08-28 · **Status:** Accepted — amends [T2](#t2)

### Context

[T2](#t2)'s contract is `Render(Parse(b)) == b`, fuzzed. That proves byte fidelity and nothing
else — and byte fidelity is not the property that breaks first.

A CRLF-terminated config parsed by a tokenizer that splits on space and tab would carry the
trailing `\r` into the last argument. `IdentityFile ~/.ssh/work/id_ed25519\r` renders back
byte-identically — the fuzz test passes — while `filepath.EvalSymlinks` fails on the path, and
[T16](#t16)'s binding resolution reports a dangling target for a key that is sitting right there.
A silent semantic corruption that passes the stated correctness gate.

### Decision

The line terminator is **trivia on the node** — `\n`, `\r\n`, or absent at EOF — preserved for
rendering and **excluded from `Args`**. §12 adds a **semantic** assertion alongside the byte
identity one: parsing a CRLF fixture must yield `Args` identical to parsing its LF twin.

Shape fixtures are committed for the cases that break naive tokenizers: CRLF throughout, mixed
terminators in one file, no trailing newline at EOF, and a zero-byte file.

### Rationale

The general lesson is worth stating once, here, because it applies to every future CST change: a
round-trip test proves you did not *lose* information, never that you *interpreted* it correctly.
The two properties need two tests. Dotfiles repositories shared across machines make CRLF a
realistic input rather than a hypothetical, and this document already acknowledges that workflow
in [T15](#t15)'s symlinked-config case.

### Consequence

- A zero-byte config is a valid parse producing an empty file with no nodes — not an error, and
  not a missing file. `hasp` must render an empty region into it correctly on first write.
- Rendering never *normalizes* terminators, including inside hasp-owned regions: a region written
  into a CRLF file matches the file it lives in. Rigid regeneration ([D7](decision-log.md#d7))
  governs a region's content, not the physical line endings of the file carrying it.

---

<a id="t25"></a>
## T25 — `MetadataLine` is its own CST node type for `#:hasp` lines

**Date:** 2026-08-28 · **Status:** Accepted — amends [T2](#t2), [T10](#t10)

### Context

[T10](#t10) defines the metadata channel as `#:hasp <key> = <toml-value>` comment lines inside a
hasp-owned region. The CST had no node for them, which implied reading them by re-scanning raw
`Line` bytes for a string prefix. That is workable for reading and wrong for writing: rigid
regeneration ([D7](decision-log.md#d7)) rebuilds a region from structure, and a metadata line that
exists only as an unrecognized `Line` has no place to be *put back* — no way to say which
`HostBlock` it belongs under, or that it must precede the directives it annotates.

### Decision

`MetadataLine{ Key string; Value string; Trivia []byte }` is a node type, recognized by the
`#:hasp` sentinel (including its trailing space), valid **only inside a hasp-owned region**. Outside one it is an ordinary
comment `Line` and is never interpreted — the sentinel confers no meaning on territory hasp does
not own.

### Rationale

Giving it a node type now, while [T10](#t10)'s keyset is still empty, costs one small type and
removes the need to retrofit the CST at the moment a real field first lands — which is exactly
when the pressure to cut a corner will be highest.

The inside-a-region-only restriction is [P2](design.md#4-principles) applied literally: a `#:hasp`
line a human typed into their own config is their comment, and hasp reading it as data would be
hasp claiming territory by string match. The marker defines ownership
([D7](decision-log.md#d7)); the sentinel only labels a line within it.

### Consequence

- [T10](#t10)'s "the keyset is empty through M3" is unchanged. This entry gives the mechanism a
  place to live; it does not put anything in it, and the P9 bar for adding a field stands.
- Round-tripping is unaffected: a `MetadataLine` renders from its retained bytes like every other
  node ([T17](#t17)), so the fuzz contract holds whether or not the sentinel is present.

---

<a id="t26"></a>
## T26 — `WriteRegion` previews carry a real diff; key material is never diffed

**Date:** 2026-08-28 · **Status:** Accepted — amends [T4](#t4)

### Context

[T4](#t4) gave `Change` a `Describe() string` returning *"one-line preview text."* That cannot
serve [D6](decision-log.md#d6) for the change kind that most needs it. `WriteRegion` regenerates
a whole region body — reordered `Include` lines, host stanzas, comments — and a one-line summary
("rewrite host group `personal.sshconfig`") tells the user nothing about whether a stanza moved or
vanished.

[P5](design.md#4-principles) requires an operation to *"describe exactly what it would change"*,
and calls previewability a constraint on how change is modelled rather than a flag. A `Change`
that carries only the new `Body`, with no access to the current bytes, structurally cannot
produce that description.

### Decision

`Change` returns a `Preview{ Summary string; Diff []DiffLine }`. `WriteRegion` computes a real
line diff of the region's current bytes against its regenerated body. Kinds with nothing to
compare — `MoveFile`, `CreateSymlink`, `Remove` — return `Diff == nil` and render as a summary
line only.

**`WriteKeyFile` returns `Diff == nil` unconditionally**, even though it has bytes and they are
new. Private key material is never rendered, to any renderer, in any mode.

### Rationale

D6 buys its safety from the user *reading* the preview. A preview that summarizes rather than
shows moves the burden of noticing a mistake back onto the person with the least information at
the moment they confirm — the same reasoning D6 itself used to reject opt-in previewing.

The `WriteKeyFile` exception is not a gap in that argument but an application of a stronger rule.
[P3](design.md#4-principles) keeps hasp out of custody of secrets; printing a freshly generated
private key to a terminal, a pipe, or a JSON consumer would put it into scrollback, logs, and
shell history in one step. The user does not need to see the bytes to judge the operation — the
summary names the path, algorithm and passphrase mode, which is the whole decision.

### Consequence

- `Diff` is a structured value, not preformatted text, so the human renderer can colorize it and
  the JSON renderer can marshal it — the same `Preview` serves both, and §10's envelope omits
  `diff` entirely rather than emitting an empty array when there is nothing to show.
- Computing the diff means `Plan()` reads the target file's current bytes. Reads are always safe
  ([P5](design.md#4-principles)), so this costs nothing in guarantees — but it does mean a `Plan`
  is computed against a snapshot, and a file changed between preview and confirm is a race the
  `Applier` must detect rather than assume away.
- A guard test asserts no renderer path can reach `WriteKeyFile`'s contents: the field is not
  exported into `Preview` at all, so the boundary is enforced by absence, as in [T1](#t1).

---

<a id="t27"></a>
## T27 — Go, and a fresh implementation rather than a repair of the predecessor

**Date:** 2026-08-29 · **Status:** Accepted
**Closes:** [`design.md` §10](design.md#10-explicitly-deferred)'s *implementation stack* item

### Context

`design.md` §10 deferred the stack completely. Before this entry closed it, the item read:
*"**Implementation stack** — language, runtime, distribution, dependencies. Nothing in this
document assumes any of them."* [`docs/tdd.md`](tdd.md) §2 then opens with **Go 1.27.1** as a
given.

No entry in this log recorded how it got there. That is a gap of exactly the kind this project's
culture exists to catch: the largest technical decision in the project — larger than any of
`T1`–`T26`, every one of which presupposes it — carried no citation at all. Two questions were
actually decided, and neither was written down: **whether to repair the Python implementation or
replace it**, and **what to write the replacement in**.

### The prior question: repair or replace

[`docs/project-assessment-2026-08.md`](project-assessment-2026-08.md) §6 proposed a five-phase
repair roadmap for the existing Python codebase. It was not taken — and *not* because repair
looked hard. The assessment says the opposite: *"the blockers are small and well-localised"* and
*"A focused day gets you back to a working single-store tool."*

It was not taken because **[D12](decision-log.md#d12) deleted most of the work the repair
consisted of.** `project-assessment-2026-08.md` §6's Phase 1 is *"Decide the store, then converge
on it."* Phase 2 is
*"Finish `sync key`"*, labelled *"this is the MVP."* D12 answers Phase 1 with *neither store*,
and dissolves Phase 2 outright in its own words: *"**`sync` dissolves.** The verb exists only to
reconcile an index against reality. With no index there is nothing to reconcile."*

So the repair's two largest phases had no work left in them once the design was ratified. What
survived from the predecessor was its **ideas**, not its code, and the assessment names which:
the verb×noun matrix — *"the good architectural idea in this codebase"*, with the instruction
*"Keep this."* — and the eleven concrete scenarios in `keys.feature`.

What remained to write was: state handling (deleted by D12), the key read path ([T1](#t1)), the
config parser ([T2](#t2) — which never existed in the predecessor; the assessment's Phase 5 lists
it as an unstarted spike), and the command surface ([D10](decision-log.md#d10),
[T3](#t3)). That is the entire program. **A rewrite was not chosen over a repair — the repair had
almost nothing left in it.**

### The language

Given a fresh implementation, four constraints actually discriminated:

1. **The derivation gap must close with no subprocess and no cgo.** Every read recomputes every
   fact (D12), on every invocation. The predecessor shelled out to `ssh-keygen` and linked
   `libmagic`; the assessment §4 counts both as toolchain weight, and
   [D14](decision-log.md#d14) records `ssh-keygen -c` failing outright on encrypted keys.
   `x/crypto/ssh` reads the whole key matrix from raw bytes in pure Go ([T1](#t1)) — the specific
   capability that made `CGO_ENABLED=0` reachable ([T9](#t9)) and removed that failure class.
2. **Distribution as a single static binary.** GoReleaser is a stated constraint on this project
   ([T9](#t9)). A tool whose job is to inspect `~/.ssh` on a machine you have just sat down at
   should not first require a runtime, a virtualenv, or a package manager.
3. **A parser hasp owns either way.** [T2](#t2) found no `ssh_config` library in any ecosystem
   offering the round-trip contract P2 requires — and the assessment's Phase 5 had already reached
   the same conclusion for Python, listing `storm`, `advanced-ssh-config`, `paramiko.SSHConfig`
   and `sshconf` as *"a genuine build-vs-buy decision and deserves its own spike."* Since the
   parser is hand-rolled in either language, the ecosystem's parser inventory stopped being a
   reason to prefer one.
4. **Static typing over a domain of small value objects.** `tdd.md` §3's domain is almost entirely
   immutable values, and its one genuinely tricky rule — `KeyIdentity` being a fingerprint *or* a
   path ([T12](#t12)) — is expressible as a sealed interface the compiler checks rather than a
   convention a reviewer enforces.

### Decision

**Go, and a fresh implementation in a new repository.** The Python codebase is not migrated, not
vendored, and not referenced by the new code. It survives as the subject of
`docs/project-assessment-2026-08.md`: a record of what was learned, including the specific failure
this entire documentation set exists to avoid repeating.

Go 1.27.1 via `mise`; targets `linux/{amd64,arm64}` and `darwin/{amd64,arm64}` ([T9](#t9)).

### Rationale

The three points above are the case for Go. The honest counterweight is smaller than expected.

**What was given up.** The `keys.feature` scenarios were executable in principle — `pytest-bdd`
was a declared dependency — but the assessment records *"zero step definitions and zero Python
test files"*, so nothing executable was actually lost. The eleven scenarios are carried forward
as journeys (`design.md` §7) and as `testscript` cases (`tdd.md` §12). Python's ecosystem
advantage for this specific domain proved narrow: `paramiko` cannot round-trip a config, and
`python-magic` solved a problem `x/crypto/ssh` does not have.

**What was explicitly not a reason: performance.** P8 is unambiguous — *"any design justified by
performance, concurrency, or scale is almost certainly solving a problem hasp does not have"* —
and tens of keys parse in milliseconds in either language. Choosing Go for speed would have meant
choosing it for the one reason this project's own principles reject.

### Consequence

- **`design.md` §10's implementation-stack item is closed**, and §10 cites this entry.
- **`T1`–`T26` all rest on this entry, and it is numbered after them.** That is an artifact of an
  append-only log, not of the decision arriving late — the choice was made before the technical
  design was written and simply never recorded. **T-ID order carries no chronology and must not
  be read as any.**
- **The predecessor repository is a historical artifact.** Nothing here depends on it, and
  `docs/project-assessment-2026-08.md` now carries a banner saying so, so that its Phase 0–5
  roadmap is not mistaken for live work.
- Every design question the assessment §5 left open is now answered in
  [`docs/decision-log.md`](decision-log.md); that mapping is recorded in the assessment itself
  rather than duplicated here.

---

<a id="t28"></a>
## T28 — Relative `IdentityFile` resolves against the key directory, and says so

**Date:** 2026-08-29 · **Status:** Accepted · **Amends:** [T5](#t5), [T16](#t16)
**Closes:** [`docs/tdd.md`](tdd.md) §15's relative-path question

### Context

An `IdentityFile` value may be a bare relative path — no leading `~`, no leading `/`. Real `ssh`
resolves it against **the working directory of the `ssh` process at connection time.**

hasp is never that process. It has no connection, no connection time, and no working directory
bearing any relationship to the one `ssh` will have. The value is therefore not resolvable by hasp
the way `ssh` resolves it — not *difficult*, but **undefined**.

`tdd.md` §5 proposed resolving against the key directory and flagged, correctly, that this was
*"a judgment call, not a transcription of OpenSSH's own rule."*

### Options

**(i) Resolve against the key directory, silently.** Matches what such a path almost always means
in a hand-written config. *Cost:* hasp reports a binding as fact when `ssh` may resolve it
somewhere else — asserting what it cannot verify, which is the thing P1 exists to prevent.

**(ii) Refuse to resolve; report the binding as unresolvable.** The strictest P1 reading,
symmetric with an unresolvable `%` token. *Cost:* J3's one-screen answer acquires a hole where a
perfectly working binding should be, and `check` reports a problem the user does not have. hasp
would be technically honest and practically wrong.

**(iii) Resolve, and report the divergence.** Both.

### Decision

**Option (iii).** hasp resolves a bare relative `IdentityFile` against the key directory
(`--key-dir`, default `~/.ssh`) — **and** emits a `check` finding, `relative-identityfile`
([T29](#t29), `severity: warning`), naming the raw value, the path hasp resolved it to, and the
fact that `ssh` will resolve it against its own working directory instead.

### Rationale

The two principles this sits between stop conflicting once the answer is allowed two parts. P7
asks the tool to answer the question — an inventory with a hole where a working binding belongs is
a worse answer than a resolved one. P1 forbids asserting what cannot be confirmed. A resolved
value *accompanied by a stated caveat* asserts nothing false: it reports what hasp did, and it
reports that `ssh` may do otherwise.

Choosing between them, rather than doing both, is what forces the bad trade. This is the same
shape as `design.md` §5.1's derivation gap, which reports `unknown` rather than guessing or
failing: hasp says exactly what it knows, and exactly what it does not.

**Rarity cuts toward the finding, not against it.** Bare relative `IdentityFile` values are
uncommon in real configs, so the finding costs almost nothing in noise. A rule that would be
unbearable at high frequency is free at this one.

### Consequence

- **`design.md` is deliberately unaffected.** This is a fact about `ssh_config(5)` and hasp's
  reading of it, not about hasp's domain — no principle moves and no upstream decision is needed.
  It is recorded here because `tdd.md` §5's divergence must be traceable to something, not because
  the design had a gap.
- `check` gains exactly one finding id ([T29](#t29)).
- The divergence stays documented in `tdd.md` §5 as a *stated* divergence. A user who hits it
  should learn why from the technical design, not by reading source.
- **If a real config ever shows key-dir resolution to be the wrong guess, the fix is to change
  what hasp resolves to — not to remove the finding.** The finding is what makes the resolution
  rule safe to change later.

---

<a id="t29"></a>
## T29 — `check` findings carry a stable `id` and a re-tunable `severity`

**Date:** 2026-08-29 · **Status:** Accepted · **Amends:** [T14](#t14)
**Closes:** [`docs/tdd.md`](tdd.md) §15's severity-taxonomy question

### Context

[T14](#t14) makes the JSON envelope a public contract — `hasp list key --json | jq` has to keep
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
| `duplicate-key-unconfirmed` | `info` | key ([T12](#t12)) |
| `key-missing-public-half` | `warning` | key |
| `key-no-profile` | `info` | key |
| `fingerprint-unknown` | `info` | key |
| `dangling-identityfile` | `error` | host |
| `unresolvable-token` | `warning` | host |
| `relative-identityfile` | `warning` | host ([T28](#t28)) |
| `shadowed-stanza` | `warning` | host ([T11](#t11)) |
| `host-no-binding` | `warning` | host ([T16](#t16)) |
| `stanza-in-multiple-groups` | `error` | host ([T13](#t13)'s partial-apply case) |
| `stanza-in-no-group` | `error` | host ([T13](#t13)'s partial-apply case) |
| `empty-profile-dir` | `info` | profile |
| `unmarked-profile-dir` | `info` | profile |
| `marker-defect` | `error` | host group ([T18](#t18)) |

**`severity` is advice, and may be re-tuned.** One of `error`, `warning`, `info`. `error` means
something is actually broken — a reference resolving to nothing, or a `Plan` that applied
partially. `warning` means it works but is probably not what was meant. `info` means it is worth
knowing and may be entirely deliberate.

**Severity never affects the exit code.** `check` exits `1` if there is any finding at all, of any
severity, and `0` if there are none — [T14](#t14)'s rule is unchanged. This is deliberate:
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
  set**, the same mechanism the settings admission rule uses ([T7](#t7)). A new finding id is then
  visible in a diff rather than discovered by a consumer.
- **`check` remains non-suppressible.** Nothing here adds a way to silence a finding — only a way
  to sort them. If suppression is ever wanted, `design.md` §5.7 and §6.2 already say it needs its
  own decision.
- Adding a finding kind is additive for a consumer filtering positively and needs no envelope
  version bump; `Envelope.Version` ([T14](#t14)) moves only if the *shape* changes.

---

<a id="t30"></a>
## T30 — The preview/apply race is detected by a witness, and fails closed

**Date:** 2026-08-29 · **Status:** Accepted · **Amends:** [T4](#t4), [T15](#t15)
**Closes:** a gap named, but not closed, by [T26](#t26)

### Context

[T26](#t26) ended by naming a problem it did not solve:

> Computing the diff means `Plan()` reads the target file's current bytes. Reads are always safe
> (P5), so this costs nothing in guarantees — but it does mean a `Plan` is computed against a
> snapshot, and a file changed between preview and confirm is a race the `Applier` must detect
> rather than assume away.

Nothing acted on it. `tdd.md` §11 presents itself as *"One table of every guard in the system and
which way it fails"* and had no row for this one — so an implementer reading that table would
have concluded, correctly per the document and incorrectly in fact, that the case was handled.

The window is real and not narrow. [D6](decision-log.md#d6)'s cycle is **preview → confirm → back
up → write**, and the confirm step is a human reading output: seconds to minutes. In that window
an editor can save `~/.ssh/config`, another shell can run `ssh-keygen`, or a dotfiles manager can
re-link the file.

**P8 does not excuse it.** *"A single human operating interactively"* rules out concurrent hasp
processes contending for a lock. It does not rule out that same human having the file open in
another window — which is the overwhelmingly likely form of this race, and arguably at its most
likely precisely when they are being asked to confirm a change to that file.

### Decision

**Every `Change` that read a "before" state records a witness of it, and `Applier` re-verifies
all witnesses before applying anything.**

```go
type Witness struct {
    Path    string
    Size    int64
    ModTime time.Time
    Sum     [32]byte // SHA-256 of the bytes Plan() actually read
}
```

- **Captured at `Plan()` time**, from the same read that produced `Preview().Diff` — no extra I/O.
- **Re-verified for the whole `Plan` before the first `Change` is applied**, not per-change as it
  goes. A `Plan` is the unit the user consented to, so it is the unit that gets validated.
- **Any mismatch fails closed.** Nothing is written, nothing is backed up, exit `3`
  ([T14](#t14)), with a message naming the file that changed and saying to re-run.

**The hash decides.** Size and mtime are carried because they make the failure cheap to explain,
but a file rewritten to byte-identical content is **not** a race — the preview is still accurate —
and hasp does not refuse on a touched mtime alone.

### Rationale

D6's safety is bought entirely by the user *reading* the preview. A plan whose preview no longer
describes the file it is about to write spends that safety without the user's knowledge: they
consented to a change against a state that no longer exists. Applying it anyway is worse than
refusing, because the refusal costs one re-run and the alternative costs a silent, unreviewed
write to a file P2 promises to protect.

Fail-closed is the default `tdd.md` §11 already declares for any guard without a stated exception,
and nothing here argues for an exception.

The competing option — re-read, recompute, and re-preview automatically — was rejected. It turns
one confirmed decision into a loop nobody asked for, and a user who has just read a diff and
pressed `y` should not be shown a *different* diff and asked again in the same breath. Re-running
is explicit, and costs one keystroke.

**Backups do not make this safe on their own.** P4 would let the user recover the clobbered file,
which is exactly the *apology rather than a safety property* shape
[D18](decision-log.md#d18) rejects for `release host`. The same reasoning applies here.

### Consequence

- **`tdd.md` §11's table gains its missing row**, which makes its claim to list every guard true.
- **A `Plan` is time-limited**, and that is worth stating explicitly: it is valid only against the
  machine state it was computed from. Nothing persists a `Plan` across processes today, and this
  entry is the reason not to start.
- A guard test mutates a target file between `Plan()` and `Apply()` and asserts three things: the
  write is refused, the target's bytes are unchanged, and **no backup was written** — a backup
  must not fire for a plan that never applies, or `~/.ssh/.hasp-backups/` fills with snapshots of
  writes that never happened.
- `WriteKeyFile` with `AllowOverwrite == false` ([T22](#t22)) is unaffected and keeps its own
  separate check: it fails closed on an existing target regardless of any witness, because it
  never read a "before" state to witness.

---

<a id="t31"></a>
## T31 — Semantic versioning, and the compatibility surface v1.0.0 freezes

**Date:** 2026-09-02 · **Status:** Accepted · **Amends:** [T14](#t14), [T29](#t29)

### Context

Four tags exist — `v0.1.0` through `v0.4.0` — and none of them rests on a stated compatibility
policy. [T14](#t14) already makes the `--json` envelope a contract, and [T29](#t29) already makes
a `check` finding's `id` permanent; a compatibility surface is therefore *implied* by decisions
already ratified, but it has never been written down as a surface in its own right — only as
promises scattered across individual entries. hasp cannot promise stability it has never
enumerated, and `design.md` §6.3 is explicit about why the promise exists at all: *"Every read has
a **machine-readable form**. hasp lives in a terminal beside other tools; a tool whose output can
only be looked at is half a tool."* [P6](design.md#4-principles) is the other half of the
argument: nothing hasp knows may be trapped inside it, which is only true in practice if a
consumer built against hasp today can trust what still holds tomorrow.

### Decision

hasp adopts **Semantic Versioning**. **v1.0.0 is tagged at the close of
[M3.5](roadmap.md#55-m35--hardening)** — the milestone that closes the SSH story and is the
natural point at which a compatibility surface can be frozen rather than merely described. M4/J9
then lands as **v1.1.0**, additive by construction: `tdd.md` §14 already constrains `Profile` to
stay additive across that boundary, which is precisely what a minor bump requires and a major one
would not.

The **public contract** — breaking any of the following needs a major version bump:

| # | Surface | Ratified by |
| --- | --- | --- |
| 1 | The four exit codes and their meanings, including that `1` stays `check`-exclusive | [T14](#t14) |
| 2 | The `--json` envelope's shape: `version`, `kind`, `data`, `warnings` | [T14](#t14) |
| 3 | `kind` strings (`key.list`, `host.show`, `check.report`, …) — permanent, retired rather than recycled, the same guarantee a `Dn`/`Tn` ID carries | [T14](#t14) |
| 4 | `check` finding `id`s | [T29](#t29) |
| 5 | The verb×noun grid and the global flag names and semantics — removing or renaming is breaking, adding is additive | [D10](decision-log.md#d10), `tdd.md` §9 |
| 6 | The on-disk marker syntax and metadata format — a change that makes an existing hasp-marked region unreadable by the new binary is breaking | `tdd.md` §6, §7; [T10](#t10), [T25](#t25) |

Row 6 is the least obvious and the most damaging of the six, because the artifact this promise
governs outlives the binary that wrote it: a `~/.ssh/config` marked in 2026 has to still parse
under whatever hasp binary someone runs against it in 2030.

The explicit **non-contract**, which matters as much as the contract does: human-readable output
([P7](design.md#4-principles) governs it, and [T14](#t14) already permits the two renderers to
diverge in *form* — a script needing stability uses `--json`); finding `severity`
([T29](#t29) makes it deliberately re-tunable); `--verbose` stderr diagnostics; and the exact
filename format inside `~/.ssh/.hasp-backups/` ([T8](#t8) — [P6](design.md#4-principles)
guarantees those backups stay legible and recoverable without hasp, not that their names never
change).

**The relationship between the two version numbers runs one way only.** [T14](#t14)'s own struct
comment describes `Envelope.Version` as "independent of hasp's own release version" — true, but a
phrasing that invites the wrong inference. Stated exactly: **a change to `Envelope.Version`
implies a major hasp version bump; a major hasp version bump does not imply a change to
`Envelope.Version`.** The envelope can stay stable across several major hasp releases; it cannot
change without one.

`hasp version`
([`tdd.md` §13](tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought))
is the surface that makes all of this checkable at runtime rather than merely asserted in a
document, so this entry is where the command is finally documented as design — [T9](#t9)
mentioned it only as a build-info consumer.

**Release notes are generated, not hand-maintained.** GoReleaser's `changelog: use: git` output is
the changelog of record; there is no hand-maintained `CHANGELOG.md`, because a file that must be
remembered is a file that drifts, and the git history already is the truth.

**The historical tag mapping**, recorded here so it is not mysterious later: `v0.1.0` → M1,
`v0.2.0` → M2, `v0.3.0` → M3, `v0.4.0` → the GoReleaser/Gitea release-plumbing commit that
followed M3's merge.

### Rationale

A compatibility surface stated after the first breaking change is a postmortem, not a promise. Six
of the nine principles in `design.md` are already load-bearing on this surface without saying so —
enumerating it now, before v1.0.0, is the only order in which "breaking this needs a major bump"
means anything.

### Consequence

**The Go module path problem is now load-bearing, not cosmetic.** `go.mod` declares
`github.com/boweeb/hasp`; the only remote is a local Gitea (`git@localhost:stuff/hasp.git`), so
`go install github.com/boweeb/hasp/cmd/hasp@latest` cannot resolve today, and Go's own module
rules mean a future v2 requires a `/v2` path suffix regardless of what the current path resolves
to. **v1.0.0 freezes the module path** as surely as it freezes the six rows above, so this has to
be settled *before* the tag, not after — [`roadmap.md` §5.5](roadmap.md#55-m35--hardening) carries
it as a named input-needed item rather than deciding it here, which is not this entry's place to
do.

---

<a id="t32"></a>
## T32 — Mage is the build/CI contract; platform workflows are thin shims

**Date:** 2026-09-02 · **Status:** Accepted

### Context

CI today is GitHub Actions YAML running against a Gitea remote — the near-term target — with GitHub
or GitLab as a long-term possibility if [T31](#t31)'s public-origin question resolves that way. As
it stands: no release workflow exists at all, so nothing triggers GoReleaser; the linter runs
unpinned (`go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest`); `.golangci.yml`
configures no linters, only a timeout; darwin is never compiled in CI, though GoReleaser targets
`darwin/arm64` among four platforms; and `mise.toml` is `[tools] go = "latest"`, which
`roadmap.md` §2 already calls "pinning the toolchain" and `tdd.md` §2 already calls "installed via
`mise` (`go = \"latest\"`)" — neither description is true of `latest`. Encoding the actual build
logic in any one platform's YAML makes a future move a rewrite instead of a shim.

### Decision

Every build/test/lint/release action becomes a **Mage target** in `magefiles/`
(`github.com/magefile/mage`, latest **v1.17.2**, verified against the module proxy this session);
a platform workflow file's only job is checkout → set up Go → invoke one target. The target set:
`Build`, `Test`, `Vet`, `Lint`, `Cross` (`GOOS=darwin` compile), `Fuzz`, `Fixtures`, `Docs`,
`Release`, and `CI` as the `mg.Deps` aggregate that runs the others.

**Zero-install bootstrap**, so CI needs no mage binary of its own: a `mage.go` carrying
`//go:build ignore` that calls `mage.Main()`, invoked as `go run mage.go <target>`. This bootstrap
has a caveat worth stating verbatim rather than discovering it against a red build later, from
magefile.org: *"because of the peculiarities of `go run`, if you run this way, go run will only
ever exit with an error code of 0 or 1."* A project with a four-code exit contract ([T14](#t14))
must not build an exit-code-sensitive pipeline — `CI`'s own exit status — on a bootstrap that
flattens every failure to `1`; the workflow shims (below) treat any non-zero exit from
`go run mage.go ci` as failure and never branch on its specific value.

**Dependency budget: a new row.** Mage is the first build-time-only dependency this project has
taken on. `tdd.md` §2's "Test-only, never shipped" table gets a sibling, "Build-time only, never
shipped," carrying `github.com/magefile/mage`. The guard is mechanical, in this project's own
style: [T13](#t13)'s existing `go list -deps` layering guard extends to assert
`github.com/magefile/mage` never appears in `cmd/hasp`'s dependency graph — a magefile importing
it is fine; `cmd/hasp` importing it is a regression. `//go:build mage` tags stay on every file
under `magefiles/`, so `go build ./...` and `go vet ./...` ignore them exactly as they do today.

**Also decided here**, because each is a direct consequence of moving CI into Mage rather than an
unrelated bundle of chores:

- `golangci-lint` is pinned to an exact version inside the `Lint` target — an unpinned `@latest`
  makes lint non-reproducible and turns an unrelated upstream release into a red build on a commit
  that changed nothing.
- `.golangci.yml` adopts a real linter set — noting that golangci-lint v2 requires `version: "2"`
  in the config file, which the current file does not declare.
- `mise.toml` is pinned to the exact Go toolchain in use, which resolves the contradiction named
  in Context.

**Gitea specifics**, named so they are not discovered the hard way: Gitea Actions consumes
GitHub-Actions-compatible YAML but needs a registered `act_runner`, and resolves a step's `uses:`
against its own `DEFAULT_ACTIONS_URL` rather than implicitly against github.com; the release shim
needs a `GITEA_TOKEN`. Both `.github/workflows/` and `.gitea/workflows/` shims are kept — each is
a handful of lines precisely because the logic lives in Mage, not in either file.

**The `Docs` target is the one that protects this project's own culture**, and is worth its own
paragraph rather than a line item. It commits the doc checkers this repository already relies on
by hand — broken links and anchors, every `Dn`/`Tn` citation resolving to a real `<a id>`, the
per-log invariants (index row count equals anchor count, IDs contiguous, every entry carrying a
`### Consequence`), and verbatim-quotation checking — and runs them in CI as part of `CI`'s
`mg.Deps` set. They are written in Go, as a Mage target, rather than adding a Python dependency:
the toolchain is already here, and a doc-verification script that needs a second language
installed is a script nobody runs locally. Two gotchas are recorded here because they have already
cost real time once and would again: (a) GitHub anchor slugs give each space its own hyphen and do
not collapse them, so an em-dash heading yields a *double* hyphen and a naive `\s+`→`-` regex
reports a dozen false breaks; (b) a quotation can wrap across source lines, so a single-line
`grep -F` produces false misses — normalize whitespace and strip `**`, `*`, and backticks before
comparing.

### Rationale

`mg.Deps` runs its dependencies concurrently and `mg.SerialDeps` runs them in sequence, and either
way Mage guarantees each dependency is "guaranteed to run exactly once in a single execution of
mage" — precisely the primitive a `CI` aggregate target needs to compose `Vet`, `Lint`, `Test`,
`Cross`, and `Docs` without hand-rolling its own dependency bookkeeping. `mage -compile <path>` can
also emit a standalone static binary later, if the zero-install `go run` path ever proves too slow
for local iteration — an escape hatch that costs nothing to have and nothing to use today.

### Consequence

- Moving to GitHub or GitLab later — the outcome [T31](#t31)'s public-origin question may
  produce — is a new shim file, not a rewrite: the workflow file changes, `magefiles/` does not.
- The doc-verification pass becomes mechanical, run on every push, rather than something someone
  has to remember to run before merging — which is exactly the discipline this repository's
  culture already claims to have and, until this entry, did not actually enforce.
- `mage.go`'s exit-code caveat means the workflow shims must treat "CI failed" as a single boolean,
  never inspect a specific non-zero code from the bootstrap path — a distinction any future
  workflow author needs to know before reaching for it.

---

<a id="t33"></a>
## T33 — Distribution is staged: self-hosted now, public channels blocked on one missing fact

**Date:** 2026-09-02 · **Status:** Accepted · **Amends:** [T9](#t9)

### Context

[T9](#t9) enumerated the full GoReleaser surface — `ko`, `nfpms`, `aur`, `homebrew_casks`,
`sboms`, `signs` — as though every section were equally reachable from where the project actually
stands. It is not: `.goreleaser.yaml`'s own header comment still opens "M0 scope only" and still
closes "Not releasing anything yet" — true when written, false since Gitea publishing was
configured. [T9](#t9) also states that generated shell completions and man pages are "packaged
into the release archives" — nothing in the repository generates either today, and
`.goreleaser.yaml` has no handling for them. `design.md` §3.1 scopes hasp to one laptop, one
human; the near-term distribution reality is a `localhost` Gitea remote, not a public one.

### Decision

**M3.5 ships**: `tar.gz` archives for all four build targets, **generated shell completions and
man pages** — closing the gap [T9](#t9) stated but never built, which cobra already provides via
`GenManTree` and the `completion` subcommand ([T3](#t3)) — plus `sboms`, published to the Gitea
release. `aur`, `homebrew_casks`, `ko`, and `signs` stay deferred.

**The synthesis that makes this one decision instead of four separate chores**, and the most
useful thing this entry records: **all four deferred channels are blocked on the same missing
fact — hasp has no publicly reachable origin.** AUR needs a fetchable source URL; Homebrew needs a
tap repository; `ko` needs a registry to push an image to; keyless signing needs a public
OIDC-issuing CI provider. A `localhost` Gitea satisfies none of the four. It is also, precisely,
**the same question** [T31](#t31) already names as the module-path blocker: one answer to "does
hasp get a publicly reachable origin" unlocks all five items — the module path and the four
deferred channels — and no answer blocks all five. No amount of additional GoReleaser
configuration substitutes for answering it.

### Rationale

Treating four TODOs as four independent chores invites solving each partway — a tap repo pointed
at a private remote, a signing setup with no public issuer to trust — none of which actually work.
Naming the single shared blocker converts four speculative pieces of GoReleaser configuration into
one tracked decision with a clear unblocking condition.

### Consequence

The deferral is now a single named blocker — recorded as an input-needed item in
[`roadmap.md` §5.5](roadmap.md#55-m35--hardening) — rather than four independent "later" items
that each look individually actionable and are not. `.goreleaser.yaml`'s stale "M0 scope only"
header becomes a tracked M3.5 doc-hygiene item rather than a comment nobody owns.

---

<a id="t34"></a>
## T34 — User-facing documentation is generated wherever it can drift

**Date:** 2026-09-02 · **Status:** Accepted

### Context

`docs/` is entirely design documentation aimed at *building* hasp. There is nothing aimed at
*using* it — no install path, no command reference, and nothing telling a user what the files hasp
leaves in `~/.ssh` actually are. That last gap is not a nicety: [P6](design.md#4-principles)
promises that "If hasp breaks, or is abandoned again, the user loses a convenience and nothing
else" — and that promise is only real if someone who has never run hasp can read those artifacts
without it.

### Decision

Split user-facing documentation by whether it can drift out of step with the binary.

**Generated, regenerated by a Mage target ([T32](#t32)'s `Docs`), CI fails if stale** — diff-check
regenerated output against the committed copy, the same mechanism [T29](#t29) and [T7](#t7)
already rely on for their own golden-list guard tests, so a gap becomes visible in a diff rather
than in a consumer's `jq` filter silently failing to match: man pages, shell completions, and a
CLI reference under `docs/cli/` generated from cobra's markdown generator. A hand-written flag
table is a table that will be wrong by v1.1.0.

**Hand-written narrative**, because it carries argument rather than syntax:

- A real root `README.md` — orientation, install, a sixty-second demo, license.
- A user guide organized around **J1–J8** ([`design.md` §7](design.md#7-journeys)), since the
  journeys are already the project's success stories and give the guide a table of contents that
  `design.md` itself keeps honest.
- A page on what hasp leaves on disk — the one this project specifically owes its user: `.hasp`
  markers, in-file region markers, the settings file, and `~/.ssh/.hasp-backups/`, **which
  [T8](#t8) states hasp never prunes**, so the user must be told plainly that it grows without
  bound and that pruning is theirs to do. Paired with the full withdrawal path — release every
  adopted resource, delete the markers — which is what makes [D14](decision-log.md#d14)'s two-way
  door and P6 literal instead of aspirational.
- A `SECURITY.md` that restates guarantees which already exist rather than inventing new ones:
  [P3](design.md#4-principles) and `design.md` §3.2's position that hasp never takes custody of
  private key material, and [D17](decision-log.md#d17)'s four constraints on the single passphrase
  prompt — generation only, never persisted, never transmitted, buffer zeroed.

### Rationale

A generated reference and a hand-rolled one fail differently, and the split follows that fault
line rather than a topic boundary. Anything that is really a restatement of the command surface —
flags, subcommands, man pages — is wrong the moment the code changes under it unless something
regenerates and checks it; anything that carries *why* — a journey, a threat model, a promise about
what happens if hasp is abandoned — has no source of truth to regenerate from and has to be
written and kept honest by hand, the same way `design.md` itself is.

### Consequence

`docs/README.md` now indexes two audiences instead of one — documentation aimed at *building*
hasp, and documentation aimed at *using* it — and the generated half of the second audience cannot
silently disagree with the binary, because [T32](#t32)'s `Docs` target fails the build when it
does.

---

<a id="t35"></a>
## T35 — The fingerprint scheme registry: open, pure-Go, no subprocess, no network

**Date:** 2026-09-03 · **Status:** Accepted

### Context

[D20](decision-log.md#d20) widens [J2](design.md#7-journeys) from matching spellings to matching
**schemes** — the same key produces a different fingerprint depending on how it was computed, and
AWS alone uses three: SHA-1 over the PKCS#8 DER of the *private* key for an AWS-created RSA key,
MD5 over the PKIX/SPKI DER of the *public* key for an imported RSA key, and SHA-256 over the SSH
wire-format public key for ED25519 (created or imported — identical either way). This session
verified all three against `https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/verify-keys.html`
and confirmed them on the repository's own fixtures:

| Scheme | Hashed input | Hash | Needs | Verified value (`testdata/keys/rsa-pem-plain-pub`) |
| --- | --- | --- | --- | --- |
| AWS created-RSA | PKCS#8 DER of the **private** key | SHA-1 | private key, decrypted | `97:47:11:3c:af:56:47:b3:f9:a9:89:36:6d:ca:be:0b:33:a0:05:f7` |
| AWS imported-RSA | PKIX/SPKI DER of the **public** key | MD5 | public half only | `a8:e7:45:95:5f:a3:f0:b1:79:6c:c2:f1:d2:80:57:ea` |
| AWS ED25519 (created *or* imported) | SSH wire-format public key | SHA-256 | public half only | `SHA256:lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA` (`testdata/keys/ed25519-openssh-plain-pub`) |
| Legacy SSH MD5 | SSH wire-format public key | MD5 | public half only | `34:29:f4:da:3c:db:49:4b:35:ba:c1:c2:cd:2e:75:a8` |

**The fourth row is the one a reader should not skip.** Legacy SSH MD5 — what `ssh-keygen -E md5`
prints, what OpenSSH printed by default before 6.8, and what a great deal of surviving
documentation shows — has **exactly the same shape** as AWS imported-RSA (16 bytes, colon-hex, 47
characters) and a **completely different value**, because one hashes the SSH wire-format blob and
the other hashes the PKIX/SPKI DER encoding. The two rows above prove it on the same key:
`34:29:f4:…` versus `a8:e7:45:…`. [T1](#t1) anticipated exactly this, noting that
`FingerprintLegacyMD5` *"exists if a future `find key` clue needs to match one"*. Registering it
here, rather than leaving it to be discovered by a user whose correct clue silently failed to
match, is the difference between an open registry and an AWS-shaped one.
[T36](#t36) carries the consequence: a 47-character clue is ambiguous and both schemes are
computed.

**Six shell "strategies" collapse to three values.** Proven on the repository's own fixtures:
`openssl rsa -pubout -outform DER | openssl md5` and
`ssh-keygen -e -m PEM | openssl rsa -RSAPublicKey_in -outform DER | openssl md5` both produce the
imported-RSA value above, on the same input. The second pipeline exists only because `openssl rsa
-in` cannot read an OpenSSH-format private key — verified this session: exit 1, `"Could not find
private key"`. hasp parses both formats natively ([T1](#t1)), so it needs **one code path per
scheme**, never one per shell incantation that happens to produce that scheme.

**A shell pipeline that loses its input silently emits a hash of nothing**, and that hash is
shaped exactly like a real fingerprint: `d41d8c…` is MD5 of the empty string, `da39a3…` is SHA-1
of the empty string. Either one slots into a fingerprint-shaped field without complaint. A native
Go implementation that reads the key bytes it already parsed cannot make that mistake — there is
no pipe stage for the input to fall out of.

### Decision

A **registry** of scheme definitions, each declaring: a stable `id`, the hash function, the
encoding it renders as, and — the load-bearing field — **which key material it needs**: public
half only, or the decrypted private key. A scheme is `func(Key) (Fingerprint, error)`, computed
entirely from bytes hasp already has in memory; nothing in the registry ever opens a socket. This
is the boundary the user ratified explicitly: **a scheme is an encoding of facts about the key,
computed locally — never a network call** — which keeps §3.2's *"Not a key distribution
mechanism."* intact even though this feature is, on its face, about matching a value from an
external console.

Every scheme is stdlib or already-required: `x509.MarshalPKCS8PrivateKey` +
`crypto/sha1` for created-RSA; `x509.MarshalPKIXPublicKey` + `crypto/md5` for imported-RSA; the
existing `x/crypto/ssh` wire-format marshaling + `FingerprintSHA256` ([T1](#t1)) for the
SSH-native/ED25519 scheme. **Zero new modules.**

**The registry is open from the start, not closed at three.** GitHub and GitLab both display
SHA-256 fingerprints — already covered by the SSH-native scheme — and other consoles will surface
their own schemes over time. Each scheme's declared input requirement is what lets a caller (§9's
`find`, [T36](#t36)) know in advance whether a given key can even be evaluated against it without
asking the user for anything.

### Rationale

The three-scheme table is not incidental complexity — it is the literal shape of the problem
[J10](design.md#7-journeys) exists to answer, and D20's own rationale is that a false negative here
reads as *"you don't have this key,"* which is worse than an honest miss. The empty-digest hazard
is the strongest available argument for computing every scheme natively rather than shelling out
to whichever `openssl`/`ssh-keygen` pipeline happens to exist on the caller's `PATH`: a broken
pipe silently produces a value indistinguishable from a real one, and [T1](#t1)'s "no subprocess"
rule already forecloses that failure class for the read path generally — this is one more scheme
falling under a rule already in force, not a new exception to it.

### Consequence

- [T36](#t36)'s `find` and any future `--investigate` surface consume the registry as a single
  dependency; adding a fourth scheme is one new entry, not a new call site scattered through the
  command surface.
- Each scheme's declared input requirement (public half vs. private key) is exactly what
  [D19](decision-log.md#d19)'s consent gate keys off of: a caller can enumerate every scheme
  computable from what has already been read, and knows in advance which remaining ones would
  require asking the user for something.
- `tdd.md` §5 gains scheme computation as a projection step in the derivation pipeline, and §12
  gains a fixture-backed test asserting the AWS vectors above by name, plus a guard that no scheme
  performs I/O beyond reading the key file already in memory.

---

<a id="t36"></a>
## T36 — `find` matches across every registered scheme; the clue's shape routes the search

**Date:** 2026-09-03 · **Status:** Accepted — amends `tdd.md` §9's `find` cell

### Context

[D20](decision-log.md#d20) requires `find key` to match a clue given in *any* scheme hasp knows,
not only the SSH-native one `tdd.md` §9 originally described ("identify a key from a fingerprint
fragment... normalizing punctuation and case"). [T35](#t35) supplies the schemes; this entry
supplies the matching algorithm — including how a bare clue, with no metadata attached, is routed
to a candidate scheme before any key is even examined.

### Decision

**Normalization**, applied to the clue before routing: strip colons and internal whitespace,
lowercase hex digits, strip base64 `=` padding (AWS emits it, `ssh-keygen` omits it — D20 names
this explicitly), and tolerate an optional `SHA256:`/`MD5:` prefix.

**The clue's raw shape NARROWS the search; it does not uniquely determine it.** That distinction
is the whole of this decision, and getting it wrong would reintroduce the silent miss
[D20](decision-log.md#d20) exists to eliminate.

| Clue shape | Candidate schemes |
| --- | --- |
| 59-character colon-hex (40 hex digits) | SHA-1 — [T35](#t35)'s created-RSA scheme. One candidate |
| 47-character colon-hex (32 hex digits) | MD5 — **two** candidates: AWS imported-RSA (MD5 over PKIX/SPKI DER) **and** legacy SSH MD5 (MD5 over the SSH wire-format blob) |
| base64, with or without a `SHA256:` prefix | SHA-256 — the SSH-native/ED25519 scheme. One candidate |

**The MD5 collision is real and was anticipated by this log.** [T1](#t1) already noted that hasp
reports SHA-256 and *"never the legacy MD5 colon-hex form, though `FingerprintLegacyMD5` exists if
a future `find key` clue needs to match one"* — this entry is that future. Legacy SSH MD5 and AWS
imported-RSA are **different hashes over different byte sequences of the same key**: the former
over the SSH wire-format public blob, the latter over the PKIX/SPKI DER encoding. They produce
different digests and identical shapes. A 47-character clue is therefore ambiguous by
construction, and `find` **computes both and compares both** rather than guessing one. Legacy MD5
clues are not exotic — they are what `ssh-keygen -E md5` prints, what OpenSSH printed by default
before 6.8, and what a great deal of still-circulating documentation shows.

`find` computes only the schemes a clue's shape admits, not every registered scheme for every key,
so the common case (an SSH-native clue) stays exactly as cheap as it is today. What the shape
buys is a smaller candidate set, never a single answer.

**Which scheme matched is the origin evidence — but only for the AWS schemes.** Per
[P10](design.md#4-principles): a match under created-RSA or imported-RSA is deterministic proof of
provenance, because AWS computes one or the other depending on exactly how the key came to exist,
so `find` reports origin as `confirmed`. A match under **legacy SSH MD5** proves nothing about AWS
provenance — it is simply another way of naming the same public key — so origin stays `possible`.
The same holds for the SSH-native/ED25519 scheme, because AWS's ED25519 fingerprint is identical
whether the key was created or imported (Background, verified against AWS's documentation). Only
two of the four registered schemes carry provenance, and conflating "the clue matched" with "the
origin is proven" is exactly the error P10's vocabulary exists to prevent.

### Rationale

D20's own rationale carries this entry: a silent miss on a clue given in a scheme hasp did not
think to try reads as "you don't have this key," which is a confidently wrong answer, not a
missing one — precisely the question §2 opens the whole document with. Routing by shape rather
than computing every scheme against every key keeps the common path — an SSH-native fingerprint,
the overwhelming majority of clues in practice — exactly as cheap as it was before this entry.

### Consequence

- `tdd.md` §9's `find` cell for **key** is amended to describe multi-scheme matching; the **host**
  and **profile** `find` cells are unaffected, since neither is fingerprint-shaped.
- A clue that matches under the created-RSA scheme is proof hasp read the *private* key
  ([T35](#t35)) — which only happens under [D19](decision-log.md#d19)'s consent gate — so a
  successful created-RSA match is also, incidentally, evidence the gate fired correctly for that
  key during this invocation.
- `check`'s existing duplicate-detection vocabulary ([T12](#t12)) already distinguishes confirmed
  from unconfirmed; `find`'s confirmed/possible split is the same distinction applied to a
  different question, which is part of the evidence [D22](decision-log.md#d22) cites for P10's
  taxonomy being right-sized rather than newly invented.

---

<a id="t37"></a>
## T37 — Confidence is a closed, permanent vocabulary in the output contract

**Date:** 2026-09-03 · **Status:** Accepted · **Amends:** [T14](#t14), [T29](#t29), [T31](#t31)

### Context

[D22](decision-log.md#d22) ratified [P10](design.md#4-principles): every reported fact carries a
status from a closed set — `derived`, `confirmed`, `possible`, `unknown`. [T14](#t14) already makes the `--json` envelope a
public contract, [T29](#t29) already makes a `check` finding's `id` permanent, and [T31](#t31)
already enumerates the compatibility surface those two entries imply. Confidence has to land in
exactly that surface, not beside it, or it is a promise nobody actually made.

### Decision

An `--investigate` response carries an `origins` array (and, more generally, any reported fact
carries the same shape wherever P10 applies):

```json
"origins": [
  {"id": "aws-ec2-created", "confidence": "possible",
   "because": ["algorithm=rsa", "format=pem", "no-console-fingerprint-supplied"]}
]
```

`because` carries **machine-readable evidence tokens, never prose** — a fixed vocabulary of short,
`key=value`-shaped strings a consumer can branch on without parsing English. Free text belongs in
the human renderer only, which is free to turn the same tokens into a sentence.

**Contrast with [T29](#t29)'s `severity` explicitly, because the two look alike and are not.**
`severity` is deliberately re-tunable — hasp may decide next year that a finding deserves a
different severity without breaking anything, because [T29](#t29) says so outright. `confidence`
is **closed and permanent**: a consumer filtering on `derived` to decide whether to *trust* a
value is making a safety decision, and a vocabulary a producer can silently expand out from under
that filter would be a contract violation dressed as a feature. Nothing about the confidence
vocabulary is subject to future re-tuning the way `severity` is.

[T31](#t31)'s compatibility-surface table gains a seventh row: **the confidence
vocabulary — `derived`, `confirmed`, `possible`, `unknown` — is closed; adding, removing, or
redefining a value is a breaking change**, exactly like [T29](#t29)'s finding `id`s (row 4) and
unlike its `severity` (explicitly named in the non-contract).

**A correction to [T31](#t31)'s own body, recorded here since this is the entry that amends it.**
T31's Decision section states *"v1.0.0 is tagged at the close of
[M3.5](roadmap.md#55-m35--hardening)"* — true when T31 was written, on 2026-09-02, against the
milestone structure that existed then. [D21](decision-log.md#d21) and [D22](decision-log.md#d22)
split that milestone: `roadmap.md` §5.5 (M3.5 — Hardening) now closes without tagging anything, and
a new §5.6 (M3.6 — Investigation) closes with the tag instead, per `design.md` §9. T31's sentence
is **pre-amendment text** — the same treatment [T27](#t27) and [T29](#t29) give an earlier draft's
superseded claim — and the current truth is `roadmap.md` §5.6, not §5.5.

### Rationale

[D22](decision-log.md#d22)'s own rationale is the evidence this taxonomy is right-sized: this one
feature genuinely needs all four statuses at once — a matched RSA console fingerprint is
`confirmed` ([T36](#t36)); an ED25519 guess is `possible`; a value read straight off an artifact
(most of what `list key` already reports) is `derived`; an underivable value stays `unknown`,
exactly as §5.1 already uses the word. Landing it in the same three entries that already govern
the output contract, rather than inventing a fourth place for stability promises to live, is what
keeps [T31](#t31)'s enumeration actually complete.

### Consequence

- `Envelope` itself ([T14](#t14)) is unchanged in shape — `version`, `kind`, `data`, `warnings` —
  because confidence lives inside `data`, not the envelope wrapper. T14 is amended in the sense
  that its `data` payloads for investigation-bearing kinds now carry a new, contract-bound shape,
  not in the sense that the struct above changes.
- [T29](#t29)'s own text is unchanged and remains correct; this entry's contrast with it is worth
  reading alongside T29 for any future reader deciding whether a new field belongs in the
  `severity` camp (re-tunable) or the `confidence` camp (closed).
- `tdd.md` §16's compatibility-surface table gains the row described above, citing this entry.

---

<a id="t38"></a>
## T38 — `ssh-agent` is a derivation source for public facts, and its contribution is labelled

**Date:** 2026-09-03 · **Status:** Accepted

### Context

§5.1's derivation gap is total for an OpenSSH-format encrypted key's **comment** — the public half
is embedded unencrypted in the private file, so the fingerprint and algorithm are derivable, but
the comment lives inside the encrypted blob and is not. A loaded `ssh-agent` already holds the
decrypted key in memory for exactly this reason — it has to, to sign challenges — and its protocol
exposes both the public key and the comment for anything currently loaded, with **no passphrase
and no decryption performed by hasp at all**.

`golang.org/x/crypto/ssh/agent` was confirmed this session to exist inside the already-required
`golang.org/x/crypto` module ([T1](#t1)), at the version already pinned (`v0.55.0`), and its
`agent.Key` type carries a `Comment` field. **This adds no new module dependency** — the same
module `go.mod` already requires for the entire key-parsing read path supplies the agent client
too.

### Decision

Under `--investigate` ([D21](decision-log.md#d21)), hasp connects to `SSH_AUTH_SOCK` if it is set,
lists the agent's loaded keys, and cross-references each by public key against the scanned key
set. For every match, the agent's comment is reported for that key, **labelled `agent-sourced`**
rather than merged into the ordinary `derived` bucket — the same treatment
[T16](#t16) gives implicit-default bindings: a real fact, but a distinct *kind* of fact, and
merging kinds that arrived differently is exactly the dishonesty T16 was written to avoid.

**Named honestly: an agent-sourced fact is real but transient.** Unlike a fact read from a file,
which persists until the file changes, an agent-sourced fact is only as durable as the agent's own
running state — it can vanish the moment the agent restarts, the key is removed from it, or the
process exits. This is a genuine wrinkle in [P1](design.md#4-principles)'s "the world is the
truth" framing: the *agent* is part of the world at the moment hasp asks, and gone from it a
moment later, which is precisely why the label matters more here than anywhere else P1 applies.

**Declared failure stance: no agent, or `SSH_AUTH_SOCK` unset, degrades silently to what is
derivable without it — never an error.** A read must stay safe
([P5](design.md#4-principles)) regardless of whether the environment happens to have an agent
running, and treating an absent agent as a failure would make `--investigate` unusable on a
machine with no agent at all, which defeats the point of a mode meant to surface more, not less.

### Rationale

This closes the comment gap for OpenSSH-format encrypted keys **without ever touching a
passphrase** — it is strictly cheaper and safer than [T39](#t39)'s prompt-based path, and is tried
first for exactly that reason. It serves [J10](design.md#7-journeys) directly: an investigation
needs more than a plain read, and an agent a user already has running is data hasp was leaving on
the table. Zero new dependency keeps [T1](#t1)'s "no subprocess" boundary, and the Go-and-stdlib-
first preference [T27](#t27) already established, both intact.

### Consequence

- `tdd.md` §5 gains the agent as a named derivation source, feeding the same projection step
  [T35](#t35)'s scheme registry feeds.
- `tdd.md` §9's `show key --investigate` and `list key --investigate` cells surface agent-sourced
  comments, visibly labelled as such in both renderers.
- `tdd.md` §11 gains a fail-open row: no agent / `SSH_AUTH_SOCK` unset → degrade to what is
  derivable without it, never an error.
- [T39](#t39)'s passphrase-gated path tries the agent first, so a key already loaded never
  triggers a prompt for a fact this entry already supplies.

---

<a id="t39"></a>
## T39 — Passphrase-gated derivation: explicit, lazy, one passphrase per invocation, degrades without a TTY

**Date:** 2026-09-03 · **Status:** Accepted · **Amends:** [T6](#t6) · **Amended by:** [T49](#t49)

### Context

[D19](decision-log.md#d19) generalizes [D17](decision-log.md#d17)'s four constraints from `new
key`'s write path to any read: hasp may read private key material only when the user explicitly
asked for an operation that needs it, only for that operation's duration, held in memory and
zeroed after, never automatically. Two concrete facts need exactly this: [T35](#t35)'s created-RSA
scheme, which hashes the *decrypted private key* and is therefore `unknown` for every encrypted
key regardless of format unless the user consents to unlocking it; and an OpenSSH-format
encrypted key's comment, when [T38](#t38)'s agent path finds nothing loaded.

### Decision

**Prompt only under `--investigate`**, and only for a key where a passphrase would unlock
something otherwise completely `unknown` — the created-RSA scheme is the case that matters, since
it is `unknown` in its entirety without it; an OpenSSH-format key's comment alone is often not
worth interrupting the user for, since the fingerprint and algorithm are already derivable without
asking. **Try the agent first** ([T38](#t38)) — a key already loaded needs no prompt at all.

**One passphrase, tried across every candidate key in the invocation** — D19's ladder, stated
plainly: not one prompt per key, one prompt per run, held in memory and zeroed after, using the
same `x/term.ReadPassword` path [T6](#t6) already established for `new key`. No new dependency.

**Declared failure stance, and it is the important one: no TTY (for example, `--investigate
--json` in a pipe) DEGRADES — it does not fail closed.** hasp reports what is derivable without
the passphrase and marks the rest `unknown` with a machine-readable reason, e.g.
`passphrase-required-no-tty`, rather than refusing to run at all.

### Rationale

Writes fail closed ([T6](#t6), [T22](#t22)) because a wrong write is destructive and irreversible
in the way P4's backups do not fully undo for key material. A read has no such asymmetry — the
worst outcome of proceeding without a passphrase is an honest `unknown`, which is exactly what
[D12](decision-log.md#d12) already treats as the correct answer when a fact cannot be derived. A
read that exits non-zero because nobody was present to type a passphrase would make
`--investigate` useless in precisely the scripted, non-interactive workflows it exists to serve —
the opposite failure from the one `new key`'s fail-closed default guards against.

**This is a deliberately different stance from `new key`'s fail-closed non-interactive path
([T6](#t6)), and the asymmetry is the point, not an inconsistency.** `new key` fails closed because
silently generating an unprotected secret is worse than refusing. `--investigate` degrades because
silently under-reporting a fact you already told hasp to go looking for is a worse failure than
`new key`'s would be, in the opposite direction — a read that refuses to run at all is *less*
honest than one that runs and says `unknown`, not more.

### Consequence

- [T6](#t6)'s index is annotated as amended by this entry: T6's own fail-closed default remains
  correct and unchanged for `new key`'s **write** path; this entry is the citation that keeps a
  future reader from assuming T6's stance generalizes to every non-interactive case, including
  reads, which it explicitly does not.
- `tdd.md` §9's `show key --investigate` grid cell inherits this mechanic; no new flag is needed
  beyond `--investigate` itself — the mode's default resolves to "prompt if a TTY is attached,
  else degrade," the mirror image of `new key`'s "prompt if a TTY is attached, else fail closed."
- `tdd.md` §11 gains a fail-open (degrade) row for this case, stated beside T6's fail-closed row so
  the read/write asymmetry is visible in the same table rather than scattered across two.
- A guard test drives `--investigate --json` with stdin redirected from `/dev/null` and asserts
  exit `0`, every derivable fact present, and the undecidable ones marked `unknown` with
  `passphrase-required-no-tty` — never a hang, never a non-zero exit.

---

<a id="t40"></a>
## T40 — The public origin is GitHub; T33's shared blocker resolves and distribution restages

**Date:** 2026-09-04 · **Status:** Accepted · **Amends:** [T31](#t31), [T32](#t32), [T33](#t33)

### Context

[T33](#t33) named one missing fact behind four deferred distribution channels — `aur`,
`homebrew_casks`, `ko`, and `signs` — and predicted that a single answer would unlock all four at
once. [T31](#t31) already made the same fact load-bearing for the Go module path, because v1.0.0
freezes it before the tag rather than after. [T32](#t32) took a local Gitea as the near-term CI
target, naming GitHub or GitLab only as a long-term possibility "if [T31](#t31)'s public-origin
question resolves that way." It has resolved: the author dropped Gitea — a temporary local service
that had become more trouble to run than it was worth — and moved `origin` to
`git@github.com:boweeb/hasp.git`. (The reason is recorded as the author's, given when the change
was made; it is deliberately not set in quotation marks, because a quotation this log cannot
resolve against a file in the repository would be a permanent false positive for the
verbatim-quotation check [T32](#t32)'s `Docs` target specifies.)

`main` is the repository's default branch, per the author. The Python predecessor's history is
temporarily co-located at the same remote on `master`, bound for cold storage and deletion.
`git merge-base main origin/master` reports no common ancestor, so the two histories are unrelated
— which keeps `docs/README.md`'s and `design.md`'s existing statements that the predecessor's
`README.rst` is not carried into this repository literally true of this history. Branch layout on
the remote beyond that single fact is not verified here: SSH access to GitHub is unavailable from
this session.

### Decision

Three consequences follow, stated plainly:

1. **The module path is now honest.** `github.com/boweeb/hasp` resolves;
   `go install github.com/boweeb/hasp/cmd/hasp@latest` works; [T31](#t31)'s freeze-before-tag
   requirement is met before v1.0.0 is tagged, not discovered after.
2. **GitHub Actions is the CI platform.** The `.gitea/workflows/` shim [T32](#t32) planned as a
   deliverable is not built — Mage remains the contract; only the platform invoking it changed.
3. **Distribution restages.** `signs` (cosign keyless signing via GitHub Actions' OIDC issuer) and
   `ko` (publishing a container image to `ghcr.io`) join M3.5's shipping set alongside archives,
   generated completions, man pages, and `sboms`. `aur` and `homebrew_casks` remain deferred.

### Rationale

Two points are worth making, and the second is the more useful one.

**[T32](#t32)'s thin-shim rule is not validated by this migration, and claiming otherwise would be
the wrong lesson to draw.** An earlier draft of this entry said the rule "paid off"; it did not,
because there was nothing yet for it to pay off *on*. The honest state of the repository is that
`magefiles/` and `mage.go` **do not exist**, the only workflow — `.github/workflows/ci.yml`, from
M0 — hardcodes `go build`, `go vet`, `golangci-lint@latest` and `go test` directly in YAML, which
is precisely the shape T32's rule exists to prevent, and there is no release workflow at all. The
migration cost exactly one thing: a `release.gitea` → `release.github` change in
`.goreleaser.yaml`.

What this *does* establish is narrower and still worth acting on: **the platform moved before any
platform-specific build logic was written, which is the cheapest moment such a move can happen and
the strongest argument for adopting T32's rule now rather than after the release pipeline is
built.** The rule remains untested. T32's own Context already names this gap, and nothing here
closes it — `magefiles/`, the zero-install `mage.go` bootstrap, a GitHub Actions shim that actually
invokes a Mage target, and a tag-triggered release workflow are all still outstanding M3.5 work.

**The two remaining deferrals are qualitatively different from the four [T33](#t33) named, and
collapsing them would be a mistake.** Origin was *a fact hasp lacked*; a Homebrew tap and an AUR
package repository are *work the author must choose to take on* — each a separate repository the
author must create and maintain. The first kind of deferral ends when someone answers a question;
the second ends when someone commits to maintenance. Naming the difference keeps `aur` and
`homebrew_casks` honest deferrals rather than ones inherited unexamined from [T33](#t33).

This entry serves `design.md` §3.1's one-laptop scope and [P6](design.md#4-principles): nothing
hasp knows may be trapped inside it, and a publicly resolvable module path plus a real release is
what makes that promise reachable by someone who is not the author.

### Consequence

- [`roadmap.md` §5.5](roadmap.md#55-m35--hardening) loses its last `INPUT NEEDED` block — M3.5 now
  has no open questions.
- [`tdd.md` §15](tdd.md#15-open-questions) drops from two open items to **one** (multi-directory
  key locations), and the T-log's own `## Still open` roll-up below is updated in the same pass —
  a count that has drifted before.
- `.goreleaser.yaml`'s `gitea_urls` and `release.gitea` became dead config pointing at `localhost`
  the moment origin moved; corrected in this pass to `release.github`.
- The predecessor's Python history is temporarily co-located at the same remote on `master`, bound
  for cold storage and deletion. `git merge-base` confirms the two histories share no ancestor, so
  the documentation set's existing claim that the predecessor's `README.rst` is not carried into
  this repository stays literally true.

---

<a id="t41"></a>
## T41 — `CI`'s `mg.Deps` aggregate excludes `Fixtures`: non-deterministic fixtures race `Test` and break `TestFullInventory`

**Date:** 2026-09-04 · **Status:** Accepted · **Amends:** [T32](#t32)

### Context

M3.5.2 (`roadmap.md`'s CI-shim chunk) rewired `.github/workflows/ci.yml` to invoke exactly one
step, `go run mage.go ci`, in place of the hardcoded `go build`/`go vet`/`golangci-lint`/`go test`
steps [T40](#t40) found still in place — the first time anyone actually ran `go run mage.go ci`
against the real workflow. [T32](#t32)'s original target set for `CI`'s `mg.Deps` aggregate was
`Build, Vet, Lint, Test, Cross, Fixtures`. Running it surfaced two real bugs, not one:

1. **A race.** `mg.Deps` runs its dependencies concurrently ([T32](#t32)'s own Rationale cites
   this). `Fixtures` shells out to `go run ./tools/genfixtures`, which rewrites all 28 files under
   `testdata/keys/` using `crypto/rand`, while `Test` reads those same files — `Test` can start
   reading a fixture directory `Fixtures` is still mid-write, producing spurious failures like "got
   27 private-key fixtures, want 28".
2. **A correctness bug, worse than the race.** Serializing `Fixtures` before `Test` does not fix
   this: `tools/genfixtures` regenerates fresh, random key material on every run, and
   `internal/cli/fullinventory_test.go`'s `TestFullInventory` hardcodes a fingerprint-derived
   assertion against the *currently-committed* fixture content — `run("find", "key",
   "T4RN-6go9")` is a mangled fragment of the real SSH fingerprint of the committed
   `testdata/keys/ed25519-openssh-plain-nopub` file, asserted to resolve to `id_ed25519_foobarco`.
   Regenerating that file with a fresh random keypair changes its fingerprint and breaks the
   assertion deterministically, every time — not flakily, and not fixable by reordering.

Direct experiment during this chunk confirmed both: `go run mage.go ci` as [T32](#t32) specified it
fails with the race symptom and, after serializing, with the `TestFullInventory` fingerprint
mismatch; `go build ./... && go vet ./... && go test ./...` run without touching `Fixtures` at all
— i.e. against `testdata/keys/` exactly as committed — passes clean, including
`TestFullInventory`. `tools/genfixtures`'s own package doc already calls it "a development-time
convenience, run via `go run ./tools/genfixtures` from the repo root": a manual
regenerate-then-commit step, not something CI should re-run and discard on every invocation.

### Decision

`CI`'s `mg.Deps` aggregate drops `Fixtures`, leaving `Build, Vet, Lint, Test, Cross` — the same set
[T32](#t32) already excluded `Fuzz` from, for the same reason: `Fixtures` is a standalone,
manually-invoked dev-time target (`go run mage.go fixtures`), not part of the blocking set every
`CI` run executes. `Fixtures` itself is unchanged as a target; only its membership in `CI`'s
dependency list is removed. `Test` does **not** gain an `mg.Deps(Fixtures)` call either — that
ordering was tried and rejected for the same correctness reason: any run that touches
`testdata/keys/` invalidates `TestFullInventory`'s hardcoded fingerprint fragment regardless of
race conditions.

### Rationale

`testdata/keys/` as checked into git is hand-committed, checked-in content that other tests
already depend on by fingerprint value, not a build artifact `CI` is entitled to regenerate and
discard. Treating `Fixtures` as part of the blocking aggregate conflated two different kinds of
target: things that verify the tree as committed (`Build`, `Vet`, `Lint`, `Test`, `Cross`) and a
thing that mutates the tree (`Fixtures`). Mixing a mutator into a concurrent `mg.Deps` set with the
tests that read its output was the root mistake — `mg.Deps`'s concurrency guarantee ("exactly once
per execution," cited in [T32](#t32)'s Rationale) says nothing about ordering between
non-dependent targets, so `Fixtures` and `Test` were always racing once both were listed together.

### Consequence

- [`tdd.md` §17](tdd.md#17-continuous-integration-and-release-automation)'s statement of `CI`'s
  actual `mg.Deps` set is corrected to `Build, Vet, Lint, Test, Cross` and now says the thin-shim
  rule is exercised by a real workflow, not still unproven.
- Anyone regenerating fixtures locally must still run `go run mage.go fixtures` by hand and commit
  the result — `CI` will never do this silently, and a future fixture regeneration that changes
  `TestFullInventory`'s fingerprint assumptions must update that test in the same commit.
- The general lesson generalizes beyond this one target: no future Mage target that mutates
  checked-in files under `testdata/` (or anywhere else `git status` would show as dirty) should be
  added to `CI`'s `mg.Deps` set without checking whether another target in that same concurrent set
  reads what it writes.

---

<a id="t42"></a>
## T42 — Signing the release covers both the checksum and the `kos`-built container image, via two GoReleaser sections

**Date:** 2026-09-05 · **Status:** Accepted · **Amends:** [T9](#t9)

### Context

[T9](#t9) named `signs` as one of the GoReleaser v2 idioms this project embraces;
[`tdd.md` §13](tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought)
repeats it without saying which artifact gets signed. Implementing the `signs:` block for M3.5.5
([`roadmap.md` §5.5](roadmap.md#55-m35--hardening)) first assumed GoReleaser had no signing hook
for `kos:`-built images, since the top-level signing documentation lists archives, installers,
packages, checksums, and `dockers:`-built manifests, with no direct mention of `kos:`. That
assumption was wrong: GoReleaser's own `kos:` documentation states that a ko-built manifest is
added to the same artifact list `dockers:`-built images use, and it is `docker_signs:` — not
`signs:` — that signs artifacts from that list via `artifacts: manifests`.

### Decision

Both artifact classes carry a keyless cosign signature, via two separate `.goreleaser.yaml`
sections because each artifact lands in a different GoReleaser artifact list: `signs:` signs the
release checksum (`sign-blob --bundle`, `artifacts: checksum`); `docker_signs:` signs the
`ghcr.io/boweeb/hasp` image manifest `kos:` pushes (`sign`, `artifacts: manifests`). Both use the
same OIDC/Fulcio/Rekor keyless flow — no private key, no separate credential to manage.

### Rationale

The checksum file covers every archive and SBOM this release produces, but it says nothing about
the container image's integrity — a signed checksum and an unsigned image would have left exactly
the gap `signs`'s stated purpose (supply-chain attestation, [`tdd.md` §13](tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought))
exists to close. Once GoReleaser's actual capability was confirmed rather than assumed, shipping
both signatures cost one more config block, not a deferred chunk.

### Consequence

- [`tdd.md` §13](tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought)'s
  staging table now says `signs` ships covering both the checksum and the container image, citing
  this entry.
- A future reader reaching for `signs:` to cover a `kos:`-built artifact should reach for
  `docker_signs:` instead — GoReleaser's own section split, not a hasp-specific quirk, but easy to
  miss since `kos:` and `docker_signs:` don't sit next to each other in the config file.

---

## Still open

**Nothing.** `T1` through `T42` are all accepted.

`T27` through `T30` came out of the second review pass rather than the first drafting of
[`docs/tdd.md`](tdd.md), and that is worth recording as a fact about the process rather than a
defect in it. The first pass ran with `docs/design.md` frozen, so gaps it found could only be
**named**, not closed. Naming them was correct. Closing them required the freeze to lift — which
is also how [D16](decision-log.md#d16), [D17](decision-log.md#d17) and
[D18](decision-log.md#d18) came to exist.

**`T35` through `T39` are the same story a second time**, for the investigation capability rather
than the settings file or the passphrase prompt: [D19](decision-log.md#d19)–[D22](decision-log.md#d22)
lifted the freeze `docs/design.md` had been under since the first reassessment, and these five
entries are what a technical design does with room newly opened upstream.

**One item in `tdd.md` §15 remains genuinely open, and it is open by choice rather than
omission**, as of the reassessment recorded in [T31](#t31)–[T40](#t40):

- **Multi-directory / non-default key locations**, deferred by `docs/design.md` §10 and not
  designed here. `--key-dir` is threaded explicitly through every layer rather than defaulted
  anywhere inside `internal/app` or `internal/domain`, so supporting more than one is additive
  whenever a case for it actually arrives.

**Two items that stood here alongside it are now closed, not merely narrowed.**

**The public-origin / module-path question** ([T31](#t31), [T33](#t33)) is closed by
[T40](#t40): the author moved `origin` to `git@github.com:boweeb/hasp.git`, which settles
`go.mod`'s path before v1.0.0 freezes it and unblocks every deferred distribution channel at
once — exactly as [T33](#t33) predicted a single answer would.

**SSH key inspection detail** — what `list key` and `show key` should report beyond the fact set
`tdd.md` §9's grid already named — was open because it awaited the user's own specification of
which candidate additions mattered. The investigation capability answered it from an unexpected
direction: rather than adding fields to the plain read, [T35](#t35)–[T39](#t39) add a
confidence-graded `--investigate` mode that reports fingerprint scheme, origin, and agent- or
passphrase-derived facts — a superset of every candidate the input-needed block in
[`roadmap.md` §5.5](roadmap.md#55-m35--hardening) once named — while leaving `list key`'s plain
output untouched, satisfying P7 exactly as the closed item's own bound required.

---

<a id="t43"></a>
## T43 — `adopt key`'s alias-preserving move is one atomic `Change`, not an ordered two-`Change` pair

**Date:** 2026-09-08 · **Status:** Accepted · **Amends:** [T20](#t20)

### Context

[T20](#t20) is correct about the ordering *rule* — a `Plan`'s `Changes` are ordered so any prefix
leaves a working state — but its own worked example has drifted from what shipped. T20 describes
`adopt key` inverting `[MoveFile, CreateSymlink]` into `[CreateSymlink, MoveFile]`, two ordered
`Change`s. That is not the mechanism in the codebase.

What actually ships: `AdoptKeyUseCase.Plan` (`internal/app/adoptkey.go`) builds a single
`ReplaceWithSymlink{From, To}` `Change` per file — one for the key, and a second, independent one
for its `.pub` sidecar when present, because the two paths never interact and there is no ordering
question between them. `ReplaceWithSymlink.Apply` (`internal/app/change_replacewithsymlink.go`)
delegates to `WriteFS.ReplaceWithSymlink` (`internal/app/plan.go`), whose documented contract is
copy the source's bytes to the destination, verify byte-for-byte, then atomically `os.Rename` a
freshly created symlink onto the source — one rename, one primitive, never a bare unlink of the
source.

### Decision

T20's ordering rule is unchanged and still governs any future `Change` kind that genuinely needs
two ordered steps. What is corrected here is the record of `adopt key`'s own mechanism: it is one
atomic `Change`, not two `Change`s kept in a careful order. There is no reachable intermediate
state at all — not even the "dangling symlink is inert" state T20's worked example relied on —
which is a strictly stronger guarantee than an ordered two-`Change` pair provides.

### Rationale

The two-`Change` shape was not merely reordered, it was rejected outright, in either direction.
`[MoveFile, CreateSymlink]` leaves the key briefly undiscoverable if the symlink step fails.
`[CreateSymlink, MoveFile]` — T20's own described order — is not equivalently safe, either:
`os.Rename`'s replace-on-conflict semantics mean a move landing on an existing path can destroy
the very file the symlink was standing in for, which is worse than the original problem, not a
fix for it. `docs/tdd.md` §4 ("Change ordering") and §11 (`Write mechanics`, point 4) already
carry this exact reasoning against the single atomic primitive that shipped instead. T20's worked
example simply predates that primitive landing; the append-only convention means the fix is a new
entry naming T20, not an edit to it.

### Consequence

- `docs/tdd.md` §4 and §11 already state the current mechanism correctly, so no `tdd.md` change
  and no code change accompanies this entry — it corrects `tech-decision-log.md` alone.
- `release`'s mirror operation, `WriteFS.ReplaceSymlinkWithFile` (`internal/app/plan.go`), follows
  the identical one-`Change` shape for the identical reason: a single atomic replace has no
  ordering problem to guard against in the first place.
- The general lesson for future write use cases: collapse a move-plus-alias operation into one
  atomic `Change` wherever the underlying filesystem primitive supports it, and reach for T20's
  ordered-multi-`Change` case only when it genuinely does not.

---

<a id="t44"></a>
## T44 — `edit key --replace-material` routes through `WriteKeyFile{AllowOverwrite: true}`, not a separate move rule

**Date:** 2026-09-08 · **Status:** Accepted · **Amends:** [T22](#t22)

### Context

[T22](#t22)'s Consequence section claims `edit key --replace-material` "deliberately does not
route through `WriteKeyFile`" and instead uses T15's copy → verify → unlink move rule. That is not
what shipped. `WriteKeyFile` (`internal/app/change_writekeyfile.go`) gained an `AllowOverwrite`
field — `false` for `new key`, always; `true` only for `--replace-material` — and
`EditKeyUseCase.planReplaceMaterial` (`internal/app/editkey.go`) builds `WriteKeyFile{...,
AllowOverwrite: true}` `Change`s directly against the key's existing real path.

### Decision

`--replace-material`'s `Plan` is: a `WriteKeyFile{AllowOverwrite: true}` `Change` against the
key's current path, plus a matched `.pub` sidecar `Change` — `WriteKeyFile{AllowOverwrite: true}`
if the incoming material has its own `.pub`, `Remove` if it doesn't but the old key did, and no
sidecar `Change` at all if neither has one. `RequiresBackup()` mirrors `AllowOverwrite` exactly, so
T22's safety property — the one legitimate overwrite is unconditionally backed up first — still
holds; only the mechanism changed.

Ordering matters here too: the `.pub` `Change` is planned *before* the private-key `WriteKeyFile`,
per T20's "any prefix leaves a working state" rule. A partial apply that stops after the `.pub`
step leaves the old, still-matching private key on disk — self-consistent either way. The reverse
order is worse: a successful private-key swap followed by a failed `.pub` cleanup would leave new
key bytes on disk with a stale `.pub` silently misreporting the old fingerprint, a P1 violation
with no `Preview` to have caught it first, since a partial-apply failure happens after `Preview`
already showed both changes as a pair. This reasoning lives in `planReplaceMaterial`'s own comment
in `internal/app/editkey.go`.

### Rationale

T22's stated reason for wanting a *separate* mechanism — this is the one path allowed to overwrite
key material, and it must be unconditionally backed up first — is satisfied directly by
`AllowOverwrite` and `RequiresBackup()`'s coupling on `WriteKeyFile` itself. There is no need for a
second `Change` kind that duplicates `WriteKeyFile`'s own write mechanics for the sake of one
caller. This keeps [D4](decision-log.md#d4)'s no-operation-destroys-an-irreplaceable-secret
guarantee, backup-first included, expressed in one place instead of two.

### Consequence

- `docs/tdd.md` §4's `WriteKeyFile` type sketch already documents `AllowOverwrite` and cites T22
  for it, so no `tdd.md` change and no code change accompanies this entry — it corrects
  `tech-decision-log.md` alone.
- T22's fail-closed default (`AllowOverwrite: false`) for `new key` is unaffected; only
  `--replace-material`'s own record is corrected.
- The `.pub`-before-private-key ordering is specific to `--replace-material`'s own two-`Change`
  shape, not a restatement of T20's general rule — it exists because this particular pair of
  `Change`s has a genuine ordering question, unlike [T43](#t43)'s single-`Change` case.

---

<a id="t45"></a>
## T45 — The clean-room test: `tools/cleanroom`, a `CleanRoom` Mage target, a post-release CI job, and a shared `internal/snapshot` package

**Date:** 2026-09-08 · **Status:** Accepted

### Context

[`roadmap.md` §5.5](roadmap.md#55-m35--hardening) exit criterion 6 states the milestone's own
mechanical proof: a clean-room install of `hasp` (README.md's one documented path, `go install
github.com/boweeb/hasp/cmd/hasp@latest`) must be able to run `list key` against a synthetic
`~/.ssh` fixture and get an accurate inventory back, then round-trip that fixture through `adopt`
and `release` byte-identical to where it started. Unlike every other M3.5 chunk, this criterion
had no assigned chunk and nothing in the repository beyond the roadmap's own prose implemented it
— confirmed by grep for "clean-room" and "clean room" turning up nothing else.

Two duplicated snapshot/diff implementations already existed by the time this entry was written:
`internal/app/adopt_release_roundtrip_test.go`'s `snapshotDir`/`assertSnapshotsEqual` (digest
includes each file's permission mode) and `internal/cli/writeguard_test.go`'s
`snapshotTree`/`assertTreeUnchanged` (digest omits it) — the first file's own doc comment already
named the reason for the duplication: `internal/cli` imports `internal/app`, so the dependency
cannot run the other way, and a `*testing.T`-coupled helper in one package cannot be called from
the other. A closer read for this entry found a **third**, `internal/cli/writeguard_alltree_test.go`'s
`snapshotTreeForGuard`/`assertTreeUnchangedForGuard` — needed because that file lives in package
`cli` itself (the only place `write.go`'s unexported `isStdinTTY` can be forced without a real
pty), and a package-`cli` file cannot see symbols defined in a package-`cli_test` file even though
both live under `internal/cli/`. `tools/cleanroom` was about to become a legitimate fourth caller
of the same logic.

### Decision

**A new package, `internal/snapshot`,** holds the digest/diff logic as two plain functions with no
`*testing.T` coupling — `Snapshot(root string) (map[string]string, error)` and
`Diff(before, after map[string]string) []string` (empty means identical) — importable from any of
`internal/app`, `internal/cli` (both packages), and `tools/cleanroom` without regard to which
already imports which. It keeps the stronger, mode-including digest `internal/app`'s version used;
the other two callers gain mode-change detection they didn't have before, not lose coverage they
did. All three prior call sites become thin wrappers: `snapshotDir`/`assertSnapshotsEqual`,
`snapshotTree`/`assertTreeUnchanged`, and `snapshotTreeForGuard`/`assertTreeUnchangedForGuard` each
keep their own name (for their own file's callers) and `t.Helper()`/`t.Errorf` shape, but their
bodies now call `internal/snapshot.Snapshot`/`.Diff` instead of reimplementing the walk. This
consolidates three existing near-duplicates into one, rather than adding a fourth.

**A new tool, `tools/cleanroom`,** matching the existing `tools/docscheck`/`tools/gendocs`/
`tools/genfixtures` convention (standalone `package main`, run via `go run ./tools/<name>`, never
imported by `cmd/hasp`). `tools/cleanroom/fixture.go` builds a small, realistic `~/.ssh`-shaped
fixture — one key already adopted into a managed profile (a `.hasp`-marked directory, D13) with
its top-level alias symlink, one still-unmanaged key, and a real ed25519 keypair via
`internal/adapter/keyfile.Generate` (the same pure-Go mechanism `new key` itself uses, T1) rather
than raw bytes, since `keyfile.Inspect` fails open and silently skips anything it cannot classify
as a key. `tools/cleanroom/main.go` execs a real `hasp` binary as a subprocess against that
fixture — `list key --json`, `adopt key ... --json --yes`, `release key ... --json --yes` — reusing
`internal/cli/render.Envelope` to unmarshal the outer `--json` envelope (its `Data` payload needs
its own narrow, unmarshal-only view, since `domain.Key`'s `Identity` field is a
`json.Marshaler`-only interface with no matching `UnmarshalJSON`). It snapshots the fixture before
and after (excluding `.hasp-backups/`, mirroring the existing round-trip test's own exclusion of
hasp's own backup store, T8/P4) via `internal/snapshot`, and reports every subprocess's own
non-zero exit and every snapshot mismatch as a hard failure — no partial-success state, per the
roadmap's own "deliberately a script rather than a judgement."

**A new, standalone Mage target, `CleanRoom`,** alongside `Fuzz`/`Fixtures`/`GenDocs`/`Release` —
excluded from `CI`'s `mg.Deps` for the same family of reasons those targets already are. Reads
`HASP_CLEANROOM_BIN`; if unset, builds the current tree to a temp binary first, so `go run mage.go
cleanroom` is also a self-contained local dev-loop smoke test.

**A new job in `.github/workflows/release.yml`, not `ci.yml`,** running after the existing
`release` job (`needs: release`) so the pushed tag and its published artifacts already exist. It
checks out the repository — unlike the hypothetical "no checkout at all" shape a simpler design
might reach for, `tools/cleanroom`'s own source lives in this repository, so the harness driving
the test legitimately needs it — but the `hasp` binary under test is never built from that
checkout. It comes from `go install github.com/boweeb/hasp/cmd/hasp@${{ github.ref_name }}`, the
tag just pushed, never `@latest`, wrapped in a bounded retry loop (six attempts, 20s apart) to
absorb the Go module proxy's own indexing lag immediately after a fresh tag push. No `container:`
directive is needed for this isolation: a GitHub Actions job is a fresh VM by default, and this
workflow does not share any `actions/cache` between the `release` job and this one, so `go install`
here already cannot resolve from the `release` job's own build/module cache — the actual risk this
criterion exists to rule out.

### Rationale

**No Dockerfile.** A container image would need to be built, published, and kept in sync with
`mise.toml`'s pinned Go version by hand — a second place that version could drift from the one
`docs/tdd.md` §17 already cites as the single source of truth. A plain GitHub Actions job already
gives the needed isolation (a fresh VM, no shared cache) for free.

**`release.yml`, not `ci.yml`.** [T41](#t41) already established the precedent this decision
extends: don't put a slow, network-dependent, and occasionally-flaky check into the aggregate that
blocks every commit and PR. `go install ...@<tag>` depends on the public Go module proxy actually
having indexed a tag that, from this workflow's own point of view, was pushed moments ago —
exactly the kind of external timing dependency `CI`'s `mg.Deps` set has no business carrying.
Gating it on `release` instead means it only runs when there is a real tag to prove, at the one
point in the release process a stranger's own `go install` would actually happen.

**`@<tag>`, not `@latest`.** `@latest` resolves to whatever the proxy currently considers the
newest version at request time — for a workflow run immediately following the tag that triggered
it, that should be the same tag, but nothing guarantees it stays that way if a second tag lands in
the same window, and it gives up the one guarantee this job actually needs: proving *this* tag's
artifact, not proximately-the-newest-one.

### Consequence

- `docs/tdd.md` §17's target-set sentence now lists `CleanRoom` alongside `Build`, `Test`, `Vet`,
  `Lint`, `Cross`, `Fuzz`, `Fixtures`, `Docs`, and `Release`, and gains a paragraph describing what
  it does and why it stays outside `CI`'s `mg.Deps`, matching how `Docs` and `Release` are already
  described there.
- `internal/app/adopt_release_roundtrip_test.go` and `internal/cli/writeguard_test.go` (and, found
  during this chunk, `internal/cli/writeguard_alltree_test.go`) all changed to call
  `internal/snapshot` instead of reimplementing it; none of their own test bodies or assertions
  changed, and the full existing suite (`go test ./...`) stays green.
- A future caller that needs "did this directory tree change" reaches for `internal/snapshot`
  first — a fourth reimplementation of the same ~20 lines would now be a regression in its own
  right, not just an opportunity missed.
- [`roadmap.md` §5.5](roadmap.md#55-m35--hardening) exit criterion 6 now has a real, mechanical
  implementation behind its prose, runnable locally (`go run mage.go cleanroom`) and in CI
  (`.github/workflows/release.yml`'s `clean-room` job) exactly as the criterion itself demands.

---

<a id="t46"></a>
## T46 — Container base image: `cgr.dev/chainguard/static` → `gcr.io/distroless/static:nonroot`

**Date:** 2026-09-09 · **Status:** Accepted

### Context

Re-cutting `v0.6.0` after making the repository public ([T40](#t40)) failed at the `ko` publish
step: `fetching base image: GET https://cgr.dev/v2/chainguard/static/manifests/latest:
MANIFEST_UNKNOWN`. Querying `cgr.dev` directly (outside CI) confirmed this isn't a missing-tag
fluke — anonymous pulls now return `401 Unauthorized` outright, and Chainguard's free tier
requires signing up with a business email, a bar a personal open-source project the author
maintains alone cannot honestly clear. `.goreleaser.yaml`'s `kos:` block had used
`cgr.dev/chainguard/static:latest` since M3.5's shipping-set, deliberately left floating rather
than pinned to a digest — the comment there contrasted this against [T32](#t32)'s dev-tool pins,
so a distroless base would keep picking up security patches on every release.

### Decision

Replace `cgr.dev/chainguard/static:latest` with `gcr.io/distroless/static:nonroot` in
`.goreleaser.yaml`'s `kos:` block. The floating-tag choice itself is unchanged — `:nonroot` still
tracks upstream patches on every release rather than freezing a digest — only the registry and
image family change, to one that still permits anonymous pulls.

`:nonroot`, not the bare `:latest` tag, is the deliberate part of this decision: Chainguard's
`static` image runs as a non-root user by default, while Google's plain
`gcr.io/distroless/static:latest` runs as **root** unless the `:nonroot` tag is used. Swapping to
the bare tag would have silently regressed the container's UID posture — exactly the sharp edge
`tdd.md` §13's "Container caveat" already calls out by name ("a mismatched UID inside the
container silently produces a `~/.ssh` the host user cannot read"). `:nonroot` keeps the same
UID-safety behavior the Chainguard image provided.

### Rationale

**Distroless, not Alpine or a full base.** [T32](#t32)'s floating-vs-pinned distinction depended on
the image being minimal and low-churn to begin with, not on it being Chainguard's specifically.
`gcr.io/distroless/static` is the same shape — no shell, no package manager, matching `tdd.md`
§13's "no `Dockerfile`, `FROM scratch`-capable" description of the container target — from a
registry with no anonymous-access gate.

**Why not authenticate to Chainguard instead of switching.** A free Chainguard account is real,
but gating it behind a business email is a signal, not an accident: personal open-source
maintainers are not the audience it is priced for. Wiring a login step and a token secret into
`release.yml` to work around that would trade a one-line `base_image:` swap for an ongoing
account-maintenance dependency this project has no organizational backing to sustain.

**This entry serves [P6](design.md#4-principles) directly — the same clause [T40](#t40) already
invoked for the module path.** T40's own rationale is that a publicly resolvable module path is
what makes P6's "nothing hasp knows may be trapped inside it" promise reachable by someone who is
not the author; a container image gated behind a business-email signup is exactly the same kind
of barrier applied to a different artifact, `ko`'s published image rather than `go install`'s
module path. An image nobody but a signed-up account can pull is, in practice, a released artifact
only the author can use — the same failure mode P6 already names for anything hasp would keep
only to itself, applied here to distribution rather than to data.

### Consequence

- `.goreleaser.yaml`'s `kos:` block now points at `gcr.io/distroless/static:nonroot`; its comment
  still names Chainguard once, specifically to explain why `:nonroot` — not the bare tag — was
  chosen; the base image itself is no longer Chainguard's.
- No `tdd.md` change: §13 already described the container target generically, without naming a
  specific registry or vendor, so nothing there was inaccurate to begin with.
- `v0.6.0` is re-cut a second time against this fix, expected to clear the `ko` publish step and
  reach [T45](#t45)'s clean-room job on the now-public repository.

---

<a id="t47"></a>
## T47 — The fingerprint-scheme no-network guard is a direct-import assertion, not a `net.Dialer` `Control` hook

**Date:** 2026-09-09 · **Status:** Accepted — amends [T35](#t35)

### Context

`tdd.md` §12 states the no-network guard for [T35](#t35)'s fingerprint scheme registry in these
terms: *"A guard test runs every registered scheme's `Compute` inside a `net.Dialer` whose
`Control` callback fails any socket creation, and asserts no scheme ever trips it."* Implementing
`internal/adapter/fpscheme` for [M3.6.1](roadmap.md#56-m36--investigation) this session, that
mechanism turned out not to work at all: a `net.Dialer` constructed inside a test has no effect on
code under test that never holds a reference to it. Every `Scheme.Compute` function in this
package operates purely on bytes already in memory (`crypto/x509`, `crypto/sha1`, `crypto/md5`,
and `golang.org/x/crypto/ssh`'s wire-format marshaling) — none of them accepts a `net.Dialer`, a
`context.Context`, or any other hook a test could use to intercept a dial that is never attempted.
A `Control`-callback guard would pass unconditionally regardless of whether the code under test
ever opened a socket, which makes it worthless as a guard: it cannot fail.

A second, unrelated problem surfaced while designing the replacement: this package's own required
dependency, `golang.org/x/crypto/ssh` (needed for `ssh.FingerprintSHA256` and SSH wire-format
marshaling — [T35](#t35)'s own table), itself directly imports `"net"` for its unrelated
`Dial`/`Listen`/`Conn` machinery elsewhere in that package. Verified this session:
`go list -deps golang.org/x/crypto/ssh` includes `net`, `net/netip`, and `net/url`. A guard built
the way [T13](#t13)'s `internal/domain` layering guard and [T32](#t32)'s `cmd/hasp` layering guard
both work — `go list -deps .`, the full transitive closure — would therefore fail permanently the
moment `fpscheme` imports `ssh` at all, for a reason completely unconnected to whether this
package's own code ever dials.

**What this guard actually stands in for.** [T35](#t35)'s "never a network call" boundary is not
incidental to the scheme registry — it is what keeps `design.md` §3.2's *"Not a key distribution
mechanism"* boundary intact even though matching a console fingerprint is, on its face, about an
external system. That boundary is itself [P3](design.md#4-principles)'s scope half: hasp manages
arrangement, not secrets, and a scheme that phoned out to verify a fingerprint against a console
would be hasp acting as exactly the distribution mechanism §3.2 forbids. This entry exists only to
make that boundary mechanically checkable rather than a claim resting on review discipline, so it
cites [P3](design.md#4-principles) directly, not only the T35/T13/T32 entries that shape its
mechanism.

### Decision

Replace the `net.Dialer`/`Control` mechanism with a `go list`-based guard, the same mechanical
shape [T13](#t13)'s and [T32](#t32)'s layering guards already use, but scoped to
**`internal/adapter/fpscheme`'s own direct imports** rather than its full transitive dependency
graph: `go list -f '{{ join .Imports "\n" }}' .` (non-transitive — direct imports only,
production code only, test files excluded), asserted to contain none of `net`, `net/http`, or
`os/exec`.

The direct-vs-transitive distinction is the load-bearing part of this decision, not an
implementation detail: a transitive check is factually impossible to pass while this package
depends on `golang.org/x/crypto/ssh` (Context above), so the only assertion that can ever be both
true and useful is "this package's own source code never imports a networking or subprocess
package" — which is exactly [T35](#t35)'s actual claim ("no scheme ever opens a socket").

### Rationale

An alternative considered: keep the `net.Dialer`/`Control` mechanism as a best-effort documentation
device even though it cannot fail. Rejected — a test that cannot fail is worse than no test, since
it reads as coverage in a diff and in `go test -v` output without providing any. [T13](#t13) and
[T32](#t32) already establish this codebase's preferred alternative to "review discipline" for a
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

---

<a id="t48"></a>
## T48 — `find key` never prompts for a passphrase; an inapplicable-for-that-reason scheme is an explicit warning, not a silent miss

**Date:** 2026-09-10 · **Status:** Accepted

### Context

[T36](#t36) supplies multi-scheme `find`'s matching algorithm — which schemes a clue's shape
admits, and that both MD5 candidates are computed rather than one guessed — but does not say
whether `find` may prompt for a passphrase to evaluate a candidate key against a scheme that needs
the decrypted private key. One registered scheme does: `aws-created-rsa`
([T35](#t35)) hashes the *decrypted* private key, so a 40-hex-digit clue routes to a scheme that,
for an encrypted candidate key with no already-derivable private material, cannot be evaluated at
all without exactly the passphrase [T39](#t39) scopes prompting to `--investigate` alone.
`tdd.md` §9's global-flag table already lists `--investigate`'s own cost as *"speed, and possibly
a passphrase prompt"* and states the flag is absent from every other verb×noun cell — `find`
included — but nothing said what `find` should report when it hits exactly the case that flag
exists to unlock.

### Decision

`find key` never prompts for a passphrase, under any circumstance. [T39](#t39)'s prompt is scoped
entirely to `--investigate`; `find` is not that flag, and does not borrow its consent. A key that
cannot be evaluated against a candidate scheme because the decrypted private key is unavailable —
concretely, [T35](#t35)'s `aws-created-rsa` scheme against an encrypted candidate key — is **never
silently absent from the result**: it is reported as an explicit warning, naming the scheme and
how many keys in the search could not be evaluated against it, distinct from an honest "this key's
fingerprint under this scheme does not match."

### Rationale

[D20](decision-log.md#d20)'s own rationale is that a false negative during `find` reads as "you
don't have this key," which is a confidently wrong answer, not a missing one. Treating "never
evaluated" identically to "evaluated and did not match" reintroduces that exact failure one layer
inside the search itself: a user holding the right key, encrypted, sees no signal distinguishing
"hasp checked and this isn't it" from "hasp never actually looked." An explicit warning keeps that
distinction visible without asking `find` to do what only `--investigate` is licensed to do — one
passphrase, held for one operation, that operation being `--investigate`, not `find`
([D19](decision-log.md#d19), [P3](design.md#4-principles)). This also keeps `find` fast and
non-interactive, matching [J2](design.md#7-journeys)'s framing of identification as an ordinary,
low-friction lookup — the opposite of [J10](design.md#7-journeys)'s deliberately higher-friction
investigation.

### Consequence

- `internal/app.FindKeys` returns `(matches []FindMatch, warnings []string)` rather than a bare
  slice: `keyfile.OpenMaterial` is always called with a `nil` passphrase, and a scheme reporting
  `domain.ReasonPrivateKeyUnavailable` for a candidate key increments that scheme's own warning
  count rather than being dropped.
- Both renderers surface the warning: `render.JSON`'s existing `warnings` field
  ([T14](#t14)) needs no contract change, and the human renderer gains its own channel (stderr,
  after the results table — advisory, not part of the answer stdout carries, `tdd.md` §10).
- `tdd.md` §9's `find` cell for **key** and §18's "Multi-scheme `find`" subsection are amended to
  state this plainly.

---

<a id="t49"></a>
## T49 — Narrowing §18/T39's comment-recovery claim: a passphrase prompt cannot recover an OpenSSH-format key's comment; only `ssh-agent` can

**Date:** 2026-09-10 · **Status:** Accepted — amends [T39](#t39)

### Context

`tdd.md` §18's "Passphrase-gated derivation" subsection and [T39](#t39) both state that, when
[T38](#t38)'s agent path finds nothing loaded, a passphrase prompt is one of two facts
passphrase-gated derivation exists to recover — the other being [T35](#t35)'s created-RSA scheme —
implying an OpenSSH-format encrypted key's comment is recoverable either way: from the agent, or
by prompting and decrypting.

That second path does not exist. Verified this session against
`$(go env GOMODCACHE)/golang.org/x/crypto@v0.56.0/ssh/keys.go` (the exact version this module
pins, `go.mod`): `parseOpenSSHPrivateKey` parses an OpenSSH-format key's on-wire comment into one
of `openSSHRSAPrivateKey`, `openSSHEd25519PrivateKey`, or `openSSHECDSAPrivateKey` — each of those
three structs carries its own `Comment string` field — but `parseOpenSSHPrivateKey` itself returns
only `(crypto.PrivateKey, error)`, and its caller, the exported
`ParseRawPrivateKeyWithPassphrase`, returns only `(interface{}, error)`. Neither return type
carries the comment anywhere a caller of the public decrypt API can reach — it is parsed, used
internally, and then discarded before the function returns. `internal/adapter/keyfile.OpenMaterial`
calls exactly this function ([T35](#t35)'s own read path), so no amount of decrypting through
hasp's own code, with a correct passphrase or otherwise, ever recovers this comment. `ssh-agent`
recovers it by a completely different mechanism ([T38](#t38)): the agent protocol exposes a
loaded key's comment directly, with no decryption performed by hasp at all, which is why that path
works when this one cannot.

### Decision

The passphrase-gated path's scope is **`aws-created-rsa` alone**. An OpenSSH-format encrypted
key's comment is **agent-only** ([T38](#t38)): if `ssh-agent` has nothing loaded for that key,
the comment stays `unknown`, full stop — a passphrase prompt is not a fallback for it, because
there is no code path by which a correct passphrase would ever produce it.

### Rationale

This is a narrowing forced by a fact about a dependency's own API shape
([P3](design.md#4-principles), [D19](decision-log.md#d19)): D19's consent gate licenses *reading*
private key material the user explicitly asked for, but licensing the read does not manufacture a
return value the library used to perform it never exposes. A subprocess wrapping `ssh-keygen`
directly could recover the comment (it parses the same on-wire structure independently), but
[T1](#t1) forbids subprocessing for exactly the reasons that rule exists — pure-Go derivation,
no shelling out — so that route is not available either. [J10](design.md#7-journeys)'s promise is
to surface everything hasp *can* determine, clearly marked with how sure hasp is; stating a
capability the code cannot actually deliver would be the opposite of that promise, not a detail in
service of it.

### Consequence

- `tdd.md` §18's "Passphrase-gated derivation" subsection is corrected: "two facts need
  [passphrase-gated derivation]" becomes one (`aws-created-rsa`), and the comment gap is described
  as agent-only, closing over the incorrect claim rather than leaving it as current truth.
- `tdd.md` §18's `ssh-agent` subsection gains a note that a `byPath`-identified key ([T1](#t1)'s
  undecidable row) has no derivable public key at all, so even a loaded agent has no signal on
  hasp's side left to cross-reference against — a related, previously unrecorded limitation this
  same reassessment surfaced.
- Chunk [M3.6.4](roadmap.md#56-m36--investigation), which wires `PassphraseGate` and the agent
  source together at a shared call site, implements exactly one passphrase-recoverable fact
  (`aws-created-rsa`) rather than two — a smaller implementation surface than §18 previously
  described, not a smaller one than [T39](#t39)'s own mechanics (one prompt per invocation, degrade
  without a TTY) required regardless.

---

<a id="t50"></a>
## T50 — `find`'s comparison folds case and separator punctuation at compare time; `fpscheme.Normalize` stays exactly as it is

**Date:** 2026-09-10 · **Status:** Accepted

### Context

[J2](design.md#7-journeys) promises *"Punctuation and case shouldn't matter,"* and
[D20](decision-log.md#d20) widens that promise to span every registered fingerprint scheme rather
than only spellings of one. [T36](#t36) specifies `fpscheme.Normalize`'s own transformations —
strip colons and internal whitespace, lowercase hex digits, strip base64 `=` padding, tolerate a
`SHA256:`/`MD5:` prefix — and `internal/adapter/fpscheme/normalize.go` implements exactly that,
deliberately **not** lowercasing base64: base64 is case-sensitive, so folding its case would
silently change which bytes a SHA-256 fingerprint names, not merely how it is spelled. That
implementation is correct on its own terms — a normalized value is also the value compared and
displayed, and corrupting it would be a real, not cosmetic, defect.

But J2's promise and base64's case-sensitivity are individually correct and jointly
unsatisfiable as a single-function contract: `hasp find key sha256:T4RN-6Go9-uzGH` (this
project's own committed `testdata/script/key-find.txtar` case, predating this chunk) must match
the key whose real fingerprint is `SHA256:t4rn6Go9uzGHCyff1lwYWtX+swf+EQhuqmE2dHSqkFE` — different
case throughout, and separator hyphens (`internal/cli/fullinventory_test.go`'s own case,
`T4RN-6go9`, is the same shape) that never appear as real content in any value a registered scheme
actually computes. `fpscheme.Normalize`, applied faithfully to [T36](#t36)'s spec, preserves both
the case difference and the hyphens verbatim — neither is a colon, whitespace, padding character,
or scheme prefix — so a direct comparison against `Normalize`'s own output silently fails on both
committed cases at once, a regression introduced when `internal/app.FindKeys` (M3.6.3's WIP) moved
from an older, single-scheme normalizer that folded both away unconditionally.

### Decision

Leave `fpscheme.Normalize` exactly as [T36](#t36) specifies it — its case- and shape-preserving
behavior remains correct for what it is actually for: [T36](#t36)'s own `CandidateSchemes` shape
routing needs an exact, uncorrupted length and alphabet to route a full-length clue to the right
candidate scheme, and a scheme's own `Compute` output must not be silently rewritten either.

`find`'s own comparison step folds **both** operands — a candidate scheme's computed value and the
normalized clue — down to lowercase letters and digits only, immediately before the substring
check, in a helper local to `internal/app.FindKeys` (`compareFold`) rather than inside
`fpscheme.Normalize` itself. This folds away case, exactly as J2 promises, and — necessarily
broader than the case tension alone — separator punctuation such as a hyphen a human inserts
purely for readability, since none of [T35](#t35)'s four registered schemes ever emits `-` or `_`
as real content: the two colon-hex schemes use only hex digits and colons, and the SSH-native
scheme emits standard, not URL-safe, base64 (`+`, `/`).

### Rationale

The residual risk this fold accepts is a theoretical false positive: two distinct base64
fingerprints, or two distinct values otherwise, differing only in a fold this comparison forgives.
Across a 43-character base64 digest or a 32/40-digit hex digest that is effectively impossible, and
even if it occurred, `find` returns a list the user disambiguates rather than acting unattended —
whereas a false *negative* is exactly the silent miss [D20](decision-log.md#d20) exists to
eliminate. The asymmetry favours folding, the same judgment [P10](design.md#4-principles)'s
confidence vocabulary already makes structurally: `find`'s match evidence is reported as
`confirmed` or `possible`, never asserted beyond what the evidence supports, so a comparison that
occasionally over-matches is caught by the very origin-evidence distinction [T36](#t36) already
requires the renderer to preserve, not silently trusted as certain.

Folding at the comparison step, not inside `Normalize`, keeps [T36](#t36)'s shape-routing
contract — and every existing test asserting `Normalize`'s and `CandidateSchemes`' exact output —
intact and unchanged; this is an additive change to how `find` compares two already-normalized
values, not a redefinition of what "normalized" means.

### Consequence

- `internal/app.FindKeys`'s match comparison uses `compareFold` on both the candidate scheme's
  computed value and the normalized clue, in place of a direct substring check against
  `fpscheme.Normalize`'s own output.
- `tdd.md` §18's "Multi-scheme `find`" subsection gains a sentence stating that `find`'s own
  comparison step folds case and separator punctuation, distinct from `Normalize`'s own,
  narrower, shape-preserving transformations.
- `internal/cli/testdata/script/key-find.txtar`'s and `internal/cli/fullinventory_test.go`'s
  existing mangled-clue assertions — both predating this chunk — pass unmodified; neither needed a
  golden-value change, only this fix to what compares them.
