package app

import (
	"path/filepath"
	"sort"

	"github.com/boweeb/hasp/internal/adapter/scan"
	"github.com/boweeb/hasp/internal/domain"
)

// deriveProfiles projects every scan.ProfileCandidate directly into a domain.Profile — this
// step needs no classification beyond what scan already determined (Managed is .hasp presence,
// D15), unlike keys.
func deriveProfiles(keyDir string) ([]domain.Profile, error) {
	candidates, err := scan.Profiles(keyDir)
	if err != nil {
		return nil, err
	}
	profiles := make([]domain.Profile, 0, len(candidates))
	for _, c := range candidates {
		profiles = append(profiles, domain.Profile{
			Path:    domain.ProfilePath(c.Path),
			Dir:     c.Dir,
			Managed: c.Managed,
		})
	}
	return profiles, nil
}

// attachKeyProfiles computes Key.Profiles as the union, over Key.Locations, of each location's
// immediate parent directory's ProfilePath — included only when that parent is itself Managed
// (T19's read side, D13). A key with a real file in one managed profile and an alias symlink in
// a second acquires both: cross-profile membership (D2) is realized purely by where a location
// physically sits, never declared.
func attachKeyProfiles(keys []domain.Key, profiles []domain.Profile) []domain.Key {
	byDir := make(map[string]domain.Profile, len(profiles))
	for _, p := range profiles {
		byDir[p.Dir] = p
	}

	out := make([]domain.Key, len(keys))
	for i, k := range keys {
		seen := map[string]bool{}
		var paths []domain.ProfilePath
		for _, loc := range k.Locations {
			parent := filepath.Dir(loc.Path)
			p, ok := byDir[parent]
			if !ok || !p.Managed {
				continue // an unmarked directory carries no taxonomy (§5.3) — no exception here
			}
			key := p.Path.String()
			if seen[key] {
				continue
			}
			seen[key] = true
			paths = append(paths, p.Path)
		}
		sort.Slice(paths, func(a, b int) bool { return paths[a].String() < paths[b].String() })
		k.Profiles = paths
		out[i] = k
	}
	return out
}
