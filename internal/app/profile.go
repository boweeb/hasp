package app

import (
	"strings"

	"github.com/boweeb/hasp/internal/domain"
)

// ListProfiles returns every profile candidate, managed or not (check's empty-profile-dir and
// unmarked-profile-dir findings both depend on unmanaged candidates staying visible here too).
func ListProfiles(m Machine) []domain.Profile {
	return m.Profiles
}

// KeyRow and HostRow attribute an aggregated row to the profile it actually came from — required
// under D2/T19: a key reachable from both work.foobarco and personal appears in the work subtree
// *and* is visibly also personal, which is the distinction T21 says decides whether it gets
// revoked on offboarding (J8).
type KeyRow struct {
	Key         domain.Key
	FromProfile domain.ProfilePath
}

type HostRow struct {
	Host        domain.Host
	FromProfile domain.ProfilePath
}

// ProfileDetail is show profile's answer: the profile itself, plus every key and host attributed
// to it (and, by default, to its descendants — T21).
type ProfileDetail struct {
	Profile domain.Profile
	Keys    []KeyRow
	Hosts   []HostRow
}

// ShowProfile aggregates the full subtree by default (T21): keys and hosts from path and every
// descendant, each row attributed to the profile it actually came from. recurse=false restricts
// the view to path alone. Only a managed profile (carries .hasp) can be the subject — an
// unmarked container directory is not addressable this way, though its descendants still
// aggregate under whichever managed ancestor recursion reaches (T21).
func ShowProfile(m Machine, path domain.ProfilePath, recurse bool) (ProfileDetail, bool) {
	var subject *domain.Profile
	for i := range m.Profiles {
		if m.Profiles[i].Path.Equal(path) {
			subject = &m.Profiles[i]
			break
		}
	}
	if subject == nil || !subject.Managed {
		return ProfileDetail{}, false
	}

	matches := func(p domain.ProfilePath) bool {
		if recurse {
			return p.IsDescendantOf(path)
		}
		return p.Equal(path)
	}

	var keys []KeyRow
	for _, k := range m.Keys {
		for _, p := range k.Profiles {
			if matches(p) {
				keys = append(keys, KeyRow{Key: k, FromProfile: p})
			}
		}
	}

	var hosts []HostRow
	for _, h := range m.Hosts {
		for _, p := range h.Profiles {
			if matches(p) {
				hosts = append(hosts, HostRow{Host: h, FromProfile: p})
			}
		}
	}

	return ProfileDetail{Profile: *subject, Keys: keys, Hosts: hosts}, true
}

// FindProfiles matches by name fragment, case-insensitive, against the dotted profile path.
func FindProfiles(m Machine, clue string) []domain.Profile {
	lower := strings.ToLower(clue)
	var out []domain.Profile
	for _, p := range m.Profiles {
		if strings.Contains(strings.ToLower(p.Path.String()), lower) {
			out = append(out, p)
		}
	}
	return out
}
