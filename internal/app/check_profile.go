package app

import (
	"fmt"
	"path/filepath"

	"github.com/boweeb/hasp/internal/domain"
)

// checkProfiles runs every profile-scoped detector.
func checkProfiles(profiles []domain.Profile, keys []domain.Key) []Finding {
	counts := countKeysByDir(keys)
	var findings []Finding
	findings = append(findings, detectEmptyProfileDir(profiles, counts)...)
	findings = append(findings, detectUnmarkedProfileDir(profiles, counts)...)
	return findings
}

// countKeysByDir counts non-alias key locations per containing directory — an alias symlink
// alone doesn't make a directory "hold" a key in the sense either profile detector below cares
// about; it's a discoverability shortcut (D5), not storage.
func countKeysByDir(keys []domain.Key) map[string]int {
	counts := map[string]int{}
	for _, k := range keys {
		for _, loc := range k.Locations {
			if loc.IsAlias {
				continue
			}
			counts[filepath.Dir(loc.Path)]++
		}
	}
	return counts
}

// detectEmptyProfileDir fires for a managed profile (carries .hasp) holding zero keys.
func detectEmptyProfileDir(profiles []domain.Profile, counts map[string]int) []Finding {
	var findings []Finding
	for _, p := range profiles {
		if !p.Managed || counts[p.Dir] > 0 {
			continue
		}
		findings = append(findings, newFinding(
			FindingEmptyProfileDir,
			Subject{Kind: "profile", Name: p.Path.String()},
			fmt.Sprintf("profile %q holds no keys", p.Path),
			nil,
		))
	}
	return findings
}

// detectUnmarkedProfileDir fires for a directory holding keys but carrying no .hasp marker — it
// looks like a profile but was never made one.
func detectUnmarkedProfileDir(profiles []domain.Profile, counts map[string]int) []Finding {
	var findings []Finding
	for _, p := range profiles {
		if p.Managed || counts[p.Dir] == 0 {
			continue
		}
		findings = append(findings, newFinding(
			FindingUnmarkedProfileDir,
			Subject{Kind: "profile", Name: p.Path.String()},
			fmt.Sprintf("directory %q holds keys but carries no .hasp marker", p.Path),
			nil,
		))
	}
	return findings
}
