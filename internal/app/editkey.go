package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/domain"
)

// EditKeyRequest is `edit key`'s input (tdd.md §9's `edit` grid cell for `key`, T19, T22). It
// bundles four independent sub-operations behind one Request rather than four separate Request
// types, because they share one target key and one CLI verb; exactly which sub-operation(s) a
// given call performs is determined by which fields are non-empty (see Plan's own doc comment for
// the "exactly one, with one documented exception" rule this enforces).
//
// Unlike AdoptKeyRequest/ReleaseKeyRequest — which carry only the plain, already-resolved path
// strings a caller extracted from a Machine — EditKeyRequest carries the target Key itself
// (Locations included), a deliberate departure from that precedent. The reason is
// --remove-alias: it is the one operation in this phase capable of destroying something if
// misapplied (deleting the wrong location), and resolving "is this Location actually an alias"
// independently, here, against the same Key value the rest of Plan reasons from, is a second,
// structural line of defense rather than trusting a single upstream CLI computation — the same
// spirit as requireWithinKeyDir's own "defense in depth" comment elsewhere in this package. Key is
// still pure data (no I/O, no clue-resolution logic) once it reaches this Request, so this does
// not reopen tdd.md §3's Request-only framing: internal/app still never resolves a
// name-or-clue itself, it only reasons over a Key value the caller already produced.
type EditKeyRequest struct {
	KeyDir string
	Key    domain.Key // resolved by the caller from a name-or-clue exactly as
	// AdoptKeyRequest.Source/ReleaseKeyRequest.Source are (app.ShowKey/app.FindKeys against a
	// derived Machine, tdd.md §3's Request-only framing) — carried whole, not just extracted
	// paths, for the reason explained above.

	// NewName renames the key's real file in place (same directory, new basename): --name.
	// Empty means no rename requested.
	NewName string

	// NewProfile relocates the key's real file into a different profile directory (same
	// basename): --profile. Nil/empty means no move requested. NewName and NewProfile may be
	// combined in one call — see Plan's doc comment.
	NewProfile domain.ProfilePath

	// AddAliasProfile + AddAliasLeaf are --add-alias=<profile>/<name>'s two halves, already
	// split by the caller (mirroring NewKeyRequest's own pre-split Profile+Name), a bare
	// "<name>" meaning a top-level alias (nil AddAliasProfile). AddAliasLeaf == "" means no
	// alias-add requested.
	AddAliasProfile domain.ProfilePath
	AddAliasLeaf    string

	// RemoveAliasProfile + RemoveAliasLeaf are --remove-alias=<profile>/<name>'s two halves,
	// split identically to AddAliasProfile/AddAliasLeaf — deliberately the same shape, not a
	// free-form "clue," so there is no ambiguity about which filesystem path is being targeted
	// by the one operation in this phase that can delete something if misused. RemoveAliasLeaf
	// == "" means no alias-remove requested.
	RemoveAliasProfile domain.ProfilePath
	RemoveAliasLeaf    string

	// ReplaceMaterialPath is the filesystem path to read new private key bytes from:
	// --replace-material=<path>. Empty means no material-replace requested.
	ReplaceMaterialPath string
}

// EditKeyUseCase implements `edit key`'s four sub-operations (tdd.md §9). Unlike every other
// write use case built so far, it does not need a new WriteFS primitive: every Change kind it
// plans (MoveFile, CreateSymlink, Remove, WriteKeyFile) already exists from Phase 1.
type EditKeyUseCase struct{}

// editKeyOp identifies which of the four sub-operations a Request is asking for.
type editKeyOp int

const (
	editKeyOpNone editKeyOp = iota
	editKeyOpRenameOrMove
	editKeyOpAddAlias
	editKeyOpRemoveAlias
	editKeyOpReplaceMaterial
)

// resolveOp determines which single sub-operation req asks for, refusing before anything is
// planned if zero or more than one are requested.
//
// **Judgment call, stated explicitly (per this phase's own instructions): a single `edit key`
// invocation performs exactly one sub-operation, with one documented exception.** --name and
// --profile may be combined into a single MoveFile (rename-and-move at once) because they both
// target the same thing — the real file's own location — and combining them is strictly safer
// than issuing two separate Moves with an intermediate state between them. Every other pair
// (e.g. --add-alias with --replace-material) is refused outright rather than composed, because
// composing them correctly would require reasoning about *ordering* between operations that touch
// related-but-different paths (a --name/--profile move changes the very path --replace-material
// or --add-alias would otherwise resolve to), and inventing that composition logic now would be
// new, unratified design territory this phase's own instructions caution against for the
// dangling-alias case — the same caution applies here by the same reasoning. A user who wants
// several edits performed can run `edit key` several times; each one previews and confirms on its
// own (D6), which is honest rather than a missing convenience.
func resolveOp(req EditKeyRequest) (editKeyOp, error) {
	renameOrMove := req.NewName != "" || len(req.NewProfile) > 0
	addAlias := req.AddAliasLeaf != ""
	removeAlias := req.RemoveAliasLeaf != ""
	replaceMaterial := req.ReplaceMaterialPath != ""

	count := 0
	var op editKeyOp
	for _, candidate := range []struct {
		flag bool
		op   editKeyOp
	}{
		{renameOrMove, editKeyOpRenameOrMove},
		{addAlias, editKeyOpAddAlias},
		{removeAlias, editKeyOpRemoveAlias},
		{replaceMaterial, editKeyOpReplaceMaterial},
	} {
		if candidate.flag {
			count++
			op = candidate.op
		}
	}

	switch count {
	case 0:
		return editKeyOpNone, fmt.Errorf("%w: at least one of --name, --profile, --add-alias, --remove-alias, --replace-material is required", ErrUsage)
	case 1:
		return op, nil
	default:
		return editKeyOpNone, fmt.Errorf("%w: edit key performs exactly one operation per invocation (--name and --profile may be combined; --add-alias, --remove-alias, and --replace-material are each independent and must be run separately)", ErrUsage)
	}
}

// Plan dispatches to exactly one of the four sub-operation planners below, after the shared
// preconditions every one of them needs: a key directory, and a Key actually named to edit.
func (uc EditKeyUseCase) Plan(req EditKeyRequest) (Plan, error) {
	if req.KeyDir == "" {
		return Plan{}, fmt.Errorf("%w: a key directory is required", ErrUsage)
	}
	if req.Key.Name == "" {
		return Plan{}, fmt.Errorf("%w: a key to edit is required", ErrUsage)
	}

	op, err := resolveOp(req)
	if err != nil {
		return Plan{}, err
	}

	switch op {
	case editKeyOpRenameOrMove:
		return planRenameOrMove(req)
	case editKeyOpAddAlias:
		return planAddAlias(req)
	case editKeyOpRemoveAlias:
		return planRemoveAlias(req)
	case editKeyOpReplaceMaterial:
		return planReplaceMaterial(req)
	default:
		return Plan{}, fmt.Errorf("internal error: unresolved edit key operation")
	}
}

// keySingleRealLocation returns k's one non-alias location, mirroring internal/cli's own
// singleRealLocation helper (adopt/release's precondition) but implemented independently here so
// internal/app's own preconditions do not depend on a CLI-layer helper it cannot import — the same
// reasoning requireWithinKeyDir's "defense in depth" gives for re-checking what a caller may have
// already checked.
func keySingleRealLocation(k domain.Key) ([]domain.KeyLocation, string, error) {
	var real []domain.KeyLocation
	for _, loc := range k.Locations {
		if !loc.IsAlias {
			real = append(real, loc)
		}
	}
	switch len(real) {
	case 1:
		return real, real[0].Path, nil
	case 0:
		return real, "", fmt.Errorf("%w: key %q has no real (non-alias) location", ErrUsage, k.Name)
	default:
		return real, "", fmt.Errorf("%w: key %q has %d real locations; not a shape edit key can act on", ErrUsage, k.Name, len(real))
	}
}

// planRenameOrMove builds --name/--profile's Plan: a single MoveFile (plus a mirrored MoveFile for
// a .pub sidecar, if one exists — the same regression AdoptKeyUseCase.Plan's own doc comment
// explains: keyfile.Inspect derives HasPublicHalf/Comment from a .pub sitting next to the
// *resolved* real path, so moving only the private key would silently leave those facts wrong the
// instant the file lands at its new location).
//
// **The dangling-alias precondition, stated in full (per this phase's own instructions, not
// invented silently): renaming or moving a key's real file can leave any existing alias symlink
// dangling, because nothing in this codebase retargets a symlink's destination when the file it
// points at moves — no section of tdd.md's §4/§5/§9/§11 describes a mechanism for that, and
// inventing one now would be new, unratified design territory. This use case refuses, before any
// write, if the key currently has more than one real location (T12's unconfirmed-duplicate shape —
// not something a single MoveFile can address anyway) or any alias location at all. This is a
// scope boundary worth naming to the project owner: a future `edit key` enhancement that
// automatically retargets aliases when their target moves would need its own design decision; this
// phase keeps the operation safe by construction (refuse) rather than by discipline (hope the
// caller remembers to check).**
func planRenameOrMove(req EditKeyRequest) (Plan, error) {
	for _, loc := range req.Key.Locations {
		if loc.IsAlias {
			return Plan{}, fmt.Errorf("%w: key %q has an alias at %s; remove it first with --remove-alias before renaming or moving the key (a rename/move does not retarget existing aliases, which would otherwise dangle)", ErrUsage, req.Key.Name, loc.Path)
		}
	}
	_, currentPath, err := keySingleRealLocation(req.Key)
	if err != nil {
		return Plan{}, err
	}
	if err := requireWithinKeyDir(req.KeyDir, currentPath); err != nil {
		return Plan{}, err
	}

	newDir := filepath.Dir(currentPath)
	newBase := filepath.Base(currentPath)

	if req.NewName != "" {
		if err := validatePathSegment("key name", req.NewName); err != nil {
			return Plan{}, err
		}
		newBase = req.NewName
	}

	if len(req.NewProfile) > 0 {
		for _, seg := range req.NewProfile {
			if err := validatePathSegment("profile segment", seg); err != nil {
				return Plan{}, err
			}
		}
		newDir = filepath.Join(append([]string{req.KeyDir}, []string(req.NewProfile)...)...)

		// **Destination existence, decided (per this phase's own instructions): `edit key
		// --profile` requires the destination directory to already exist — managed or not, per
		// the Classify table's own point that a directory is a legitimate profile *container*
		// without a `.hasp` marker, only "managed" requires one — and refuses if it does not,
		// rather than creating it.** WriteFS.Move (fswrite.FS.Move) does not create intermediate
		// directories itself (it calls os.CreateTemp directly against the destination's own
		// directory, which fails if that directory does not exist), and there is deliberately no
		// "create a directory" Change kind in this codebase yet (`new profile`, the one verb that
		// would need one, is not built as of this phase) — silently MkdirAll-ing one here, outside
		// any Change/Preview, would be exactly the kind of unpreviewed write D6 exists to forbid:
		// a write the user never saw named before confirming. Refusing with a clear error is
		// consistent with AdoptKeyUseCase's own refusal on an unmanaged/missing target directory.
		info, statErr := os.Stat(newDir)
		if statErr != nil {
			if errors.Is(statErr, fs.ErrNotExist) {
				return Plan{}, fmt.Errorf("%w: target profile directory %s does not exist; edit key --profile relocates a key, it does not create directories", ErrUsage, newDir)
			}
			return Plan{}, fmt.Errorf("check %s: %w", newDir, statErr)
		}
		if !info.IsDir() {
			return Plan{}, fmt.Errorf("%w: target profile path %s is not a directory", ErrUsage, newDir)
		}
	}

	newPath := filepath.Join(newDir, newBase)
	// T8 / tdd.md §12, mirroring every other write use case's own defense-in-depth check.
	if err := requireWithinKeyDir(req.KeyDir, newPath); err != nil {
		return Plan{}, err
	}
	if filepath.Clean(newPath) == filepath.Clean(currentPath) {
		return Plan{}, fmt.Errorf("%w: nothing to change; the new location is identical to the current one", ErrUsage)
	}

	changes := []Change{MoveFile{From: currentPath, To: newPath, Reason: "edit key: rename/move the real key file"}}
	if hasSidecar(currentPath + ".pub") {
		changes = append(changes, MoveFile{From: currentPath + ".pub", To: newPath + ".pub", Reason: "edit key: rename/move the .pub sidecar alongside its key"})
	}

	return Plan{
		Summary: fmt.Sprintf("edit key %q: move %s to %s", req.Key.Name, currentPath, newPath),
		Changes: changes,
	}, nil
}

// planAddAlias builds --add-alias's Plan (T19's exact mechanism, D2's write path): a single
// CreateSymlink at the requested profile-qualified path, pointing at the key's canonical (real)
// location.
//
// No .pub-sidecar handling here, unlike rename/move: an alias never moves or copies anything —
// the real file and its .pub (if any) stay exactly where they already are, so keyfile.Inspect's
// derivation off the *resolved* real path is completely unaffected by an alias's existence. This
// is a deliberate, reasoned absence, not an oversight (contrast with planRenameOrMove/
// planReplaceMaterial, both of which do touch the real file and so do need to think about the
// sidecar).
//
// Precondition: exactly one real location for the key to point the new alias at (mirroring
// keySingleRealLocation's use elsewhere) — but, unlike planRenameOrMove, no restriction on
// existing aliases: adding a second alias alongside a first is exactly D2's ordinary, intended
// multi-profile-membership case (T19), not a hazard.
func planAddAlias(req EditKeyRequest) (Plan, error) {
	for _, seg := range req.AddAliasProfile {
		if err := validatePathSegment("profile segment", seg); err != nil {
			return Plan{}, err
		}
	}
	if err := validatePathSegment("alias name", req.AddAliasLeaf); err != nil {
		return Plan{}, err
	}

	_, target, err := keySingleRealLocation(req.Key)
	if err != nil {
		return Plan{}, err
	}
	if err := requireWithinKeyDir(req.KeyDir, target); err != nil {
		return Plan{}, err
	}

	aliasDir := req.KeyDir
	if len(req.AddAliasProfile) > 0 {
		aliasDir = filepath.Join(append([]string{req.KeyDir}, []string(req.AddAliasProfile)...)...)
	}
	aliasPath := filepath.Join(aliasDir, req.AddAliasLeaf)
	if err := requireWithinKeyDir(req.KeyDir, aliasPath); err != nil {
		return Plan{}, err
	}

	// CreateSymlink.Apply (via fswrite.FS.Symlink) already fails closed if aliasPath exists, but
	// checking here too lets Plan() report a clear usage error before anything is even previewed,
	// rather than deferring the same refusal to Apply time.
	if _, statErr := os.Lstat(aliasPath); statErr == nil {
		return Plan{}, fmt.Errorf("%w: %s already exists; --add-alias never replaces an existing filesystem entry", ErrUsage, aliasPath)
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return Plan{}, fmt.Errorf("check %s: %w", aliasPath, statErr)
	}

	return Plan{
		Summary: fmt.Sprintf("edit key %q: add alias %s -> %s", req.Key.Name, aliasPath, target),
		Changes: []Change{CreateSymlink{Path: aliasPath, Target: target}},
	}, nil
}

// planRemoveAlias builds --remove-alias's Plan: a single Remove targeting exactly the alias
// location the caller named, resolved the same way planAddAlias resolves --add-alias's argument
// (<profile>/<name>, or a bare <name> for a top-level alias) — deliberately not a free-form clue,
// so there is never any ambiguity about which filesystem path is about to be deleted.
//
// **This is the one place in this phase where a mistake could actually destroy key material
// (T22's own reasoning: Remove is restricted to hasp-authored artifacts, never a key file, but
// nothing stops a caller from constructing one against the wrong path) — treated with the same
// care Phase 3a treated adopt/release's own preconditions.** The resolved path is looked up
// against req.Key.Locations (not merely assumed from the flag's shape), and Plan refuses outright
// unless that exact path is a known location of this key AND that location's IsAlias is true.
// Refusing on "not a known location at all" (rather than silently treating an arbitrary path as
// removable) is itself part of the same fail-closed posture: --remove-alias only ever removes
// something this Key value already reports as one of its own alias locations.
//
// No .pub-sidecar handling here: `adopt key` can leave a matching `.pub` alias behind, but
// --add-alias itself never creates one, and inferring "also remove the paired .pub alias" would
// extend adopt/release's own established sidecar-pairing behavior into a mechanism this phase was
// not asked to build. A user who also wants an alias's .pub sidecar gone can run --remove-alias a
// second time against it directly. Named here as a deliberate scope boundary, not an oversight.
func planRemoveAlias(req EditKeyRequest) (Plan, error) {
	for _, seg := range req.RemoveAliasProfile {
		if err := validatePathSegment("profile segment", seg); err != nil {
			return Plan{}, err
		}
	}
	if err := validatePathSegment("alias name", req.RemoveAliasLeaf); err != nil {
		return Plan{}, err
	}

	aliasDir := req.KeyDir
	if len(req.RemoveAliasProfile) > 0 {
		aliasDir = filepath.Join(append([]string{req.KeyDir}, []string(req.RemoveAliasProfile)...)...)
	}
	aliasPath := filepath.Join(aliasDir, req.RemoveAliasLeaf)
	if err := requireWithinKeyDir(req.KeyDir, aliasPath); err != nil {
		return Plan{}, err
	}

	var matched *domain.KeyLocation
	for i := range req.Key.Locations {
		if filepath.Clean(req.Key.Locations[i].Path) == filepath.Clean(aliasPath) {
			matched = &req.Key.Locations[i]
			break
		}
	}
	if matched == nil {
		return Plan{}, fmt.Errorf("%w: %s is not a known location of key %q", ErrUsage, aliasPath, req.Key.Name)
	}
	if !matched.IsAlias {
		return Plan{}, fmt.Errorf("%w: %s is key %q's real file, not an alias; --remove-alias never removes the real key file", ErrUsage, aliasPath, req.Key.Name)
	}

	return Plan{
		Summary: fmt.Sprintf("edit key %q: remove alias %s", req.Key.Name, aliasPath),
		Changes: []Change{Remove{Path: aliasPath, Reason: "edit key: remove alias"}},
	}, nil
}

// planReplaceMaterial builds --replace-material's Plan (T22, tdd.md §4/§11 point 5 — see this
// package's WriteKeyFile doc comment for the T22-Consequence-vs-§4/§11 discrepancy this
// implementation resolves in §4/§11's favor, since WriteKeyFile{AllowOverwrite:true} is what T22's
// own Decision section, and every other phase's own Change kind, already builds against).
//
// Precondition: exactly one real location, mirroring planAddAlias — WriteKeyFile needs exactly one
// concrete target path to overwrite. Unlike planRenameOrMove, existing aliases impose no
// restriction here: replace-material writes new bytes to the *same* path the real file already
// occupies, so any alias symlink pointing at it keeps resolving correctly and immediately reflects
// the new material — there is no dangling-alias hazard for this operation, because nothing moves.
func planReplaceMaterial(req EditKeyRequest) (Plan, error) {
	_, currentPath, err := keySingleRealLocation(req.Key)
	if err != nil {
		return Plan{}, err
	}
	if err := requireWithinKeyDir(req.KeyDir, currentPath); err != nil {
		return Plan{}, err
	}

	// Fail closed before any Change is built if the incoming file is not actually a parseable
	// private key (T22's own instruction: validate first, plan nothing otherwise).
	if _, err := keyfile.Inspect(req.ReplaceMaterialPath); err != nil {
		return Plan{}, fmt.Errorf("%w: %s does not contain a parseable private key: %v", ErrUsage, req.ReplaceMaterialPath, err)
	}

	contents, err := os.ReadFile(req.ReplaceMaterialPath)
	if err != nil {
		return Plan{}, fmt.Errorf("read %s: %w", req.ReplaceMaterialPath, err)
	}

	// .pub sidecar handling, decided (per this phase's own instructions): the incoming
	// replacement material's own .pub, if any, replaces whatever .pub currently sits beside the
	// key being edited — and if the incoming material has *no* .pub but the existing key *did*
	// have one, that old .pub is now stale and actively wrong, not merely absent: keyfile.Inspect
	// treats a present, parseable .pub as authoritative for Fingerprint whenever one exists,
	// unconditionally overriding a fingerprint it could otherwise derive straight from the private
	// key material. Left in place, a stale .pub would make hasp report the *old* key's fingerprint
	// against the *new* private key's bytes after a successful --replace-material — a silently
	// wrong fact, which P1 forbids. So: incoming .pub present -> replace; incoming .pub absent but
	// an old one exists -> remove it.
	//
	// Ordering (T20's "any prefix leaves a working state" rule, review-flagged): the .pub Change
	// is built and ordered BEFORE the private-key WriteKeyFile below, not after. If Apply succeeds
	// on the .pub step but fails before the private-key step ever runs, the prefix that did apply
	// leaves the *old, still-matching* private key behind (possibly now missing its .pub, or
	// carrying the new one early) — keyfile.Inspect still derives a self-consistent answer from
	// whatever private key is actually on disk. The reverse order — private key first — has a
	// worse failure mode: a successful private-key swap followed by a failed .pub cleanup leaves
	// new bytes on disk with a stale .pub silently misreporting the old fingerprint, which is
	// exactly the P1 violation this whole mechanism exists to prevent, and unlike Preview (which
	// always shows both changes before either runs), a partial-apply failure gives that violation
	// no preview to have been caught by first.
	existingPub := currentPath + ".pub"
	incomingPub := req.ReplaceMaterialPath + ".pub"
	var changes []Change
	switch {
	case hasSidecar(incomingPub):
		pubContents, pubErr := os.ReadFile(incomingPub)
		if pubErr != nil {
			return Plan{}, fmt.Errorf("read %s: %w", incomingPub, pubErr)
		}
		changes = append(changes, WriteKeyFile{Path: existingPub, Contents: pubContents, Mode: 0o644, AllowOverwrite: true})
	case hasSidecar(existingPub):
		changes = append(changes, Remove{Path: existingPub, Reason: "edit key --replace-material: the old .pub sidecar described the replaced key material and would otherwise report a stale fingerprint"})
	}
	changes = append(changes, WriteKeyFile{Path: currentPath, Contents: contents, Mode: 0o600, AllowOverwrite: true})

	return Plan{
		Summary: fmt.Sprintf("edit key %q: replace material at %s", req.Key.Name, currentPath),
		Changes: changes,
	}, nil
}
