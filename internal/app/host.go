package app

import (
	"path/filepath"
	"strings"

	"github.com/boweeb/hasp/internal/domain"
)

// ListHosts returns every host, or only those in the named host group when group is non-empty
// (tdd.md §9's --group flag).
func ListHosts(m Machine, group string) []domain.Host {
	if group == "" {
		return m.Hosts
	}
	var out []domain.Host
	for _, h := range m.Hosts {
		if hostGroupName(h.HostGroup) == group {
			out = append(out, h)
		}
	}
	return out
}

// hostGroupName derives a host group's short name from its file path: "work.sshconfig" -> "work"
// (D9's naming rule); the default group's own file has no such suffix to strip.
func hostGroupName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".sshconfig")
}

// ShowHost finds a host by exact pattern match. domain.Host already carries everything full
// detail needs (resolved bindings, derived profiles) — no reverse lookup required, unlike
// ShowKey.
func ShowHost(m Machine, pattern string) (domain.Host, bool) {
	for _, h := range m.Hosts {
		for _, p := range h.Patterns {
			if p == pattern {
				return h, true
			}
		}
	}
	return domain.Host{}, false
}

// FindHosts matches by pattern fragment, or by the name/fingerprint of a bound key (tdd.md §9).
func FindHosts(m Machine, clue string) []domain.Host {
	lowerClue := strings.ToLower(clue)
	normalizedClue := normalizeFingerprintClue(clue)

	matchingKeys := map[string]bool{}
	for _, k := range m.Keys {
		if strings.Contains(strings.ToLower(string(k.Name)), lowerClue) {
			matchingKeys[identityMapKey(k.Identity)] = true
			continue
		}
		if normalizedClue != "" && k.Identity.Kind() == domain.IdentityFingerprint &&
			strings.Contains(normalizeFingerprintClue(k.Identity.Value()), normalizedClue) {
			matchingKeys[identityMapKey(k.Identity)] = true
		}
	}

	var out []domain.Host
	for _, h := range m.Hosts {
		if hostMatchesPattern(h, lowerClue) || hostMatchesKey(h, matchingKeys) {
			out = append(out, h)
		}
	}
	return out
}

func hostMatchesPattern(h domain.Host, lowerClue string) bool {
	for _, p := range h.Patterns {
		if strings.Contains(strings.ToLower(p), lowerClue) {
			return true
		}
	}
	return false
}

func hostMatchesKey(h domain.Host, matchingKeys map[string]bool) bool {
	for _, b := range h.Bindings {
		if matchingKeys[identityMapKey(b.Key)] {
			return true
		}
	}
	return false
}
