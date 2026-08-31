package app

import (
	"fmt"
	"path/filepath"
)

// ReleaseKeyRequest is `release key`'s input — the exact inverse of AdoptKeyRequest (D14). Alias
// and Source are both resolved by the caller from a name-or-clue exactly as AdoptKeyRequest.Source
// is (tdd.md §3's Request-only framing): Alias is the top-level symlink adopt left behind, Source
// is the key's real file, currently inside a managed profile directory.
type ReleaseKeyRequest struct {
	KeyDir string
	Alias  string // absolute path to the top-level alias (symlink) adopt left behind
	Source string // absolute path to the key's real file, inside a managed profile directory
}

// ReleaseKeyUseCase moves a managed key back out of its profile directory — the exact inverse of
// AdoptKeyUseCase (D14, tdd.md §9's `release` grid cell). Without this inverse, `adopt` would be
// a one-way door, which D14's own rationale names directly as unacceptable.
type ReleaseKeyUseCase struct{}

// Plan builds release's plan, ordered per §4's rule that the least recoverable step goes last:
//
//  1. ReplaceSymlinkWithFile restores a real, independent copy of Source's bytes at Alias,
//     replacing the symlink that currently sits there. If this fails, Alias is untouched — still
//     a working symlink to Source, exactly the pre-release (adopted) state.
//  2. (If a .pub sidecar exists at Source+".pub" — AdoptKeyUseCase.Plan's own doc comment
//     explains why one usually does) the same restore, mirrored for the sidecar.
//  3. Remove deletes the now-redundant Source copy. If this fails, Alias already holds a
//     complete, valid, independent copy of the key — fully discoverable on its own — and Source
//     is merely a redundant duplicate (a check finding, never a broken or undiscoverable state).
//  4. (If present) the mirrored Remove for the .pub sidecar's own redundant copy.
//
// Both restores are ordered before both removes, deliberately: a restore failing leaves the
// pre-release state fully intact and working, so nothing should be removed until every restore
// this Plan needs has already succeeded.
//
// This is not a single atomic primitive the way adopt's is, because release's terminal step
// (removing Source) is an ordinary, already-backed-up Remove — §11 point 4's alias-preserving
// replace has no mirror-image need to fold cleanup into the same os.Rename the way adopt's does,
// since Alias is fully valid and discoverable the moment step 1 succeeds.
func (uc ReleaseKeyUseCase) Plan(req ReleaseKeyRequest) (Plan, error) {
	if req.KeyDir == "" {
		return Plan{}, fmt.Errorf("%w: a key directory is required", ErrUsage)
	}
	if req.Alias == "" || req.Source == "" {
		return Plan{}, fmt.Errorf("%w: both the alias and the real key location are required", ErrUsage)
	}

	// T8 / tdd.md §12, mirroring AdoptKeyUseCase.Plan's own defense-in-depth reasoning.
	if err := requireWithinKeyDir(req.KeyDir, req.Alias); err != nil {
		return Plan{}, err
	}
	if err := requireWithinKeyDir(req.KeyDir, req.Source); err != nil {
		return Plan{}, err
	}
	if filepath.Clean(req.Alias) == filepath.Clean(req.Source) {
		return Plan{}, fmt.Errorf("%w: the alias and the real key location must be different paths", ErrUsage)
	}

	// release only makes sense for a key adopt actually placed — refuse if Source's own directory
	// was never marked managed in the first place, the mirror image of AdoptKeyUseCase.Plan's own
	// check on the destination.
	if err := requireManagedProfileDir(filepath.Dir(req.Source)); err != nil {
		return Plan{}, err
	}

	restores := []Change{ReplaceSymlinkWithFile{Alias: req.Alias, Source: req.Source}}
	removes := []Change{Remove{Path: req.Source, Reason: "release: redundant profile-directory copy, now duplicated at the released location"}}
	if hasSidecar(req.Source + ".pub") {
		restores = append(restores, ReplaceSymlinkWithFile{Alias: req.Alias + ".pub", Source: req.Source + ".pub"})
		removes = append(removes, Remove{Path: req.Source + ".pub", Reason: "release: redundant profile-directory .pub copy, now duplicated at the released location"})
	}

	return Plan{
		Summary: fmt.Sprintf("release %s from %s", filepath.Base(req.Source), filepath.Dir(req.Source)),
		Changes: append(restores, removes...),
	}, nil
}
