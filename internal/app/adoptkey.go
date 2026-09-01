package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/boweeb/hasp/internal/domain"
)

// AdoptKeyRequest is `adopt key`'s input (tdd.md §9's `adopt` grid cell, D13, D14). Source is the
// key's current, single, non-alias location — resolved by the caller (internal/cli, via
// app.ShowKey/app.FindKeys against a derived Machine) from whatever name-or-clue the user typed,
// exactly as NewKeyRequest's own doc comment keeps internal/app ignorant of how a CLI argument
// became a path (tdd.md §3's Request-only framing).
type AdoptKeyRequest struct {
	KeyDir  string
	Profile domain.ProfilePath // the managed profile to adopt the key into; required, non-empty
	Source  string             // absolute path to the key's current real (non-alias) location
}

// AdoptKeyUseCase moves an unmanaged key into a managed profile directory, leaving a top-level
// alias behind so ssh's default identity probing keeps finding it (D13, tdd.md §9's `adopt` grid
// cell).
type AdoptKeyUseCase struct{}

// Plan builds adopt's plan: a ReplaceWithSymlink Change, From=req.Source, To=the key's new path
// inside the target profile directory — plus a second ReplaceWithSymlink for the key's .pub
// sidecar, if one exists at req.Source+".pub" (as it always does for a `new key`-generated key).
// This second Change is necessary, not optional polish: keyfile.Inspect derives HasPublicHalf and
// Comment from a `.pub` file sitting next to the *resolved* (real, non-alias) key path, so a
// Plan that relocated only the private key file would silently leave those facts wrong the moment
// the key lands in its new, managed location — a regression discovered while proving this exact
// use case's own CLI round trip, not a hypothetical.
//
// This does not reopen §4's "Change ordering" question: it is not the naive
// [MoveFile, CreateSymlink] shape §4 walks through and rejects (both Changes here are themselves
// the single atomic copy-verify-replace primitive §11 point 4 describes, applied twice to two
// independent, unrelated paths — the private key and its .pub sidecar never interact, so there is
// no unsafe intermediate state to order around regardless of which of the two runs first).
func (uc AdoptKeyUseCase) Plan(req AdoptKeyRequest) (Plan, error) {
	if req.KeyDir == "" {
		return Plan{}, fmt.Errorf("%w: a key directory is required", ErrUsage)
	}
	if req.Source == "" {
		return Plan{}, fmt.Errorf("%w: a source key location is required", ErrUsage)
	}
	if len(req.Profile) == 0 {
		return Plan{}, fmt.Errorf("%w: a target profile is required", ErrUsage)
	}
	for _, seg := range req.Profile {
		if err := validatePathSegment("profile segment", seg); err != nil {
			return Plan{}, err
		}
	}

	// T8 / tdd.md §12: "hasp writes nothing outside the key directory." req.Source arrives here
	// from the CLI layer with no shape guarantee of its own — validate it here, in internal/app,
	// independent of whatever internal/cli did or didn't check, exactly as newkey.go's own comment
	// explains for req.Name/req.Profile.
	if err := requireWithinKeyDir(req.KeyDir, req.Source); err != nil {
		return Plan{}, err
	}

	profileDir := filepath.Join(append([]string{req.KeyDir}, []string(req.Profile)...)...)

	// adopt's entire point is establishing profile membership through location (D13); the
	// derivation pipeline only attributes a profile to a key when its parent directory is itself
	// Managed (attachKeyProfiles, T19's read side) — so adopting into an unmarked directory would
	// silently produce a key with no profile membership at all, defeating adopt without ever
	// raising an error. Refuse before planning anything.
	if err := requireManagedProfileDir(profileDir); err != nil {
		return Plan{}, err
	}

	dest := filepath.Join(profileDir, filepath.Base(req.Source))
	// Defense in depth alongside validatePathSegment above, mirroring newkey.go's own
	// requireWithinKeyDir call: refuse to plan a write outside req.KeyDir even if a future bug
	// reaches this point with an already-malformed path.
	if err := requireWithinKeyDir(req.KeyDir, dest); err != nil {
		return Plan{}, err
	}
	if filepath.Clean(dest) == filepath.Clean(req.Source) {
		return Plan{}, fmt.Errorf("%w: %s already lives in profile %s", ErrUsage, filepath.Base(req.Source), req.Profile)
	}

	changes := []Change{ReplaceWithSymlink{From: req.Source, To: dest}}
	if hasSidecar(req.Source + ".pub") {
		changes = append(changes, ReplaceWithSymlink{From: req.Source + ".pub", To: dest + ".pub"})
	}

	return Plan{
		Summary: fmt.Sprintf("adopt %s into profile %s", filepath.Base(req.Source), req.Profile),
		Changes: changes,
	}, nil
}

// hasSidecar reports whether path exists — a read, safe under P5 — used to decide whether adopt's
// and release's Plans need a second Change for a key's .pub sidecar. Not every key has one (T1's
// derivation matrix; a legacy or undecidable key may be private-key-only), so this check must be
// conditional rather than assumed.
func hasSidecar(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// requireManagedProfileDir confirms dir already carries hasp's .hasp marker (D13, D15) — a read,
// safe under P5, performed before any Change is planned. Shared by AdoptKeyUseCase (the
// destination must already be managed) and ReleaseKeyUseCase (the source must already be
// managed, since release only makes sense for a key adopt actually placed).
func requireManagedProfileDir(dir string) error {
	marker := filepath.Join(dir, markerFileName)
	if _, err := os.Lstat(marker); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: profile directory %s is not managed (no %s marker present)", ErrUsage, dir, markerFileName)
		}
		return fmt.Errorf("check %s: %w", marker, err)
	}
	return nil
}
