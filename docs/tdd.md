---
Status: DRAFT
DateCreated: 2026-08-28
Related:
  - "[`docs/design.md`](design.md)"
  - "[`docs/decision-log.md`](decision-log.md)"
  - "[`docs/tech-decision-log.md`](tech-decision-log.md)"
  - "[`docs/project-assessment-2026-08.md`](project-assessment-2026-08.md)"
---

# hasp — Technical Design

> **Scope of this document.** `docs/design.md` is upstream and says nothing about language,
> storage, parsing, or libraries, by design — those are downstream decisions, derived from it,
> not smuggled into it. This is that downstream document. It answers, concretely, how a Go
> program satisfies every principle (P1–P9) and decision (D1–D15) `design.md` states, and it
> closes the items `design.md` §10 named as deferred and load-bearing.
>
> This document does not amend `docs/design.md` or `docs/decision-log.md`. Where a technical
> requirement reaches new design territory — as the settings file in §8 does — the gap is closed
> here and a recommendation to ratify it upstream is stated explicitly, not smuggled in as an
> implementation detail. Every technical decision below cites the principle, decision, or journey
> it serves; the reasoning behind each one, including alternatives rejected, lives in
> `docs/tech-decision-log.md` under a `Tn` ID.
>
> No project-management content appears here — no estimates, no ordering as schedule. `M1`–`M4`
> (`design.md` §9) are referenced only as scope anchors: "this applies from M2 onward" states a
> boundary of applicability, not a plan.

---

## 1. Scope — what this document closes, and what remains upstream

`docs/design.md` §10 named five items as explicitly deferred, plus one item already settled by
`D12`. This document's job is to close every one of them that is genuinely a technical question,
and to say plainly which ones it does not close.

| `design.md` §10 item | Closed here | Where |
| --- | --- | --- |
| Implementation stack | Yes | §2 |
| Configuration parsing | Yes | §6 |
| The metadata format | Yes | §7 |
| Interface — command surface, output formats | Yes | §9, §10 |
| Multi-directory / non-default locations | **No** — remains deferred | §15 |
| ~~Storage~~ | Already closed by D12; nothing to add | — |
| Cross-machine synchronization | Not addressed — still "deliberately undecided," `design.md` §3.3 | — |

Two items get their own dedicated treatment because they are the subtlest technical decisions
in this document, not footnotes to a section that closes something else:

- **The settings file (§8)** reaches the last rung of P9's taxonomy ladder — the first case that
  has. It is handled as a design decision in its own right, with a hard admission rule and a
  recommendation to ratify it upstream as `D16`.
- **`new key`'s passphrase handling (§9, §11)** is held to the same standard: no existing
  sentence in `design.md` licenses hasp asking for a passphrase, §3.2 states the opposite as a
  hard boundary, and this document says so plainly rather than construct a reading that isn't
  there — with its own recommendation to ratify upstream as `D17`.

**The `Plan` type (§4)** is the architectural keystone this document is organized around: D6's
"preview is the default for every write" is only structural, rather than a flag bolted onto each
command, if change is represented as data before it is applied. Every later section — the
derivation pipeline, config parsing, safety model — assumes `Plan` exists.

---

## 2. Stack — Go, stdlib-first, and a dependency budget

**Go 1.26.6, `linux/amd64`**, installed via `mise` (`go = "latest"`). Dev host is x86 Arch Linux;
targets are Linux and macOS, per `design.md` §3.1's "one laptop" scope (P8) extended to the two
platforms the author actually uses.

**stdlib-first.** P8 — *"the scale is one laptop... any design justified by performance,
concurrency, or scale is almost certainly solving a problem hasp does not have"* — applies
equally to dependency weight. Every third-party package below earns its place with a specific,
named requirement it satisfies that the standard library does not; every package considered and
rejected is listed with why.

### Dependency budget

| Dependency | Used for | Justified by |
| --- | --- | --- |
| `golang.org/x/crypto/ssh` | Key parsing, classification, fingerprinting, generation, marshaling | [T1](tech-decision-log.md#t1) |
| `golang.org/x/term` | Interactive passphrase prompt with no local echo (`ReadPassword`) | [T6](tech-decision-log.md#t6) |
| `github.com/spf13/cobra` | Command tree, flag parsing, shell completions, man page generation | [T3](tech-decision-log.md#t3) |
| `github.com/pelletier/go-toml/v2` | Decoding `settings.toml` and metadata fragments | [T7](tech-decision-log.md#t7), [T10](tech-decision-log.md#t10) |
| `text/tabwriter` (stdlib) | Human-readable, one-screen tabular output (P7) | — |

`go-toml/v2` is chosen over `BurntSushi/toml` on a routine basis, not a C6-weight one: hasp only
ever **decodes** TOML — settings are hand-authored by the user, and metadata fragments are
synthesized line by line ([T10](tech-decision-log.md#t10)), never round-trip-encoded as a whole
document — and `go-toml/v2`'s decode-only-shaped API and maintenance activity fit that usage
directly.

**Test-only, never shipped:**

| Dependency | Used for |
| --- | --- |
| `github.com/rogpeppe/go-internal/testscript` | End-to-end `.txtar` command-line tests (§12) |
| `testing.F` (stdlib `go test -fuzz`) | The CST round-trip fuzz test (§12); no external fuzzing library |

### Explicit non-dependencies

| Not used | Why |
| --- | --- |
| Any Go `ssh_config` library | [T2](tech-decision-log.md#t2) / `F2` — none offers a round-trip fidelity contract P2 requires |
| `libmagic` / any cgo | [T1](tech-decision-log.md#t1) — the full key-format matrix is readable in pure Go |
| A database or ORM | D12 — hasp persists nothing |
| `spf13/viper` | [T6](tech-decision-log.md#t6) — the flag → env → settings → default chain is small enough (roughly 50 lines) to own directly, and owning it keeps the precedence rule auditable in one place rather than behind a general-purpose configuration library's own merge semantics |
| An XDG-paths library | [T7](tech-decision-log.md#t7) — `os.UserConfigDir()` already does this |
| `ssh-keygen`, or any subprocess, for key operations | [T1](tech-decision-log.md#t1) |

---

## 3. Architecture — DDD adapted to a stateless CLI

Four layers, dependency direction strictly inward — nothing in `internal/domain` imports
anything outside the standard library, checked mechanically rather than left to convention
([T13](tech-decision-log.md#t13)):

```text
cmd/hasp/                    main(); resolves --key-dir/env defaults; maps errors -> exit codes (§10)
internal/domain/              Key, Host, Profile, HostGroup, Binding, value objects; zero 3rd-party imports
internal/app/                 one use case per verb x noun cell (§9); owns the Plan type (§4);
                               declares ports as interfaces at the point of consumption
internal/adapter/
  scan/                       filesystem walk -> raw facts (§5)
  keyfile/                    x/crypto/ssh wrapper: parse, classify, fingerprint, generate, marshal
  sshconfig/                  the CST: Parse/Render, marked regions (§6)
  settings/                   TOML settings reader, read-only (§8)
  backup/                     BackupStore: snapshot into ~/.ssh/.hasp-backups/ (§11)
  fswrite/                    atomic writer, symlink-through, mode preservation (§11)
internal/cli/                 cobra command tree (§9), flags, human + JSON renderers (§10)
```

### Domain types (sketch)

```go
package domain

type Fingerprint string // ssh.FingerprintSHA256 output; "" means undecidable (§5.1, T1)

type KeyName string      // the stable handle; survives rename (§5, T12)

type ProfilePath []string // ["work", "foobarco"] for the profile "work.foobarco"

func (p ProfilePath) String() string { return strings.Join(p, ".") }

type KeyFormat int

const (
    FormatUnknown KeyFormat = iota
    FormatOpenSSH
    FormatPKCS1
    FormatPKCS8
    FormatSEC1
    FormatDSA
)

type Key struct {
    Identity      KeyIdentity   // fingerprint or path (§5, T12)
    Locations     []string      // absolute paths; >1 when aliased (D5)
    Name          KeyName
    Algorithm     string        // "ssh-ed25519", "ssh-rsa", ...
    Bits          int           // 0 when not applicable
    Comment       string        // "" if unreadable
    Format        KeyFormat
    Encrypted     bool
    HasPublicHalf bool
    Profiles      []ProfilePath // derived, never declared (D13); may be empty
}

type BindingKind int

const (
    BindingExplicit       BindingKind = iota // from an IdentityFile line
    BindingImplicitDefault                   // one of ssh_config(5)'s default identity paths (§5, T16)
)

type Binding struct {
    Key  KeyIdentity
    Kind BindingKind
}

type Host struct {
    Patterns   []string          // the "Host" line's arguments
    HostGroup  string            // which host-group file this stanza lives in (D9)
    Managed    bool              // inside hasp's markers? (D7, D14)
    Bindings   []Binding         // explicit and implicit-default, resolved and labelled (§5, T16)
    Profiles   []ProfilePath     // derived from Bindings (T5)
}

type Profile struct {
    Path     ProfilePath
    Dir      string // absolute path; ProfilePath joined onto the key directory
    Managed  bool   // carries a .hasp marker? (§5.3, D13, D15)
}
```

### DDD deviations from Cosmic Python — summary

Full argument and consequences: [T13](tech-decision-log.md#t13). Two deviations, each tied to a
specific principle already in force, not to "DDD is inconvenient here":

- **No Unit of Work, no repository-with-a-session** — nothing is persisted (D12), so there is no
  transaction to own. `Plan`/`Applier` (§4) is the Go-appropriate stand-in, and is explicitly
  *not* a UoW: it gives no cross-aggregate rollback, only a per-`Change` backup-then-write
  guarantee.
- **No message bus, no domain events** — P8's *"a single human operating interactively"* means
  there is no second consumer for an event to reach. Cross-cutting effects are ordinary function
  calls inside the use case building the `Plan`.

Ports are declared **at the point of consumption**, in `internal/app`, per use case — Go's
"accept interfaces, return structs" idiom — rather than centralized in a separate abstractions
module:

```go
// internal/app/newkey.go
type keyFileWriter interface {
    WriteKeyFile(path string, contents []byte, mode fs.FileMode) error
}

type NewKeyUseCase struct {
    Now func() time.Time // injected clock; no direct time.Now() call anywhere in app or domain
}

func (uc NewKeyUseCase) Plan(req NewKeyRequest) (Plan, error) { /* ... */ }
```

**The aggregate boundary differs by direction.** Reads use one read-model aggregate — a single
derived snapshot of "the machine," built once per invocation, covering keys/hosts/profiles/
bindings together — justified by P8 (cheap at this scale) and *required* by P7 (a one-screen
answer to "which hosts use this key" needs the join computed once). Writes are file-scoped: one
`Change` per file, each independently backed up, with **no cross-file transaction** — POSIX
offers no atomic multi-file rename, and this document states that plainly rather than implying
otherwise.

---

## 4. The `Plan` model — change is data before it is applied

This is the load-bearing idea the rest of the write path depends on. D6 requires preview to be
the default for *every* write, uniformly, with no per-operation judgment call about which changes
are risky enough to deserve it. P5 states this is a modeling constraint from the start, not an
interface flag: *"the ability to preview is not a convenience flag bolted on later; it
constrains how change is modeled from the beginning."*

A design where a use case performs I/O and reports what it did cannot satisfy this — the preview
has to exist *before* anything happens, which means the thing being previewed must already be a
value. Full type sketch and rationale: [T4](tech-decision-log.md#t4). The essential shape:

```go
type Plan struct {
    Summary string
    Changes []Change // ordered — see "Change ordering" below
}

type Change interface {
    Preview() Preview        // structured preview; consumed by both renderers (§10). T26.
    RequiresBackup() bool
    Apply(fsys WriteFS) error
}

type Preview struct {
    Summary string     // one-line description, always present
    Diff    []DiffLine // a line-oriented diff against the prior bytes; nil where there is no
                        // natural "before" (MoveFile, CreateSymlink, Remove), or where diffing
                        // would expose secret material — see WriteKeyFile below
}

type DiffLine struct {
    Kind DiffKind // DiffContext, DiffAdded, DiffRemoved
    Text string
}

// Concrete Change kinds:
type MoveFile struct{ From, To, Reason string }

type WriteRegion struct {
    File   string
    Marker RegionID
    Before []byte // the region's current rendered bytes, captured as a *read* when the Plan
                   // was built (P5 — reads are always safe) so Preview() has something to diff
    After  []byte // the region's fully regenerated replacement (§6 "Rigid regeneration")
}

type CreateSymlink struct{ Path, Target string }

type CreateMarker struct{ Dir string; Header []byte }

type Remove struct{ Path, Reason string } // e.g. release profile's `.hasp` removal (§9);
                                           // always RequiresBackup() == true (P4) — nothing
                                           // hasp deletes is deleted without a prior copy

type WriteKeyFile struct {
    Path           string
    Contents       []byte
    Mode           fs.FileMode
    AllowOverwrite bool // false for `new key`, always. true only for `edit key
                         // --replace-material` ([T22](tech-decision-log.md#t22)), whose
                         // RequiresBackup() is unconditionally true so the material being
                         // replaced is snapshotted to ~/.ssh/.hasp-backups/ before Apply ever
                         // reaches the rename — P4 satisfied for the one legitimate overwrite
} // the sole D14 exception to "hasp never writes to a private key file." When AllowOverwrite is
  // false, Apply fails closed — returns ErrKeyFileExists — the instant Path already exists,
  // before a temp file is even written; hasp never overwrites key material it did not just
  // verify it authored in this same Plan (§11, D4). Preview() always returns a Summary only,
  // Diff == nil, regardless of AllowOverwrite: hasp does not render private key bytes to a
  // terminal or a log, even bytes it is about to write itself.
```

**Change ordering.** `Plan.Changes` is not built in whatever order is convenient — it is ordered
so that **any prefix of it leaves the machine in a working state, with the least recoverable
change last.** A `Plan` that fails partway through should fail into a state at least as good as
the one it started from, never a worse one. `adopt key` is the worked example, because it is the
one operation most exposed to this: it both moves a key file and must leave a top-level alias
behind so `ssh`'s default identity probing keeps finding it (D13). The naive plan,
`[MoveFile, CreateSymlink]`, violates the rule — if the move succeeds and the symlink fails, the
key sits in its profile directory with no top-level name at all, and the least recoverable
failure (losing discoverability) happened last. Simply reversing the order does not fix it
either: `os.Rename`'s replace-on-conflict semantics mean creating the symlink *before* the move
would silently destroy the original file the symlink is supposed to stand in for — worse, not
better. The correct sequence, detailed in [§11](#11-safety-model--declared-failure-stances), is
D4's move rule refined for the alias-preserving case: copy, verify, then atomically **replace**
the source with a symlink to the destination — never a bare unlink — so every prefix of the
sequence leaves a real, readable key at the original path.

Every write-verb use case has the signature `Plan(req Request) (Plan, error)`. It may read the
machine to compute the plan — P5 guarantees reads are always safe — but returns before writing
anything. A single `Applier` is the only thing in the codebase that executes a `Plan`:

```go
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

### The command-loop shape this produces

1. The `cli` layer parses flags into a `Request` value — pure data, no I/O.
2. `Request` goes to a `UseCase.Plan(req)`, which reads the machine and returns `(Plan, error)`.
3. The `cli` layer calls `Preview()` on every `Change` and renders the result — this render call
   **is** the preview (P5); it happens whether or not the write proceeds. §10 states how the
   human and JSON renderers each present a `Preview`, including its optional `Diff`.
4. If `--yes` was not passed, the CLI checks whether stdin is a TTY. If it is, it prompts to
   confirm. If it is not, and `--yes` was not passed, hasp fails closed (§11) rather than guess.
5. On confirmation, `Applier.Apply(plan)` runs — backing up, then writing, per `Change`, in order.

**The resulting inversion is worth stating in exactly these terms: `--dry-run` does not exist,
because preview *is* the default.** The flag that exists is `--yes`; its *absence* is what
triggers the preview-only path, not its presence triggering an extra one.

A `Plan` with zero `Changes` is a legitimate, renderable value — "nothing to do" is reported
through the identical preview path as a real change, so there is no separate no-op branch to
keep in sync with the real one.

---

## 5. Derivation pipeline — scan, classify, resolve, project

Every read is a fresh scan (§6.3, D12) in four steps: **scan** the filesystem for raw candidates,
**classify** each one, **resolve** the relations between them, **project** the answer to the
question actually asked (P7 — the projection is the *useful* answer, never a raw data dump by
default).

### Classify — restating the D13 symmetry

> directory = profile · file = host group · symlink = alias · key file = key · config stanza =
> host, and nothing else exists

| On-disk shape | Classified as | Managed when |
| --- | --- | --- |
| A directory under the key directory | Profile container, or a `Profile` if it carries `.hasp` | `.hasp` present (D13, D15) |
| A regular file matching `*.sshconfig`, or `~/.ssh/config` | Host group (D9) | N/A — a container, not itself managed/unmanaged |
| A symlink whose target is a key file | Alias (D5) | Follows the target key's status |
| A file whose content parses as a private key (`x/crypto/ssh`) | Key | Managed iff its directory is a managed profile (D14 — location is the only signal) |
| A `Host` stanza inside a host-group file | Host | Managed iff inside one of hasp's marked regions (D7, D14) |

### The key identity rule

Full argument: [T12](tech-decision-log.md#t12). Identity is the **fingerprint when derivable**,
else the **canonical (symlink-resolved, absolute) path**:

```go
type KeyIdentity interface{ identityKey() string }

type byFingerprint Fingerprint
func (f byFingerprint) identityKey() string { return "fp:" + string(f) }

type byPath string // resolved, absolute
func (p byPath) identityKey() string { return "path:" + string(p) }
```

Two files sharing a fingerprint are **one key with two locations** (`design.md` §5.1). A key
that is legacy PEM, encrypted, and missing its public half ([T1](tech-decision-log.md#t1)'s
undecidable row) is identified by path alone, and **can never be deduplicated** against another
undecidable key — hasp cannot prove two such files are the same secret, and P1 forbids reporting
a fact it cannot verify. `check` reports this as its own advisory finding — *"possible duplicate,
cannot confirm"* — distinct from a confirmed duplicate (two fingerprint-identified files sharing
a fingerprint).

Renaming an undecidable key's file therefore changes its path-based identity, by definition —
this is why rename ([J6](design.md#7-journeys)) is implemented as a `MoveFile` addressed by the
key's stable `KeyName` handle, never by re-deriving identity after the move and expecting it to
match. `KeyName` is the stable handle across a rename; `KeyIdentity` changing for the `byPath`
case is the *correct*, expected consequence of the move, not a defect.

### Cross-profile membership is realized by alias location

Full argument: [T19](tech-decision-log.md#t19). D2 states a key may belong to more than one
profile, but D13 puts a key file in exactly **one** directory — a key cannot simultaneously *be*
two files. The mechanism that reconciles the two is already ratified and was never spelled out
as D2's write path: **`Key.Profiles` is the union of the profiles of every directory in
`Key.Locations` (§3), not just the one the file physically occupies.** An alias (D5) is a
symlink, and a symlink can live in a different profile directory than its target. Creating one
there — `hasp edit key "id_ed25519_foobarco" --add-alias=personal/id_ed25519_foobarco`, the argument
itself profile-qualified so it is unambiguous with the unrelated `--profile` flag that *moves* a
key's canonical file (§9) — is how a user realizes the state D2 declares legitimate: the real
file stays in `work.foobarco`, a symlink
appears in `personal`, `Locations` gains the symlink's path, and the next scan derives
`Profiles = [work.foobarco, personal]` from that union — computed, not declared, exactly as D12
requires. `edit key --add-alias` (§9) was previously framed only as a naming convenience for
tools that hardcode `id_rsa`; it is that, **and** it is D2's only write path, and both readings
are the same mechanism at different call sites.

### Binding resolution

A host's `IdentityFile` values are resolved to canonical key locations through, in order:
tilde expansion, then OpenSSH token expansion, then relative-path resolution, then symlink
resolution.

**Tokens.** Verified against `ssh_config(5)`'s own `TOKENS` section: `IdentityFile` accepts
`%%`, `%C`, `%d`, `%h`, `%i`, `%j`, `%k`, `%L`, `%l`, `%n`, `%p`, `%r`, and `%u`. hasp resolves
the four the predecessor's own README scope names — the rest are left as literal text if they
appear, which is itself a `check` finding ("unresolvable `IdentityFile` token"):

| Token | Real `ssh_config(5)` meaning | hasp's resolution |
| --- | --- | --- |
| `%d` | Local user's home directory | `os.UserHomeDir()` |
| `%h` | The remote hostname | The stanza's `HostName` directive if set, else the literal `Host` pattern — matching OpenSSH's own fallback. Defined only for a stanza with a single, non-wildcard pattern; a wildcard or multi-pattern stanza has no single `%h` value, and hasp reports the binding as unresolvable rather than guess (fail-open reporting, §11) |
| `%r` | The remote username | The stanza's `User` directive if set, else the local username (`%u`) — OpenSSH's own fallback rule |
| `%u` | The local username | `user.Current().Username` |

**A deliberate, stated divergence from real `ssh`.** OpenSSH resolves a bare relative
`IdentityFile` path (no leading `~`, no leading `/`) relative to the **current working directory
of the `ssh` process at connection time** — a behavior that is fragile even for `ssh` itself, and
meaningless for hasp, which is never the process making the connection and has no analogous
"working directory" to inherit. hasp instead resolves a bare relative path **relative to the key
directory** (`--key-dir`, default `~/.ssh`), because that is what essentially every real-world
relative `IdentityFile` value in a hand-written config actually means in practice. This is a
judgment call, not a transcription of OpenSSH's own rule — see §15.

**Symlinks and dangling targets.** The resolved path is passed through `filepath.EvalSymlinks`.
A target that does not exist is not a hard failure — it is reported as an unresolved binding and
surfaces as a `check` finding (§11's fail-open rule for scanning: *"unreadable key file mid-scan
→ fail-open, report and continue"* extends naturally to an unreadable/missing binding target).

**Multiple `IdentityFile` lines** on one stanza all contribute — real `ssh_config` tries each in
sequence, and hasp's binding set is the union, exactly mirroring the multi-profile union rule in
[T5](tech-decision-log.md#t5).

### Implicit default-identity probing

Full argument: [T16](tech-decision-log.md#t16). A stanza with **no** `IdentityFile` line — the
most common shape of a hand-written stanza — is not a stanza with zero bindings. Real `ssh`
still tries its own built-in default identity files, verified against `ssh_config(5)` on this
host (`OpenSSH_10.5p1`):

> The default is `~/.ssh/id_rsa`, `~/.ssh/id_ecdsa`, `~/.ssh/id_ecdsa_sk`, `~/.ssh/id_ed25519`,
> `~/.ssh/id_ed25519_sk` and `~/.ssh/id_mldsa44_ed25519`.

(Note what is absent: `id_dsa` is not in this list — modern OpenSSH dropped it from the
defaults, though a key at that path is still a legitimate, readable artifact under
[T1](tech-decision-log.md#t1) and [T23](tech-decision-log.md#t23).)

hasp resolves a `Binding{Kind: BindingImplicitDefault}` for every one of those six paths that
exists in the key directory and is not suppressed. Exactly one thing suppresses them:
**`IdentityFile none`** — `ssh_config(5)`: *"an argument of `none` may be used to indicate no
identity files should be loaded."* **`IdentitiesOnly yes` does *not* suppress them** — verified
against the same man page, and worth stating precisely because it is a common misreading:
`IdentitiesOnly` restricts which identities an *agent* is allowed to offer beyond *"the
configured authentication identity and certificate files (either the default files, or those
explicitly configured...)"* — the default files are explicitly named as still in scope. hasp
does not model agent-offered identities at all (§3.2 — hasp is not an agent), so
`IdentitiesOnly` has no effect on hasp's binding resolution and is not treated as a suppressor.

Explicit and implicit-default bindings are **labelled, never merged into one undifferentiated
set** — `Binding.Kind` carries the distinction into `internal/domain` (§3) and into the JSON
envelope (§10), because a consumer needs to tell "you wrote this" from "`ssh` would fall back to
this" apart, and because `check`'s "bound to no key" finding (§9) must fire only when **neither**
kind resolves to anything.

### Host profile membership (closes the gap D13 left for hosts)

Full argument: [T5](tech-decision-log.md#t5). A host's profile set is the union of the profile
sets of the keys its bindings resolve to — computed, never declared, and correct for unmanaged
hosts as well as managed ones, which is what lets M1's read-only survey report it before `adopt`
exists at all.

---

## 6. Configuration parsing — a lossless CST for a file hasp does not fully own

Full survey and rejection of existing libraries: [T2](tech-decision-log.md#t2). No Go
`ssh_config` library offers the round-trip fidelity contract P2 requires — `kevinburke/ssh_config`
states in its own README that `Match` is unsupported and that it only *"attempts to preserve
comments."* hasp owns a small, line-oriented Concrete Syntax Tree instead.

### CST node types

Full argument for the `Directive` shape below: [T17](tech-decision-log.md#t17). `ssh_config(5)`
states two facts the original type sketch missed, both verified against the man page this
session and both load-bearing for the round-trip contract below: *"Configuration
directives are separated from their values by whitespace or exactly one '=' character (which
may be surrounded by whitespace)"* — so `Port 22`, `Port = 22`, and `Port=22` are all valid and
byte-different — and *"'#' outside of a quoted string may be used to add a comment to the end of
a line"* — so a tokenizer that treats every `#` as a comment start breaks on the first
`ProxyCommand` argument or ad-hoc value containing a literal `#` inside quotes. A `Directive`
type that stores only a parsed `Keyword` and `Args` cannot reconstruct either of these — it would
normalize `Port=22` to `Port 22` on write, a P2 violation the moment a real config uses that
style. The type below stores the raw bytes and exposes parsing as a read-only accessor instead:

```go
package sshconfig

type Node interface{ node() }

// Line is an unrecognized or purely human line: reconstructed byte-exact from its parts.
type Line struct {
    Raw        []byte // full line content, not including the terminator
    Terminator []byte // "\n", "\r\n", or nil for a final line with no trailing newline
}

type Directive struct {
    LeadingTrivia []byte // blank lines/comments immediately preceding this line, verbatim
    Keyword       string // as written; ssh_config keywords are case-insensitive and hasp does
                          // not normalize a human's casing outside a region it owns
    RawValue      []byte // everything between the keyword and the trailing comment/terminator —
                          // separator ('=' or whitespace), quoting, internal spacing, untouched
                          // by Parse. Synthesized fresh only when hasp authors the line itself,
                          // inside a region it owns (D7 elaboration 1) — there is no prior human
                          // formatting to preserve there
    Trivia        []byte // a trailing inline comment, if any, including its leading whitespace;
                          // a '#' inside a double-quoted RawValue span is never treated as this
    Terminator    []byte // "\n", "\r\n", or nil
}

// Args parses RawValue into tokens on demand, honoring ssh_config(5)'s quoting rule: a
// double-quoted span may contain whitespace and is not split on it. Used by hasp's semantic
// logic (§5's binding resolution, §9's check findings) — never consulted by Render, which only
// ever replays RawValue's bytes.
func (d Directive) Args() []string

type HostBlock struct {
    Patterns      []string
    Directives    []Node // Directive for recognized keywords, Line for everything else — never an error
    LeadingTrivia []byte // blank lines/comments immediately preceding "Host"
}

type MarkedRegion struct {
    Begin, End []byte // the exact marker lines hasp wrote, e.g. "# >>> hasp:managed >>>"
    Body       []Node // hasp-owned; regenerated wholesale on write, D7 elaboration 1.
                       // May contain MetadataLine (§7) alongside Directive/HostBlock/Line.
}

// MetadataLine is a `#:hasp key = value` line inside a MarkedRegion body (§7) — its own node
// type rather than a re-scanned Line (T25).
type MetadataLine struct {
    Key, Value string
    Terminator []byte
}

// MarkerDefectKind enumerates the detectable ways a marker pair can be malformed (T18).
type MarkerDefectKind int

const (
    DefectUnmatchedBegin   MarkerDefectKind = iota // a begin marker with no matching end
    DefectUnmatchedEnd                             // an end marker with no matching begin
    DefectDuplicateBegin                           // a second begin marker before the first is closed
    DefectEndBeforeBegin                           // an end marker with no begin preceding it
    DefectNestedInHostBlock                        // a marker line found inside a Host stanza's directive list
)

type MarkerDefect struct {
    Kind   MarkerDefectKind
    Line   int    // 1-based line number in the input, for a human-facing report
    Detail string
}

type File struct {
    Nodes         []Node         // ordered: Line | HostBlock | MarkedRegion
    MarkerDefects []MarkerDefect // populated at parse time; parsing itself never errors —
                                  // a malformed marker is data, not a failure (see §11)
}

func Parse(b []byte) *File // no error return: parsing a byte slice cannot fail by construction —
                            // every byte lands in some node, and marker problems become
                            // MarkerDefects, not an error value
func (f *File) Render() []byte
```

**The contract, and it is fuzzed (§12):**

```text
Render(Parse(b)) == b               // for arbitrary bytes b, outside marked regions
Render(Parse(Render(r))) == Render(r) // idempotent for any region body r hasp produced
```

D7 elaboration 3 is what keeps this small: hasp models the **syntax** of the whole file
completely — every byte parses into *some* node — but the **semantics** only for the directives
it manages. Everything else is carried as an opaque `Line`, never dropped, never an error (P6).

### The natively-managed directive set

Salvaged from the predecessor's `README.rst` (F4), plus `Include`, which hasp itself writes
inside its own regions (§6, below) and must therefore understand semantically there:

`AddKeysToAgent` · `Ciphers` · `ControlMaster` · `ControlPath` · `ControlPersist` ·
`ForwardAgent` · `HashKnownHosts` · `HostKeyAlgorithms` · `HostName` · `IdentitiesOnly` ·
`IdentityFile` · `Include` · `KexAlgorithms` · `LogLevel` · `MACs` · `PasswordAuthentication` ·
`Port` · `ProxyCommand` · `PubkeyAuthentication` · `StrictHostKeyChecking` · `User` ·
`UserKnownHostsFile`

Any other directive round-trips as an opaque `Line`/`Directive` today, and becomes natively
understood later by adding it to this list — no migration of existing files is required, because
the CST already carries it, unmodified, either way.

### Marker syntax and where regions appear

This whole section is J7's practical test: *"add or change a host entry and know that everything
I had written in that file by hand is still exactly as I left it."* Two distinct cases, per D7's
own asymmetry between files hasp authors wholesale and the one file it co-owns:

- **A host group hasp creates outright** (`~/.ssh/<group>.sshconfig`, D9) carries a single header
  comment, not a delimiter pair — the *entire file* is the region, so there is nothing to
  delimit. The header states the file is hasp-managed and that hand edits will be lost on the
  next write, mirroring D15's `.hasp`-header discipline rather than inventing a second
  convention beside it:

  ```text
  # hasp:owned — this file is fully managed by hasp. Hand edits here are lost on the next write.
  # See ~/.ssh/config for how it is included.
  ```

- **`~/.ssh/config`**, the co-owned default (D7's harder case), gets an explicit begin/end pair,
  because the rest of the file is human territory:

  ```text
  # >>> hasp:managed >>>
  Include ~/.ssh/work.sshconfig
  Include ~/.ssh/personal.sshconfig
  # <<< hasp:managed <<<
  ```

**Region discovery** on every scan is a single linear pass: find the sentinel line pair, feed
everything between them back through `Parse` as the region body, and treat everything outside the
pair as opaque `Line`/`HostBlock` nodes preserved exactly. Because parsing is recursive over the
same grammar, a `MarkedRegion`'s body is not a special case for the parser — it is `Parse` called
again on a byte slice. The same pass populates `File.MarkerDefects` (§11): an unmatched or
duplicate begin marker, an end marker with no begin, or a marker line found nested inside a
`HostBlock`'s directive list are all detected here, as data attached to the `File` value, never
as a `Parse` failure.

**Rigid regeneration.** On any write touching a `MarkedRegion`, the *entire* region body is
rebuilt from the current domain state and re-rendered — never patched in place — which is what
D7 elaboration 1 means by "hasp is authoritative" inside its own markers. A hand edit placed
inside a region between scans is intentionally lost on the next hasp write; this is the trade P2
explicitly names as forfeit, not an oversight.

### Host-group composition and precedence

Full argument: [T11](tech-decision-log.md#t11). OpenSSH's own semantics are
**first-obtained-value-wins** per parameter — the first matching value for a keyword wins, later
`Include`d files or later stanzas are ignored for that keyword. hasp writes **one `Include` line
per host group it manages**, never a glob, specifically so the *position* of each line in the
marked region — controlled entirely by hasp — is the precedence order. This is not a defense
against nondeterminism: `ssh_config(5)` states plainly that `Include`'s own wildcards *"will be
expanded and processed in lexical order"* — a single glob would resolve deterministically. The
reason to avoid one anyway is that lexical filename order is not necessarily the order the user
*wants* — a group named `personal.sshconfig` should not have to be renamed `0-personal.sshconfig`
just to win a precedence fight it should never have had to enter. Explicit `Include` lines give
hasp direct control over order as a property of the `Plan` (§4), independent of how the group
files happen to be named.

Adding or removing a managed host group is always a `WriteRegion` change to this same region
(append or remove one `Include` line), never a separate edit outside markers. `check` gains a
finding directly from this semantic: a later-included group defining a `Host` pattern already
fully shadowed by an earlier group or stanza is unreachable configuration — this is the thing
first-obtained-value-wins "silently bites," named and caught rather than left for a human to
discover the hard way.

---

## 7. Metadata format — declaration inside a region, never a cache

Closes `design.md` §10's "the metadata format" item, created by D7 elaboration 3: *"anything hasp
needs to record that the format cannot express natively is written into its own region in
comment form... metadata may record what the machine cannot tell you; it may never cache what
the machine can."*

### Syntax

**Sentinel-prefixed TOML**, inside a marked region only:

```text
#:hasp <key> = <toml-value>
```

Stripping the `#:hasp` sentinel from a metadata line yields a syntactically valid single-line
TOML fragment. In the CST (§6), such a line is its own node type, `MetadataLine{Key, Value,
Terminator}`, recognized at parse time by the sentinel and stored pre-split — not left as an
opaque `Line` for something downstream to re-scan. A metadata reader collects every
`MetadataLine` in a region and feeds `Key = Value` pairs to the same TOML decoder settings
already use (§2, §8) — no bespoke grammar beyond the sentinel match `Parse` already performs.
Example, inside a host group hasp owns wholesale:

```text
# hasp:owned — this file is fully managed by hasp. Hand edits here are lost on the next write.
Host foobarco-prod
    #:hasp created_by = "new host"
    HostName prod.foobarco.internal
    User jesse
    IdentityFile ~/.ssh/work/foobarco/id_ed25519
```

Non-sentinel `#` comments hasp itself writes inside a region (section banners, for instance) are
ordinary trivia and are never parsed as data. **Sentinel presence, not comment syntax, is what
marks a line as metadata** — the same presence-not-contents discipline D15 already established
for the `.hasp` marker, reused rather than reinvented (P9).

### The line D7 draws, enforced by construction

A fingerprint is derivable ([T1](tech-decision-log.md#t1)) and may therefore **never** appear on
the right-hand side of a `#:hasp` line — writing one would be a D12 violation in exactly the
shape D15 already forbids for the `.hasp` marker. A profile assignment would, in principle, be
legitimate content for this channel — except that §5's host-profile derivation
([T5](tech-decision-log.md#t5)) already computes it from key location, removing the single
largest anticipated consumer of this mechanism before it ever needed a value.

**At the milestone scope this document covers (through M3), hasp writes no field through this
channel.** That is a deliberate, stated fact, not an oversight — the mechanism exists because D7
requires it be available; its keyset is empty because T5 closed the one gap it was built to fill.
Any future proposal to populate a field here must independently clear the same P9 bar every
taxonomy proposal must — *"show that the existing structure genuinely cannot carry the fact"* —
the same standing test `design.md` §5.7 already applies to anything hasp would have to be *told*.

No schema version field is needed while the keyset is empty; `#:hasp _schema = N` remains
available, at zero present cost, whenever a real field needs one.

---

## 8. Settings — the last rung of P9's ladder

P9's ladder ends: *"Only as a last resort, a hasp settings file — and no case has yet required
one."* `new key`'s non-interactive default-passphrase-mode requirement (§9, §11) is **the first
case that does.** This section treats that fact deliberately, per the project's own discipline
that a gap in the ratified design is amended, not reinterpreted — full argument and every
alternative considered: [T7](tech-decision-log.md#t7).

**hasp reads settings and never writes them.** D3 already anticipated this — *"hasp's own
settings are settings, are not part of the verb grid, and are edited directly"* — so there is no
`hasp config set` and no settings-mutation verb of any kind. The file is edited with an editor,
exactly as P6 already promises for everything hasp knows.

- **Format:** TOML, `#` comments — the same convention `.hasp` uses (D15), ridden rather than
  duplicated (P9).
- **Location:** `os.UserConfigDir()/hasp/settings.toml` (stdlib; no XDG library) —
  `~/.config/hasp/settings.toml` on Linux, `~/Library/Application Support/hasp/settings.toml`
  on Darwin. **Outside `~/.ssh` deliberately** — it is not key material, not arrangement, and
  must never be swept into the profile scanner or the backup rotation (§11).
- **The admission rule:** settings may hold *only* user preferences that (a) cannot be derived,
  and (b) exist to enable non-interactive execution. They may **never** hold facts about the
  machine (P1/D12 — precisely what the abandoned TOML state file held) and **never** hold
  taxonomy (P9 rungs 1–2 — a profile assignment here would be D13's rejected option (iii),
  readmitted through a different door). Anything else stays a flag.
- **v1 keyset**, deliberately minimal, because nothing else has yet cleared the bar:

  ```toml
  # ~/.config/hasp/settings.toml
  # hasp reads this file. hasp never writes to it.
  [new_key]
  default_passphrase_mode = "prompt"   # "none" | "prompt" | "stdin"
  ```

- **Optional, throughout.** A missing file is not an error; every key has a built-in default
  that makes hasp fully useful with zero configuration, preserving §6.3's *"works on first run,
  nothing configured"* exactly as written.

**Recommend an upstream `D16`** in `docs/decision-log.md`, flagged here rather than written
there — that call belongs to the ratifying author. It should record that P9's last rung was
reached, name the case that required it, and restate the admission rule above as a permanent
constraint on this file's growth, not merely this document's judgment call.

---

## 9. Command surface — the 3×8 grid, spelled out

Closes `design.md` §10's "interface" item for the parts that are design, not presentation: which
flags exist, and what each verb×noun cell does. D10 fixes the grid at 3 first-class nouns × 8
verbs; second-class nouns (alias, binding, host group) surface as flags on their owner, never as
nouns of their own.

### Global flags

| Flag | Effect |
| --- | --- |
| `--key-dir` | Key directory; default `~/.ssh`. Never defaulted inside `internal/app`/`internal/domain` — always resolved once, in `cmd/hasp`, and passed down explicitly (§12) |
| `--json` | Machine-readable output (§10) |
| `--yes` | Consent to apply a `Plan` non-interactively; see §4's inversion — there is no `--dry-run` |
| `--verbose` | Enables `log/slog` output to stderr; off by default |
| `--no-color` | Disables ANSI in the human renderer |

### The grid

`check` (J5 — *"tell me what's untidy... help me fix each one deliberately"*) is advisory
everywhere: it reports on managed and unmanaged resources alike, but repairing a finding it
raises on an unmanaged resource requires `adopt` first, exactly as J5 anticipates.

| Verb | key | host | profile |
| --- | --- | --- | --- |
| **list** | Every key found (managed + unmanaged): name, algorithm, fingerprint (or `unknown`), format, encrypted?, profiles, comment | Every host stanza across all groups, or `--group=<g>` to scope one: pattern, resolved bindings — explicit and implicit-default, labelled (§5, T16) — derived profile(s) | Every directory carrying `.hasp`, with key/host counts |
| **show** `<handle>` | Full detail for one key: identity, every location (aliases), profiles, hosts that bind it (J8) | Full stanza detail: directives, resolved `IdentityFile` targets, derived profile(s), which host group it lives in | Keys and hosts, **aggregated across child profiles by default** ([T21](tech-decision-log.md#t21)) — `show profile work` includes `work.foobarco` and `work.acme`, per J8's own offboarding example; `--no-recurse` scopes to the profile alone |
| **find** `<clue>` | Identify a key from a fingerprint fragment, normalizing punctuation and case (J2) | Match by pattern fragment, or by the name/fingerprint of a bound key | Match by name fragment |
| **new** | Generate a keypair (`ed25519` by default, [T1](tech-decision-log.md#t1)), optionally in a profile (`--profile`); passphrase flags below. Fails closed if the target path already exists — `new key` never overwrites ([T22](tech-decision-log.md#t22), §11) | Create a stanza (`--key=<name>`, `--group=<g>`) inside a marked region | Create the directory (`mkdir -p` semantics for intermediate segments) and, for the leaf, a `.hasp` marker with a `#` header (D15) |
| **edit** | Rename (`--name`); `--add-alias`/`--remove-alias`; `--replace-material=<path>`; move between profiles (`--profile`) | Change directives; move between groups (`--group`); rebind (`--key`) | Rename (moves the directory, D13); nothing else — key/host membership is edited on the key/host, not here |
| **check** | Duplicates confirmed (same fingerprint) or unconfirmed ([T12](tech-decision-log.md#t12)); missing public half; key in no profile; fingerprint `unknown` | Dangling `IdentityFile` targets; unresolvable tokens (§5); unreachable stanzas shadowed by `Include` order ([T11](tech-decision-log.md#t11)); host bound to no key — fires only when **neither** an explicit **nor** a resolvable implicit-default binding exists (§5, T16); a stanza present in more than one host group, or a stanza missing from all of them, after a `Plan` that touched more than one file only partially applied ([T13](tech-decision-log.md#t13)) | Empty profile directories; a directory that looks like a profile (holds keys) but carries no `.hasp` |
| **adopt** | Move an unmanaged key into a managed profile directory; leaves a top-level alias so default identity probing still finds it (D13) | Wrap an existing hand-written stanza in hasp's markers | Add a `.hasp` marker to an existing directory that already holds keys |
| **release** | Move a managed key back out of its profile directory (inverse of `adopt`) | Remove the stanza from hasp's markers, re-inserting it as plain text immediately after the marked region — content is never deleted, only unmanaged (see §15) | Remove `.hasp` via a `Remove` change (§4), backed up first (P4); preview shows the marker's own contents first, so removing a note the user wrote is never a silent loss (D15) |

**`new key`'s passphrase flags.** Mechanics — full argument in §11 and
[T6](tech-decision-log.md#t6):

| Flag | Effect |
| --- | --- |
| `--passphrase` | Prompt interactively (`x/term.ReadPassword`, no local echo) |
| `--no-passphrase` | Generate without one, explicitly |
| `--passphrase-stdin` | Read the secret from stdin — for non-interactive use |

Mutually exclusive. Resolution order when none is given: **flag → env
(`HASP_NEW_KEY_PASSPHRASE_MODE`) → settings (§8) → built-in default** ("prompt if a TTY is
attached, else fail closed").

**This is new design territory, not a licensed reading of an existing carve-out, and §15 records
it as such.** No sentence in `design.md` authorizes `new key` to *ask* for a passphrase — §3.2's
*"Not a passphrase manager. hasp never asks for, stores, or transmits a passphrase"* is a stated
hard boundary, and D14's *"hasp never writes to a private key file. The single exception is
generating a new key..."* licenses hasp **authoring** the file, not **prompting** about it; the
two are different acts, and conflating them would be exactly the kind of citation this project's
own culture exists to catch. The mechanics (generation-time only, never to decrypt an existing
key, never persisted, buffer zeroed) are sound regardless of the citation and are kept — but the
framing changes: this is the user's deliberate choice, made explicitly for non-interactive
operation, that **narrows a stated hard boundary**, and it is pending upstream ratification
exactly as the settings file (§8) is. Recommend an upstream `D17` alongside `D16`.

**A deliberate absence, worth naming:** there is no `--profile` flag on any `host` command that
*assigns* a profile. A host's profile membership is computed (§5, T5), not declared — the grid
has nothing to write there because there is nothing to write.

---

## 10. Output contract — stdout is data, stderr is diagnostics

Full argument: [T14](tech-decision-log.md#t14). §6.3 requires *"every read has a
machine-readable form"* — this document treats that as a public contract, since `hasp list key
--json | jq` has to keep working release over release, even though `design.md` itself does not
use that phrase.

### Exit codes

| Code | Meaning | Who returns it |
| --- | --- | --- |
| `0` | Clean — no findings, or a write completed | Any command |
| `1` | Findings — `check` surfaced issues; not a hasp failure | `check`, exclusively |
| `2` | Usage — bad flags/arguments, or a fail-closed non-interactive precondition (§11) | Any command |
| `3` | Error — hasp could not complete the operation | Any command |

Cobra never calls `os.Exit` itself; `cmd/hasp`'s `main()` maps the error `rootCmd.Execute()`
returns to one of the four codes via a small sentinel-error taxonomy. `check` is the **only**
use case permitted to resolve to `1` — no other verb ever does, so `if hasp check ...; then`
reads unambiguously across the entire surface, forever.

### Rendering a `Preview` (§4)

Full argument for why `Change` carries a diff at all: [T26](tech-decision-log.md#t26). The
human renderer prints each `Change`'s `Preview().Summary`, and where `Diff != nil`, a
unified-diff-style block beneath it — `+`/`-`/context-prefixed lines, colorized unless
`--no-color`. Where `Diff == nil` (`MoveFile`, `CreateSymlink`, `Remove`, and always for
`WriteKeyFile`) only the summary line appears — there is nothing to diff, or, for `WriteKeyFile`,
deliberately nothing shown even though bytes exist (§4, §11). The JSON renderer marshals the same
`Preview` value as part of a `plan.preview`-kind envelope: `{"summary": "...", "diff":
[{"kind": "added"|"removed"|"context", "text": "..."}], ...}`, `diff` omitted (not merely empty)
when it is `nil`, so a consumer can tell "no diff exists for this change" apart from "the diff is
empty."

### JSON envelope — versioned from the first release

```go
type Envelope struct {
    Version  int             `json:"version"`  // envelope schema version — independent of hasp's own release version
    Kind     string          `json:"kind"`      // "key.list", "host.show", "check.report", ...
    Data     json.RawMessage `json:"data"`
    Warnings []string        `json:"warnings,omitempty"`
}
```

A consumer piping through `jq` can branch on `.version` whenever it needs to, rather than only
after hasp breaks the shape without warning. Both renderers — human and JSON — are fed the **same**
internal read-model value ([T13](tech-decision-log.md#t13)'s single read aggregate): one code
path computes an answer, two display it, so the two forms can diverge in formatting but never in
content.

`log/slog` writes to stderr, off by default, enabled by `--verbose`. Nothing else ever writes to
stdout except the rendered answer.

---

## 11. Safety model — declared failure stances

One table of every guard in the system and which way it fails. Stated once, here, rather than
scattered across use cases, so an implementer never has to guess which way a new guard should
fail — the default is fail-closed unless a row here says otherwise.

| Situation | Stance |
| --- | --- |
| A write is requested with no TTY and no `--yes` | **Fail-closed** |
| `new key` runs non-interactively with no explicit passphrase-mode choice ([T6](tech-decision-log.md#t6)) | **Fail-closed** |
| `new key`'s target path already exists | **Fail-closed** — refused before any byte is written; `new key` never overwrites ([T22](tech-decision-log.md#t22), D4) |
| A backup cannot be written | **Fail-closed** — the write does not proceed |
| `File.MarkerDefects` is non-empty for a file about to be written to (§6: unmatched/duplicate begin, end-before-begin, or a marker nested inside a `HostBlock`) | **Fail-closed** for writes to that file; **fail-open** for reads (report every node that parsed cleanly, flag each defect by kind and line, §6) |
| A fingerprint is underivable | **Fail-open** — report `unknown` (D12, explicitly) |
| A key file is unreadable mid-scan | **Fail-open** — report and continue; J1 must never abort a survey over one bad file |
| An `IdentityFile` token or target is unresolvable (§5) | **Fail-open** — reported as a `check` finding, scan continues |
| A `Plan` touching more than one file fails partway through | **No rollback across files** ([T13](tech-decision-log.md#t13)) — each already-applied `Change` stands, exactly as `Preview()`ed; the resulting inconsistency (a stanza in two groups, or in none) is a `check` finding, never a silently hidden gap |

### Write mechanics

Full argument: [T15](tech-decision-log.md#t15), [T20](tech-decision-log.md#t20),
[T22](tech-decision-log.md#t22).

1. **Atomic write.** Temp file in the same directory as the target → `fsync` the temp file →
   `os.Rename` onto the target → `fsync` the containing directory (needed for the rename to be
   durable across a crash on Linux; easy to skip, easy to regret). This is the general mechanism
   every `Change.Apply` uses, `WriteKeyFile` included — see point 5.
2. **A symlinked `~/.ssh/config` is resolved and written through — never replaced.** Detected via
   `os.Lstat`, resolved via `filepath.EvalSymlinks`; the atomic-write dance in (1) runs against
   the resolved target, not the symlink. Dotfiles-managed configs make this layout common, and
   silently replacing the symlink with a plain file is destructive by omission — exactly the
   class of bug D4's spirit exists to prevent, even though D4's letter is about key material.
3. **Mode and ownership preserved** across every write — `os.Stat` the original before writing,
   apply the same mode to the temp file before the rename in (1).
4. **D4's move rule, applied literally: copy → verify → unlink source, never unlink first.**
   Verification is fingerprint re-derivation and comparison when derivable; byte-for-byte
   comparison for the undecidable case. This is the mechanism behind `edit key --profile` (a
   plain relocation, no alias required). **`adopt` refines it further**, per §4's ordering
   principle: because `adopt` must also leave a top-level alias behind (D13), the terminal step
   is not a bare unlink but an **atomic replace of the source with a symlink to the
   destination** — copy to the destination, verify, then `os.Rename` a freshly created symlink
   onto the (still-occupied) source path. `os.Rename` replaces an existing target atomically, so
   this single step both removes the now-redundant original copy and installs the alias in one
   operation; if it fails, the source path still holds a complete, valid, readable copy of the
   key (not yet deduplicated into a symlink, which is a `check` finding, and idempotently
   retryable — never a state where the top-level name resolves to nothing).
5. **`new key` and `edit key --replace-material` are covered by this same discipline, not an
   exception to it.** `WriteKeyFile.Apply` follows point 1's atomic-write mechanism for the
   bytes; whether it is allowed to reach the final rename depends on `AllowOverwrite` (§4).
   `new key` never sets it, so `Apply` refuses outright — see the table above — before a temp
   file is even created. `edit key --replace-material` sets it, and only because its `Change`
   is unconditionally `RequiresBackup() == true`: the material being replaced is copied to
   `~/.ssh/.hasp-backups/` (§12) by the `Applier`'s own backup-before-`Apply` ordering (§4)
   *before* the rename that replaces it ever runs — P4 holds for the one write that
   legitimately overwrites key material on purpose.

**Passphrase handling, restated as a safety rule, not just a feature (§9, [T6](tech-decision-log.md#t6)):**
accepted only at generation time, never to decrypt an existing key, never persisted anywhere,
buffer zeroed immediately after use. No function in `internal/domain` or `internal/app` accepts
a passphrase alongside an *existing* `Key` value — this absence is the enforcement mechanism, and
it is legible in a diff (P6), not merely documented here.

---

## 12. Testing strategy — a function of a directory

D12's gift: hasp is a pure function of a directory, so a fixture tree is the entire test setup.
The key directory is **always injected, never defaulted**, in every test — `internal/app` and
`internal/domain` constructors take `--key-dir`'s resolved value as an explicit parameter,
always. The predecessor's `ContextBorg` global singleton is named directly in
`docs/project-assessment-2026-08.md` §5.1 as *why it had zero tests*; this is the discipline that
rules that out structurally, not by convention.

### Fixture matrix

Committed, throwaway keys spanning the full matrix
[T1](tech-decision-log.md#t1)'s derivation-gap table depends on — 3 algorithms × 2 formats ×
2 encryption states × 2 public-half states = 24 fixtures, under `testdata/keys/`, named for what
they are (`ed25519-openssh-encrypted-nopub`, `rsa-pem-plain-pub`, ...):

| Axis | Values |
| --- | --- |
| Algorithm | `ed25519`, `rsa`, `ecdsa` |
| Format | `openssh`, `pem` |
| Encryption | `plain`, `encrypted` |
| Public half | `pub`, `nopub` |

**Plus 4 DSA fixtures**, PEM-only ([T23](tech-decision-log.md#t23)): `x/crypto/ssh`'s
`ParseRawPrivateKey` genuinely parses DSA (*"It supports RSA, DSA, ECDSA, and Ed25519..."*), and
a decade-old DSA key is exactly the kind of cruft `design.md` §2's motivating inventory
describes, even though that specific inventory happens not to include one. `ecdsa`/`openssh`-mode
generation is skipped for DSA, matching how DSA keys actually exist in the wild. Generation
friction is real and worth stating: verified this session, `ssh-keygen -t dsa` on
`OpenSSH_10.5p1` fails outright — `unknown key type dsa` — because OpenSSH removed DSA from
`-t`'s accepted values. The DSA fixtures cannot come from `ssh-keygen` at all; they are
constructed directly via Go's `crypto/dsa` (present in stdlib, documented as legacy) plus
`encoding/pem`, not shelled out.

These 28 fixtures are generated **once**, at fixture-authoring time, by a small dev-only script
that may shell out to `ssh-keygen` for the non-DSA cases — this is a development-time
convenience, not a hasp runtime dependency, and does not contradict [T1](tech-decision-log.md#t1)'s
"no subprocess" rule, which governs what hasp itself execs when it runs, not how its test
fixtures were produced once.

**Plus a small set of shape fixtures for the CST** ([T18](tech-decision-log.md#t18),
[T24](tech-decision-log.md#t24)), committed under `testdata/sshconfig/`: a CRLF-terminated
config, a config with no trailing newline on its final line, and a genuinely empty (0-byte)
file — each must still satisfy `Render(Parse(b)) == b` and must never produce a `MarkerDefect`
by itself.

### `testscript` end-to-end cases

`rogpeppe/go-internal/testscript` `.txtar` files fit "pure function of a filesystem" almost
exactly: each case sets up a synthetic `HOME`/key-dir via `env` directives, runs `hasp` commands,
and asserts against golden `stdout`/`stderr`. This is where the full read → preview → confirm →
write cycle (§4) gets exercised end to end, including exit codes (§10).

### The CST fuzz test

```go
func FuzzSSHConfigRoundTrip(f *testing.F) {
    // seed corpus: real-world ssh_config snippets, committed under testdata/sshconfig/
    f.Fuzz(func(t *testing.T, b []byte) {
        got := sshconfig.Render(sshconfig.MustParse(t, b))
        if !bytes.Equal(got, b) {
            t.Fatalf("round-trip mismatch:\n got: %q\nwant: %q", got, b)
        }
    })
}
```

This is [T2](tech-decision-log.md#t2)'s contract, tested directly rather than asserted in prose.

### Guard tests

- **"hasp writes nothing outside the key directory."** Every test redirects `HOME` to a
  `t.TempDir()` via `t.Setenv`, runs the operation under test, then diffs a pre/post snapshot of
  the entire temp `HOME` tree against an allowlist containing exactly the key-dir subtree
  ([T8](tech-decision-log.md#t8)'s stated invariant, enforced mechanically). The settings path
  (§8) is **not** on that allowlist — it is asserted byte-identical before and after every test,
  since hasp reads settings and never writes them; a test that needs a settings file writes one
  as fixture setup, outside the operation under test, never as a side effect the assertion has to
  tolerate.
- **The settings admission rule holds** ([T7](tech-decision-log.md#t7)). A test asserts the exact
  field set of the Go struct settings decode into (by name, via reflection or a golden list) —
  not just that decoding succeeds. Any PR adding a field changes this test's expected list
  visibly, which is the mechanical equivalent of [T1](tech-decision-log.md#t1)'s "no decrypt
  call site exists" and [T13](tech-decision-log.md#t13)'s `go list -deps` guard: the rule that
  matters most, because it guards against the exact shape of the project's original failure, is
  the one that does not get to rely on a reviewer noticing.
- **D4's move rule, including the alias-preserving refinement** ([T15](tech-decision-log.md#t15),
  §11). Inject a write failure partway through `adopt` (a read-only destination) and assert the
  original file still exists, readable, at its original path afterward — regardless of which
  step failed.
- **The symlinked-config case** ([T15](tech-decision-log.md#t15)). Construct a symlinked
  `~/.ssh/config` fixture and assert the symlink's *target inode* is unchanged after a write, not
  merely that the content is correct.
- **`WriteKeyFile` never silently overwrites** ([T22](tech-decision-log.md#t22)). Pre-create a
  file at the target path, run `new key` against it, and assert both that the command fails
  closed and that the pre-existing file's bytes are byte-identical afterward.

---

## 13. Build & distribution — GoReleaser as a constraint, not an afterthought

Full argument and section-by-section verification: [T9](tech-decision-log.md#t9). GoReleaser is
a stated constraint on this project; its v2 idioms are embraced rather than worked around.

- **`CGO_ENABLED=0`**, unconditionally achievable because [T1](tech-decision-log.md#t1) removed
  every dependency on `ssh-keygen` and `libmagic` the predecessor needed.
- **Build info** injected via `-X` ldflags (version, commit, build date), consumed by
  `hasp version`; `runtime/debug.ReadBuildInfo()` (`func ReadBuildInfo() (info *BuildInfo, ok
  bool)`) is the fallback for a `go install`-built binary that never went through GoReleaser, so
  `hasp version` never reports "unknown" for a build that carries module information at all.
- **Four targets:** `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`.
- **`ko`** for the container image — no `Dockerfile`, `FROM scratch`-capable, and only actually
  usable because of the `CGO_ENABLED=0` bullet above.
- **`nfpms`** for `.deb`/`.rpm`/`.apk`; **`aur`** for the Arch User Repository (the dev host's own
  package manager — dogfooding, not an arbitrary addition); **`homebrew_casks`** for macOS.
- **`sboms`** and **`signs`** for supply-chain attestation.
- Generated shell completions and man pages (from cobra, [T3](tech-decision-log.md#t3)) are
  packaged into the release archives, not left for a `go install` user to generate by hand.

**Explicit non-choices:** `dockers`/`docker_manifests` and `brews` are deprecated in GoReleaser
v2; hasp uses `ko` and `homebrew_casks` in their place. A future contributor reaching for either
deprecated section is reaching for the wrong one.

**Container caveat, stated plainly.** hasp manages the machine it runs *on* — §3.2 is explicit
that it is *"not configuration management... does not reach across a fleet."* A container image
is therefore for CI/scripted use, with the key directory bind-mounted in, and UID/`0700`-mode
mapping across that boundary is the sharp edge: a mismatched UID inside the container silently
produces a `~/.ssh` the host user cannot read.

---

## 14. Constraints M4 must not foreclose

Per D8: Git and GPG identity ([J9](design.md#7-journeys)) stay a **direction**, named only as
forward constraints on the model here, not designed. Reaching for this early is the specific
failure mode `docs/project-assessment-2026-08.md` documents this project having already lived
through once, and D8 is explicit that no near-term milestone may include it.

What the model above must not foreclose:

- **`Profile` (§3) must remain additive.** Nothing in its current shape (`Path`, `Dir`,
  `Managed`) should need to change type or be redesigned to grow a Git identity or a signing-key
  reference later — new fields, not a new type.
- **P9 governs J9 exactly as it governs SSH** (`design.md` §5.7, restated by D13): the natural
  structures for Git/GPG are `gitconfig` conditional includes and repository layout, not a
  hasp-owned registry. Nothing in `internal/adapter` should assume `sshconfig`
  ([T2](tech-decision-log.md#t2)) is the only configuration format hasp will ever parse — the
  package boundary (a dedicated `internal/adapter/sshconfig`, not a monolithic "config" adapter)
  already keeps this door open structurally.
- **The settings admission rule (§8) must survive J9's arrival unweakened.** A Git identity fact
  that *can* be derived from `gitconfig` must never be allowed into `settings.toml` "just because
  it's convenient" when J9 lands — the same discipline that governs the SSH-era settings file
  today governs it then.

---

## 15. Open questions

Genuine gaps, stated as questions, not silently dropped and not resolved by guessing.

1. **Ratifying `D16`.** §8 recommends `docs/decision-log.md` gain a `D16` recording that P9's
   last rung was reached and why. This document does not write it — that is the ratifying
   author's call, per the project's own stated process for amending `design.md`.

2. **Ratifying `D17`.** §9 recommends a second entry recording that `new key`'s passphrase
   prompt **narrows** §3.2's stated hard boundary — *"Not a passphrase manager. hasp never asks
   for, stores, or transmits a passphrase"* — rather than being licensed by it. No sentence in
   `design.md` authorizes this; it is the user's deliberate choice, made for non-interactive
   operation, and the project's own discipline says a narrowed boundary is amended, not quietly
   reinterpreted. This document does not write `D17` either, for the same reason it does not
   write `D16`.

3. **Relative `IdentityFile` resolution (§5) is this document's own interpretation, not a
   transcription of OpenSSH's behavior.** Real `ssh` resolves a bare relative path against its
   own connection-time working directory — a behavior with no meaningful analogue for hasp.
   Resolving it against the key directory instead is a judgment call made here because it matches
   what such paths mean in practice in hand-written configs; it should be confirmed rather than
   taken as derived from `ssh_config(5)` itself.

4. **`release host`'s exact behavior (§9) extrapolates D15's rule, rather than restating one
   `design.md` already states for hosts.** D15 requires `release`'s preview to show a marker
   file's contents before removing it, for **profiles**. This document applies the same
   "never silently lose content" spirit to hosts — re-inserting the stanza as plain text outside
   the marked region rather than deleting it — but `design.md` does not say this for hosts
   directly, and it is worth the ratifying author's explicit confirmation.

5. **Multi-directory / non-default key locations remain deferred**, exactly as `design.md` §10
   left them. Nothing in §3's domain types or §5's derivation pipeline hardcodes a single
   directory — `--key-dir` is a value threaded explicitly throughout (§12) rather than a global —
   so multi-directory support is additive later. It is not designed here.

6. **A severity taxonomy for `check` findings** — whether *"possible duplicate, cannot confirm"*
   ([T12](tech-decision-log.md#t12)) and a confirmed duplicate should carry a distinct `severity`
   field in the JSON envelope (§10), or whether `kind` alone is sufficient to let a consumer
   filter — is an interface-polish question for whoever implements `check`'s output schema, not
   settled here.
