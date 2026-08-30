package domain

import "strings"

// ProfilePath is a hierarchical profile name written as a path: ["work", "foobarco"] for the
// profile "work.foobarco" (design.md §5.3). Name segments map to directory-path segments one to
// one under the key directory.
type ProfilePath []string

// ParseProfilePath splits a dotted profile name ("work.foobarco") into its ProfilePath segments.
// The inverse of ProfilePath.String.
func ParseProfilePath(s string) ProfilePath {
	if s == "" {
		return nil
	}
	return strings.Split(s, ".")
}

func (p ProfilePath) String() string { return strings.Join(p, ".") }

func (p ProfilePath) MarshalJSON() ([]byte, error) { return marshalStringer(p) }

// Equal reports whether p and other name the same profile.
func (p ProfilePath) Equal(other ProfilePath) bool {
	if len(p) != len(other) {
		return false
	}
	for i := range p {
		if p[i] != other[i] {
			return false
		}
	}
	return true
}

// IsDescendantOf reports whether p is ancestor's exact profile or lives anywhere in its subtree
// — used by "show profile"'s default recursion (T21): work.foobarco and work.acme are both
// descendants of work, and of themselves.
func (p ProfilePath) IsDescendantOf(ancestor ProfilePath) bool {
	if len(p) < len(ancestor) {
		return false
	}
	for i := range ancestor {
		if p[i] != ancestor[i] {
			return false
		}
	}
	return true
}

// Profile is a persona: who you are being (design.md §5.3). "work.foobarco" is a profile; "work" is
// its parent and is also a profile if it too carries a .hasp marker.
type Profile struct {
	Path    ProfilePath `json:"path"`
	Dir     string      `json:"dir"`     // absolute path; ProfilePath joined onto the key directory
	Managed bool        `json:"managed"` // carries a .hasp marker? (§5.3, D13, D15)
}
