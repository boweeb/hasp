package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/domain"
)

// keyGenerator is the port NewKeyUseCase.Plan consumes, declared at the point of consumption
// (tdd.md §3's ports-at-consumption pattern, mirroring the keyFileWriter sketch) — a function
// value rather than an interface, since keyfile.Generate is already the single, stateless
// operation this use case needs, and a one-method interface would name nothing an interface
// value gives that a func value doesn't.
type keyGenerator func(keyfile.GenerateOptions) (keyfile.GeneratedKey, error)

// NewKeyRequest is `new key`'s input (tdd.md §9's `new` grid cell). Passphrase resolution (flag
// -> env -> settings -> default, T6) happens entirely in internal/cli before this Request is
// built — internal/app knows nothing about env vars, TTYs, or settings files, only the decided
// secret (tdd.md §3's Request-only framing).
type NewKeyRequest struct {
	Name    string // the file's basename under KeyDir (or KeyDir/Profile)
	KeyDir  string
	Profile domain.ProfilePath // nil/empty means the top level of KeyDir

	// Passphrase, if non-empty, encrypts the generated private key (D17, T6: accepted only at
	// generation time, never persisted). It is handed straight to keyfile.Generate, which zeroes
	// it in place before Plan returns — the caller's own copy is zeroed as a side effect.
	Passphrase []byte
}

// NewKeyUseCase generates a new ed25519 keypair (tdd.md §9's `new key` grid cell, T1's default
// algorithm).
type NewKeyUseCase struct {
	// Generate overrides keyfile.Generate for tests; nil means the production default.
	Generate keyGenerator
}

// Plan builds the two WriteKeyFile changes `new key` needs — the private key, then its .pub
// sibling — both AllowOverwrite: false (T22): new key never overwrites. Private key first: if
// the private-key write fails, nothing has been created; if the .pub write fails after the
// private key succeeded, the machine is left with a private-key-only key, which is itself a
// complete, valid, working state (T1's own derivation matrix has no public-half requirement) —
// the reverse order would instead risk leaving an orphaned .pub file with no key behind it,
// which is nonsensical and never a legitimate end state. This is tdd.md §4's ordering principle:
// any prefix of Plan.Changes leaves the machine at least as good as where it started.
//
// Plan performs no I/O beyond generating the keypair in memory (keyfile.Generate never touches
// disk) — reads are always safe (P5), and generation is not a read, but nothing here writes
// either. `new key`'s recorded facts are never asserted here; they come from re-deriving the
// written file through the same keyfile.Inspect path every other key goes through (T6), which is
// why WriteKeyFile's own Preview carries a Summary only, never a claim about the key's algorithm
// or fingerprint.
func (uc NewKeyUseCase) Plan(req NewKeyRequest) (Plan, error) {
	if req.Name == "" {
		return Plan{}, fmt.Errorf("%w: a key name is required", ErrUsage)
	}
	if req.KeyDir == "" {
		return Plan{}, fmt.Errorf("%w: a key directory is required", ErrUsage)
	}

	// T8 / tdd.md §12: "hasp writes nothing outside the key directory." req.Name and req.Profile
	// arrive here straight from the CLI's positional argument and --profile flag with no shape
	// guarantee — validate them here, in internal/app, rather than relying on internal/cli to have
	// done it, since this invariant must hold independent of cobra: later phases (adopt, edit key
	// --profile) will build NewKeyRequest-shaped values through other entry points and must not be
	// able to reintroduce this hole by skipping a CLI-layer check. domain.ParseProfilePath is left
	// alone deliberately: it's a pure dotted-string splitter used by read paths too (e.g. `show
	// profile`), and constraining its output isn't right for every caller — the write-path
	// invariant belongs here, at the point profile segments are actually joined into a filesystem
	// path.
	if err := validatePathSegment("key name", req.Name); err != nil {
		return Plan{}, err
	}
	for _, seg := range req.Profile {
		if err := validatePathSegment("profile segment", seg); err != nil {
			return Plan{}, err
		}
	}

	generate := uc.Generate
	if generate == nil {
		generate = keyfile.Generate
	}

	dir := req.KeyDir
	if len(req.Profile) > 0 {
		dir = filepath.Join(append([]string{req.KeyDir}, []string(req.Profile)...)...)
	}
	path := filepath.Join(dir, req.Name)
	pubPath := path + ".pub"

	// Defense in depth alongside validatePathSegment above: even if a future caller reaches this
	// point with an already-malformed dir/path (a bug in validation, or a request built by some
	// other codepath that forgot to validate), refuse to plan a write outside req.KeyDir rather
	// than merely trusting the per-segment checks already ran.
	if err := requireWithinKeyDir(req.KeyDir, path); err != nil {
		return Plan{}, err
	}

	gen, err := generate(keyfile.GenerateOptions{Passphrase: req.Passphrase})
	if err != nil {
		return Plan{}, fmt.Errorf("generate key: %w", err)
	}

	return Plan{
		Summary: fmt.Sprintf("generate new key %q", req.Name),
		Changes: []Change{
			WriteKeyFile{Path: path, Contents: gen.PrivateKeyPEM, Mode: 0o600, AllowOverwrite: false},
			WriteKeyFile{Path: pubPath, Contents: gen.PublicKeyLine, Mode: 0o644, AllowOverwrite: false},
		},
	}, nil
}

// validatePathSegment rejects a name or profile segment that isn't a single, literal path
// component: empty, containing a path separator (both the OS-native filepath.Separator and the
// portable '/', checked independent of GOOS so this behaves identically in tests on any
// platform — a name with an embedded '/' would still split a path open even on a system whose
// native separator is '\'), or one of the "." / ".." special directory names. This is the primary
// defense for T8's "hasp writes nothing outside the key directory" invariant (tdd.md §12): reject
// the shape before it's ever joined into a path at all. A malformed name is a usage mistake
// (exit code 2, tdd.md §10), not an internal failure, so it wraps ErrUsage like every other
// argument-shape rejection in this file.
func validatePathSegment(kind, s string) error {
	if s == "" {
		return fmt.Errorf("%w: %s must not be empty", ErrUsage, kind)
	}
	if strings.ContainsRune(s, filepath.Separator) || strings.ContainsRune(s, '/') {
		return fmt.Errorf("%w: %s %q must be a plain name, not a path", ErrUsage, kind, s)
	}
	if strings.ContainsRune(s, 0) {
		return fmt.Errorf("%w: %s must not contain a null byte", ErrUsage, kind)
	}
	if s == "." || s == ".." {
		return fmt.Errorf("%w: %s %q is not a valid name", ErrUsage, kind, s)
	}
	return nil
}

// requireWithinKeyDir confirms path resolves to a descendant of keyDir, using filepath.Rel on
// their absolute forms and rejecting a ".." result or one that starts with "../" — this is a
// lexical containment check only, not a symlink-aware one, and it has a known, accepted gap: if
// an intermediate path component under keyDir (e.g. a profile directory) is itself a symlink to
// somewhere outside keyDir, this check does not detect it, because it never resolves symlinks —
// it only rejects ".."-shaped lexical escapes, which is what validatePathSegment's per-component
// restriction combines with it to close. Reaching that gap requires an attacker who can already
// write inside keyDir to plant the symlink first, which P8's single-user-laptop threat model
// treats as out of scope (the same trust boundary every other write in this codebase already
// assumes). keyDir itself is not resolved before this call — filepath.Abs above is the first
// resolution it gets — so this function's only real job is the lexical ".."-escape check; treat it
// as a second, independent line of defense against a future construction bug, not a symlink
// boundary.
func requireWithinKeyDir(keyDir, path string) error {
	absKeyDir, err := filepath.Abs(keyDir)
	if err != nil {
		return fmt.Errorf("resolve key directory: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}
	rel, err := filepath.Rel(absKeyDir, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: target path escapes the key directory", ErrUsage)
	}
	return nil
}
