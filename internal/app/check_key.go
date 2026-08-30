package app

import (
	"fmt"
	"sort"

	"github.com/boweeb/hasp/internal/domain"
)

// checkKeys runs every key-scoped detector against the Machine's derived keys.
func checkKeys(keys []domain.Key) []Finding {
	var findings []Finding
	findings = append(findings, detectDuplicateKeyConfirmed(keys)...)
	findings = append(findings, detectDuplicateKeyUnconfirmed(keys)...)
	findings = append(findings, detectKeyMissingPublicHalf(keys)...)
	findings = append(findings, detectKeyNoProfile(keys)...)
	findings = append(findings, detectFingerprintUnknown(keys)...)
	return findings
}

// detectDuplicateKeyConfirmed fires for a Key with two or more non-alias Locations — the same
// fingerprint means the same secret; a second independent copy of it is a confirmed duplicate
// (T12), distinct from an intentional alias (D5, a symlink, never counted here).
func detectDuplicateKeyConfirmed(keys []domain.Key) []Finding {
	var findings []Finding
	for _, k := range keys {
		var paths []string
		for _, loc := range k.Locations {
			if !loc.IsAlias {
				paths = append(paths, loc.Path)
			}
		}
		if len(paths) < 2 {
			continue
		}
		sort.Strings(paths)
		findings = append(findings, newFinding(
			FindingDuplicateKeyConfirmed,
			Subject{Kind: "key", Name: string(k.Name)},
			fmt.Sprintf("key %q exists as %d independent copies sharing one fingerprint", k.Name, len(paths)),
			map[string]any{"paths": paths},
		))
	}
	return findings
}

// detectDuplicateKeyUnconfirmed groups path-identified (undecidable) keys by (Format,
// SizeBytes) — the only two facts still derivable without a passphrase for such a key (T1) — and
// flags every group of two or more as a possible, unconfirmable duplicate (T12): hasp cannot
// prove two undecidable files are the same secret, so it reports the suspicion rather than
// merging or ignoring it (P1).
func detectDuplicateKeyUnconfirmed(keys []domain.Key) []Finding {
	type groupKey struct {
		format domain.KeyFormat
		size   int64
	}
	groups := map[groupKey][]domain.Key{}
	var order []groupKey
	for _, k := range keys {
		if k.Identity.Kind() != domain.IdentityPath {
			continue
		}
		gk := groupKey{format: k.Format, size: k.SizeBytes}
		if _, ok := groups[gk]; !ok {
			order = append(order, gk)
		}
		groups[gk] = append(groups[gk], k)
	}

	var findings []Finding
	for _, gk := range order {
		group := groups[gk]
		if len(group) < 2 {
			continue
		}
		var allPaths []string
		for _, k := range group {
			for _, loc := range k.Locations {
				allPaths = append(allPaths, loc.Path)
			}
		}
		sort.Strings(allPaths)

		for _, k := range group {
			findings = append(findings, newFinding(
				FindingDuplicateKeyUnconfirmed,
				Subject{Kind: "key", Name: string(k.Name)},
				"possible duplicate, cannot confirm: fingerprint is underivable",
				map[string]any{"paths": allPaths},
			))
		}
	}
	return findings
}

func detectKeyMissingPublicHalf(keys []domain.Key) []Finding {
	var findings []Finding
	for _, k := range keys {
		if k.HasPublicHalf {
			continue
		}
		findings = append(findings, newFinding(
			FindingKeyMissingPublicHalf,
			Subject{Kind: "key", Name: string(k.Name)},
			fmt.Sprintf("key %q has no .pub file", k.Name),
			nil,
		))
	}
	return findings
}

func detectKeyNoProfile(keys []domain.Key) []Finding {
	var findings []Finding
	for _, k := range keys {
		if len(k.Profiles) > 0 {
			continue
		}
		findings = append(findings, newFinding(
			FindingKeyNoProfile,
			Subject{Kind: "key", Name: string(k.Name)},
			fmt.Sprintf("key %q belongs to no profile", k.Name),
			nil,
		))
	}
	return findings
}

func detectFingerprintUnknown(keys []domain.Key) []Finding {
	var findings []Finding
	for _, k := range keys {
		if k.Identity.Kind() != domain.IdentityPath {
			continue
		}
		findings = append(findings, newFinding(
			FindingFingerprintUnknown,
			Subject{Kind: "key", Name: string(k.Name)},
			fmt.Sprintf("key %q's fingerprint cannot be derived without a passphrase", k.Name),
			nil,
		))
	}
	return findings
}
