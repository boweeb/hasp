package app

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// Plan is change represented as data before it is applied — the architectural keystone tdd.md §4
// is organized around. D6 makes preview the default for every write, uniformly; that is only
// structural, rather than a flag bolted onto each command, if the thing being previewed already
// exists as a value before anything is touched (T4).
//
// Plan.Changes is ordered so that any prefix of it leaves the machine in a working state, with
// the least recoverable change last (T20) — a property each use case's Plan-construction logic
// must satisfy; Applier itself only ever walks the slice in order and does not reorder it.
//
// A Plan with zero Changes is a legitimate, renderable value ("nothing to do") — T4's own stated
// consequence — so there is no separate no-op branch anywhere that consumes a Plan.
type Plan struct {
	Summary string
	Changes []Change

	// Witnesses are the "before" states every Change read while this Plan was being built (T30).
	// Applier.Apply re-verifies every one of them, for the whole Plan, before the first Change is
	// applied — not per-Change as it goes, because the Plan is the unit the user consented to.
	Witnesses []Witness
}

// Change is one typed, previewable, independently-backed-up unit of write work (T4). Every
// concrete kind is plain data plus these three methods — nothing about the safety machinery
// depends on a call site remembering to ask for a backup or a preview; both are properties of the
// Change itself.
type Change interface {
	// Preview returns a structured description of what Apply would do, consumed identically by
	// both renderers (tdd.md §10, T26). This IS the preview D6 requires — it exists before
	// anything is written, because Change is data first.
	Preview() Preview

	// RequiresBackup reports whether Applier must snapshot this Change's target before Apply
	// runs (P4). A property of the Change, never of caller diligence.
	RequiresBackup() bool

	// Apply performs the write, using only the primitives WriteFS exposes.
	Apply(fsys WriteFS) error
}

// backupSource is implemented by every Change kind whose RequiresBackup() can return true; it
// names the path Applier asks BackupStore to snapshot before Apply runs. Kept internal to the
// Applier<->Change wiring, deliberately not part of the Change interface itself (tdd.md §4's
// Change sketch has exactly three methods) and deliberately not taking a Change as BackupStore's
// parameter type either — see WriteFS's doc comment on why ports here take only stdlib types.
type backupSource interface {
	backupPath() string
}

// Preview is a Change's structured "what would this do" answer (T26). Diff is nil where there is
// no natural "before" to compare against (MoveFile, CreateSymlink, Remove), and unconditionally
// nil for WriteKeyFile regardless of AllowOverwrite — hasp never renders private key bytes to any
// renderer, in any mode (P3).
type Preview struct {
	Summary string     `json:"summary"`
	Diff    []DiffLine `json:"diff,omitempty"`
}

// DiffKind labels one line of a Preview's Diff.
type DiffKind int

const (
	DiffContext DiffKind = iota
	DiffAdded
	DiffRemoved
)

func (k DiffKind) String() string {
	switch k {
	case DiffAdded:
		return "added"
	case DiffRemoved:
		return "removed"
	default:
		return "context"
	}
}

// MarshalJSON renders as the same string the human renderer would print (tdd.md §10), so a diff
// line seen in the terminal is findable in --json output without translating between vocabularies
// — the same convention internal/domain's marshalStringer establishes for its own enums.
func (k DiffKind) MarshalJSON() ([]byte, error) { return json.Marshal(k.String()) }

// DiffLine is one line of a unified-diff-style Preview.Diff.
type DiffLine struct {
	Kind DiffKind `json:"kind"`
	Text string   `json:"text"`
}

// Witness is the "before" state a Change read while a Plan was being built — size, mtime, and a
// content hash of the bytes actually read (T30). Applier re-verifies every Witness in a Plan
// before applying the first Change; a mismatch means the machine changed between preview and
// confirm, and the whole Plan is refused, before the first write, with nothing backed up.
//
// Size and ModTime exist to make a refusal cheap to explain; they are never the basis for the
// pass/fail decision. Sum decides: a file rewritten to byte-identical content is not a race, and
// must not trip the guard, even though its ModTime changed.
type Witness struct {
	Path    string
	Size    int64
	ModTime time.Time
	Sum     [32]byte // SHA-256 of the bytes Plan() actually read
}

// NewWitness reads path and captures a Witness of its current state. Call this from the same
// read that produces a Change's Preview().Diff, while a use case is building a Plan — capturing
// the witness then costs no extra I/O (T30).
func NewWitness(path string) (Witness, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Witness{}, fmt.Errorf("witness %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return Witness{}, fmt.Errorf("witness %s: %w", path, err)
	}
	return Witness{Path: path, Size: info.Size(), ModTime: info.ModTime(), Sum: sha256.Sum256(data)}, nil
}

// verify re-reads w.Path and reports whether its content hash still matches what was witnessed.
// Any read failure (including the file having been removed) is treated as a mismatch — the
// machine no longer looks like what the Plan was built against, exactly as a content change
// would.
func (w Witness) verify() error {
	data, err := os.ReadFile(w.Path)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrPlanStale, w.Path, err)
	}
	if sha256.Sum256(data) != w.Sum {
		return fmt.Errorf("%w: %s changed since this plan was built; re-run to see the current state", ErrPlanStale, w.Path)
	}
	return nil
}

// WriteFS is the write-mechanics port every Change.Apply uses to touch the filesystem — declared
// here, at the point Applier consumes it (tdd.md §3's ports-at-point-of-consumption pattern,
// mirroring the keyFileWriter example). Every method signature uses only stdlib types, so the
// concrete adapter (internal/adapter/fswrite) never needs to import this package at all; Go's
// structural interfaces are what make that possible.
//
// The four methods below are §11's "Write mechanics" 1-4, at the granularity each Change kind
// actually needs.
type WriteFS interface {
	// WriteFile atomically writes data to path: a temp file in the same directory, fsync, an
	// os.Rename onto the (resolved) target, then fsync of the containing directory (§11 point 1).
	// If path is itself a symlink, the write goes through to its resolved target — the symlink
	// itself is never replaced (§11 point 2). If a file already exists at the resolved target,
	// its current mode is preserved and the mode parameter is ignored; otherwise the new file is
	// created with mode (§11 point 3).
	WriteFile(path string, data []byte, mode fs.FileMode) error

	// Symlink creates a new symlink at path pointing at target. It fails if path already exists —
	// WriteFS never silently replaces an existing filesystem entry with a symlink.
	Symlink(path, target string) error

	// Remove deletes the file at path.
	Remove(path string) error

	// Move performs D4's move rule (§11 point 4): copy from's bytes to to, verify the copy
	// byte-for-byte, then remove from — from is never unlinked first. Move fails if to already
	// exists. If Move returns an error, from is guaranteed to still exist, readable and unchanged,
	// at its original path — regardless of which step failed.
	Move(from, to string) error

	// ReplaceWithSymlink is `adopt`'s alias-preserving move (§4 "Change ordering", §11 point 4,
	// T20): copy oldPath's bytes to newTarget (refusing if newTarget already exists), verify the
	// copy byte-for-byte, then atomically os.Rename a freshly created symlink (oldPath ->
	// newTarget) onto oldPath — replacing the original file there with an alias to its new home in
	// a single, atomic operation. oldPath is never bare-unlinked. If ReplaceWithSymlink returns an
	// error, oldPath is guaranteed to still hold either its original, unmodified file (if the
	// terminal rename never ran) or the new alias (if it did) — never neither, and never a partial
	// symlink. Note this is not itself a guarantee of retryability: if the error occurs after the
	// verified copy already landed at newTarget but before the terminal rename, newTarget now
	// exists, and this method refuses on a subsequent call for the same newTarget (see its own
	// implementation) — recovery from that state is manual (§11 point 4), not automatic.
	ReplaceWithSymlink(oldPath, newTarget string) error

	// ReplaceSymlinkWithFile is `release`'s mirror of ReplaceWithSymlink: copy source's bytes to a
	// fresh, independent file, verify the copy byte-for-byte, then atomically os.Rename it onto
	// path — replacing whatever currently sits there (ReplaceSymlinkWithFile refuses unless it is
	// a symlink, the shape adopt leaves behind) with a real, self-contained copy. source is never
	// touched or removed by this call; that is the caller's own, separate step (mirroring D4's
	// move rule read in reverse — the destination is replaced only once a verified copy exists,
	// and the redundant source is cleaned up afterward, not before). If ReplaceSymlinkWithFile
	// returns an error, path is guaranteed to still hold whatever it held before the call,
	// unchanged.
	ReplaceSymlinkWithFile(path, source string) error
}

// BackupStore snapshots a file's current bytes into the backup store (~/.ssh/.hasp-backups/, T8)
// before a destructive write. Applier calls Snapshot immediately before Change.Apply whenever
// RequiresBackup() is true, and never otherwise. A source path that does not exist is not an
// error — P4 only requires that prior state be preserved, and there is none to preserve.
type BackupStore interface {
	Snapshot(path string) error
}

// Result is what a successfully applied Plan produced.
type Result struct {
	Applied []Change
}

// Applier is the only thing in the codebase permitted to execute a Plan (T4). It re-verifies
// every Witness the Plan carries before touching anything, then walks Changes in order, backing
// up (when required) immediately before each one's Apply.
type Applier struct {
	FS      WriteFS
	Backups BackupStore
}

// Apply executes p: first re-verifying every Witness (T30) — any mismatch aborts the whole Plan
// before the first write, with nothing backed up — then walking p.Changes in order, backing up
// (per Change.RequiresBackup) immediately before each Change's own Apply.
func (a *Applier) Apply(p Plan) (Result, error) {
	for _, w := range p.Witnesses {
		if err := w.verify(); err != nil {
			return Result{}, err
		}
	}

	for _, c := range p.Changes {
		if c.RequiresBackup() {
			bs, ok := c.(backupSource)
			if !ok {
				return Result{}, fmt.Errorf("internal error: %T requires a backup but declares no backup source", c)
			}
			if err := a.Backups.Snapshot(bs.backupPath()); err != nil {
				return Result{}, fmt.Errorf("backup failed, aborting before write: %w", err)
			}
		}
		if err := c.Apply(a.FS); err != nil {
			return Result{}, err
		}
	}

	return Result{Applied: p.Changes}, nil
}
