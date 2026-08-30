package domain

import "encoding/json"

// Fingerprint is ssh.FingerprintSHA256 output — the "SHA256:..." form OpenSSH >=6.8 uses (T1).
// "" means the fingerprint is undecidable: a legacy-PEM, encrypted key with no public half
// (design.md §5.1's derivation gap). hasp reports that fact as unknown rather than guessing.
type Fingerprint string

// IdentityKind distinguishes the two shapes a KeyIdentity can take, for rendering (T12).
type IdentityKind int

const (
	IdentityFingerprint IdentityKind = iota
	IdentityPath
)

func (k IdentityKind) String() string {
	switch k {
	case IdentityFingerprint:
		return "fingerprint"
	case IdentityPath:
		return "path"
	default:
		return "unknown"
	}
}

// KeyIdentity is a key's intrinsic identity: the fingerprint when derivable, else the resolved
// (symlink-followed), absolute path (T12). Two files with the same fingerprint are one key with
// two locations; a path-identified key can never be deduplicated against another, because hasp
// cannot prove two undecidable files are the same secret (P1).
//
// identityKey is unexported so only this package can produce a KeyIdentity — construct one via
// NewFingerprintIdentity or NewPathIdentity.
type KeyIdentity interface {
	identityKey() string
	// Kind reports which case this identity is.
	Kind() IdentityKind
	// Value is the fingerprint string or the resolved path, depending on Kind.
	Value() string
	// String is Value, for convenient human-readable rendering.
	String() string
	json.Marshaler
}

// NewFingerprintIdentity builds a fingerprint-based KeyIdentity.
func NewFingerprintIdentity(f Fingerprint) KeyIdentity { return byFingerprint(f) }

// NewPathIdentity builds a path-based KeyIdentity from an already-resolved (symlink-followed),
// absolute path. Callers are responsible for resolving the path first (T12) — this constructor
// does not touch the filesystem.
func NewPathIdentity(resolvedAbsPath string) KeyIdentity { return byPath(resolvedAbsPath) }

type byFingerprint Fingerprint

func (f byFingerprint) identityKey() string  { return "fp:" + string(f) }
func (f byFingerprint) Kind() IdentityKind   { return IdentityFingerprint }
func (f byFingerprint) Value() string        { return string(f) }
func (f byFingerprint) String() string       { return string(f) }
func (f byFingerprint) MarshalJSON() ([]byte, error) {
	return json.Marshal(identityView{Kind: IdentityFingerprint.String(), Value: string(f)})
}

type byPath string // resolved, absolute

func (p byPath) identityKey() string { return "path:" + string(p) }
func (p byPath) Kind() IdentityKind  { return IdentityPath }
func (p byPath) Value() string       { return string(p) }
func (p byPath) String() string      { return string(p) }
func (p byPath) MarshalJSON() ([]byte, error) {
	return json.Marshal(identityView{Kind: IdentityPath.String(), Value: string(p)})
}

type identityView struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}
