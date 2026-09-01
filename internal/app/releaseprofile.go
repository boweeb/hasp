package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/boweeb/hasp/internal/domain"
)

// ReleaseProfileRequest is `release profile`'s input — the exact inverse of AdoptProfileRequest
// (D14, D18). No Source/Alias to resolve, unlike ReleaseKeyRequest: the target directory is built
// entirely from KeyDir + Profile, exactly as AdoptProfileRequest's is.
type ReleaseProfileRequest struct {
	KeyDir  string
	Profile domain.ProfilePath
}

// ReleaseProfileUseCase unmarks a managed profile directory by removing its .hasp marker (tdd.md
// §9's `release` grid cell, D14, D18). The inverse of AdoptProfileUseCase: nothing about the
// directory's contents changes, only the marker's presence.
type ReleaseProfileUseCase struct{}

// Plan builds release profile's plan: a single Remove Change targeting the directory's .hasp
// marker. Refuses before planning anything if the directory carries no marker to remove (release
// only makes sense for a profile adopt actually marked, mirroring ReleaseKeyUseCase.Plan's own
// requireManagedProfileDir check).
//
// Plan reads the marker's own bytes here — a read, safe under P5 — and carries them into the
// Remove Change's PriorContent, purely so Preview() can render them before Apply ever runs
// (D15 elaboration 4 / D18: "release must show the file's contents in its preview, so that
// removing a marker the user has written in is never a silent loss"). This does not reopen D15's
// presence-not-contents rule: D15 forbids hasp *parsing or acting on* a marker's contents, and
// nothing here does either — the bytes are carried verbatim into a Preview for display and never
// inspected, branched on, or fed into any decision Plan or Apply makes.
//
// The marker also gets a Witness (T30). Without one, the "never a silent loss" guarantee above
// is only as strong as the preview being accurate at the moment it was rendered — a note added to
// the marker during the preview-to-confirm window would still be deleted on Apply, having never
// been shown. The Witness closes that: Applier refuses the whole Plan if the marker's content
// changed between Plan() and Apply(), before Remove ever runs.
func (uc ReleaseProfileUseCase) Plan(req ReleaseProfileRequest) (Plan, error) {
	if req.KeyDir == "" {
		return Plan{}, fmt.Errorf("%w: a key directory is required", ErrUsage)
	}
	if len(req.Profile) == 0 {
		return Plan{}, fmt.Errorf("%w: a profile is required", ErrUsage)
	}
	for _, seg := range req.Profile {
		if err := validatePathSegment("profile segment", seg); err != nil {
			return Plan{}, err
		}
	}

	dir := filepath.Join(append([]string{req.KeyDir}, []string(req.Profile)...)...)
	// T8 / tdd.md §12, mirroring AdoptProfileUseCase.Plan's own defense-in-depth check.
	if err := requireWithinKeyDir(req.KeyDir, dir); err != nil {
		return Plan{}, err
	}

	marker := filepath.Join(dir, markerFileName)
	content, err := os.ReadFile(marker)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Plan{}, fmt.Errorf("%w: profile directory %s is not managed (no %s marker present)", ErrUsage, dir, markerFileName)
		}
		return Plan{}, fmt.Errorf("read %s: %w", marker, err)
	}

	witness, err := NewWitness(marker)
	if err != nil {
		return Plan{}, fmt.Errorf("witness %s: %w", marker, err)
	}

	return Plan{
		Summary: fmt.Sprintf("release profile %s", req.Profile),
		Changes: []Change{Remove{
			Path:         marker,
			Reason:       "release: unmark this directory as a managed profile (D14, D18)",
			PriorContent: content,
		}},
		Witnesses: []Witness{witness},
	}, nil
}
