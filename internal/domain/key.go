package domain

// KeyName is a key's stable handle within hasp: unique, human-chosen, and by default the
// filename it was discovered under. It survives rename — KeyIdentity does not necessarily
// (T12): renaming a path-identified key's file changes its KeyIdentity by definition, but its
// KeyName is the handle that stays stable across the move.
type KeyName string

// KeyFormat is the on-disk encoding of a private key file.
type KeyFormat int

const (
	FormatUnknown KeyFormat = iota
	FormatOpenSSH
	FormatPKCS1
	FormatPKCS8
	FormatSEC1
	FormatDSA
)

func (f KeyFormat) String() string {
	switch f {
	case FormatOpenSSH:
		return "openssh"
	case FormatPKCS1:
		return "pkcs1"
	case FormatPKCS8:
		return "pkcs8"
	case FormatSEC1:
		return "sec1"
	case FormatDSA:
		return "dsa" // legacy OpenSSL "DSA PRIVATE KEY" PEM, read-only (T23)
	default:
		return "unknown"
	}
}

func (f KeyFormat) MarshalJSON() ([]byte, error) { return marshalStringer(f) }

// KeyLocation is one absolute path at which a Key's material or an alias to it is found.
//
// This is a deliberate, documented departure from tdd.md §3's literal sketch, which types
// Key.Locations as []string: T12/D5 require distinguishing a location that is a real,
// independent copy of the key's bytes from one that is a symlink alias to another location —
// the former is what check's duplicate-key-confirmed finding warns about (two files sharing a
// fingerprint), the latter is D5's ordinary, intentional alias mechanism, and the two must never
// be conflated. A bare []string cannot carry that distinction.
type KeyLocation struct {
	Path    string `json:"path"`    // absolute
	IsAlias bool   `json:"isAlias"` // true when this location is a symlink to another location (D5)
}

// Key is a cryptographic keypair — the first-class object (design.md §5.1).
type Key struct {
	Identity      KeyIdentity   `json:"identity"` // fingerprint or path (§5, T12)
	Locations     []KeyLocation `json:"locations"`
	Name          KeyName       `json:"name"`
	Algorithm     string        `json:"algorithm"` // "ssh-ed25519", "ssh-rsa", ...; "" if undecidable
	Bits          int           `json:"bits"`       // 0 when not applicable or undecidable
	Comment       string        `json:"comment"`    // "" if unreadable
	Format        KeyFormat     `json:"format"`
	Encrypted     bool          `json:"encrypted"`
	HasPublicHalf bool          `json:"hasPublicHalf"`
	Profiles      []ProfilePath `json:"profiles"` // derived, never declared (D13); may be empty
}
