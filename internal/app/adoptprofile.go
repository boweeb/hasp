package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/boweeb/hasp/internal/domain"
)

// AdoptProfileRequest is `adopt profile`'s input (tdd.md §9's `adopt` grid cell: "Add a `.hasp`
// marker to an existing directory that already holds keys", D13). Unlike AdoptKeyRequest, there
// is no Source to resolve — the target directory is built entirely from KeyDir + Profile, exactly
// as NewKeyRequest's own directory is (newkey.go).
type AdoptProfileRequest struct {
	KeyDir  string
	Profile domain.ProfilePath
}

// AdoptProfileUseCase marks an existing directory as a managed profile by writing its .hasp
// marker (tdd.md §9's `adopt` grid cell, D13, D15). Simpler than AdoptKeyUseCase: nothing moves,
// no top-level alias is needed, and the whole Plan collapses to the one CreateMarker Change
// change_marker.go already provides — no new WriteFS primitive is needed for this verb.
type AdoptProfileUseCase struct{}

// Plan builds adopt profile's plan: a single CreateMarker Change, refusing before anything is
// planned if either of the grid cell's own preconditions is unmet — the target directory must
// already exist ("an existing directory that already holds keys" is the grid cell's own framing;
// mkdir-p is explicitly out of scope, mirroring CreateMarker's own doc comment), and it must not
// already carry a marker (adopting an already-managed directory a second time is a usage mistake,
// not a defect CreateMarker's own ErrMarkerExists needs to surface through Apply — checking here
// lets Plan refuse with a clearer, adopt-specific message before any Change is even built, the
// same shape AdoptKeyUseCase.Plan uses for requireManagedProfileDir).
func (uc AdoptProfileUseCase) Plan(req AdoptProfileRequest) (Plan, error) {
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
	// T8 / tdd.md §12, mirroring every other write-path Request's own defense-in-depth check
	// (newkey.go, adoptkey.go): refuse a target outside req.KeyDir even if a future construction
	// bug reaches this point with an already-malformed profile path.
	if err := requireWithinKeyDir(req.KeyDir, dir); err != nil {
		return Plan{}, err
	}

	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Plan{}, fmt.Errorf("%w: profile directory %s does not exist; adopt profile only marks an existing directory, it never creates one", ErrUsage, dir)
		}
		return Plan{}, fmt.Errorf("check %s: %w", dir, err)
	}
	if !info.IsDir() {
		return Plan{}, fmt.Errorf("%w: %s is not a directory", ErrUsage, dir)
	}

	marker := filepath.Join(dir, markerFileName)
	if _, err := os.Lstat(marker); err == nil {
		return Plan{}, fmt.Errorf("%w: profile directory %s is already managed (a %s marker is already present)", ErrUsage, dir, markerFileName)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Plan{}, fmt.Errorf("check %s: %w", marker, err)
	}

	return Plan{
		Summary: fmt.Sprintf("adopt profile %s", req.Profile),
		Changes: []Change{CreateMarker{Dir: dir, Header: []byte(profileMarkerHeader)}},
	}, nil
}
