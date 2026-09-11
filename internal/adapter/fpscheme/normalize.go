package fpscheme

import (
	"strings"
	"unicode"

	"github.com/boweeb/hasp/internal/domain"
)

// Normalize strips a fingerprint clue down to a canonical, comparable form (T36, D20): colons and
// internal whitespace removed, base64 "=" padding stripped (AWS emits it, ssh-keygen omits it —
// D20 names this explicitly), and a leading "SHA256:"/"MD5:" prefix tolerated regardless of its
// own case.
//
// Hex digits are lowercased; base64 is not (tdd.md §18/T36 says "hex lowercased," deliberately
// not "everything lowercased" — base64's alphabet is case-sensitive, and lowercasing an SSH-
// native SHA-256 fingerprint's base64 body would silently change which bytes it names, not
// merely its spelling). Normalize decides which rule applies by inspecting the stripped clue
// itself: if every remaining character already falls inside the hex alphabet (0-9, a-f, A-F —
// the only characters any real colon-hex fingerprint ever contains), the whole thing is
// lowercased; otherwise it is base64-shaped and left exactly as given.
//
// Normalize performs no interpretation of which scheme a clue belongs to — CandidateSchemes does
// that, deliberately kept separate so a caller can normalize a value it intends to compare
// directly (e.g. against a Scheme's own Compute output) without also asking hasp to guess a
// scheme for it.
func Normalize(clue string) string {
	s := clue
	switch {
	case len(s) >= 7 && strings.EqualFold(s[:7], "SHA256:"):
		s = s[7:]
	case len(s) >= 4 && strings.EqualFold(s[:4], "MD5:"):
		s = s[4:]
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == ':' || unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	stripped := strings.TrimRight(b.String(), "=")

	if isHexCaseInsensitive(stripped) {
		return strings.ToLower(stripped)
	}
	return stripped
}

// sha256Base64Len is the full-shape length of a SHA-256 digest (32 bytes) rendered as base64
// with "=" padding already stripped by Normalize: 44 base64 characters for 32 bytes, minus the
// one trailing "=" a 32-byte input always produces (32 % 3 == 2, so the final base64 group is
// padded by exactly one character). This is the *only* base64-shaped length T36 treats as
// fully determining a scheme; anything else base64-shaped is a fragment (below).
const sha256Base64Len = 43

// CandidateSchemes returns every scheme id a normalized clue's shape admits, per T36's routing
// table. It never returns fewer schemes than the shape allows and never guesses a single
// "correct" one — T36's central rule, stated there in exactly these terms: "shape narrows the
// candidate set; it never uniquely determines a scheme."
//
// Only a clue matching one of T36's three *full* shapes — 40 hex digits, 32 hex digits, or a
// full-length (43-character) base64 SHA-256 digest — narrows to fewer than every registered
// scheme. A **fragment** of any of those shapes returns every registered scheme, not a guess:
// J2 explicitly supports identifying a key from a fingerprint fragment (tdd.md §9's `find` cell;
// `internal/app/key.go`'s `FindKeys` does a substring match), and hex digits are a subset of the
// base64 alphabet, so a short hex-looking fragment is genuinely indistinguishable from a fragment
// of a base64-encoded SHA-256 value — there is no shape signal left to narrow with. Routing a
// fragment to a single scheme anyway reintroduces, one layer down, precisely the silent miss
// D20 exists to eliminate: "shape narrows; it never uniquely determines," applied honestly, means
// a shape that determines nothing narrows to nothing. A clue containing a character outside
// every scheme's alphabet (garbage, or empty after normalization) still returns nil — that is not
// ambiguity, it is certainty that no scheme can ever match.
//
// clue must already be normalized (Normalize) before it is passed here — CandidateSchemes does
// not normalize its input, so a caller can route the same normalized value it also intends to
// compare against a computed fingerprint, without paying for normalization twice.
func CandidateSchemes(normalizedClue string) []domain.SchemeID {
	switch {
	case isHexCaseInsensitive(normalizedClue) && len(normalizedClue) == 40:
		// 40 hex digits (59 characters before normalization strips 19 colons): SHA-1,
		// T35's created-RSA scheme. One candidate.
		return []domain.SchemeID{domain.SchemeAWSCreatedRSA}
	case isHexCaseInsensitive(normalizedClue) && len(normalizedClue) == 32:
		// 32 hex digits (47 characters before normalization strips 15 colons): MD5 — two
		// candidates that share this shape and differ in value (T35/T36's MD5 collision).
		return []domain.SchemeID{domain.SchemeAWSImportedRSA, domain.SchemeLegacySSHMD5}
	case isBase64ish(normalizedClue) && len(normalizedClue) == sha256Base64Len:
		// Full-length SHA-256, base64-encoded, with or without a stripped "SHA256:" prefix:
		// the SSH-native scheme. One candidate.
		return []domain.SchemeID{domain.SchemeSSHNativeSHA256}
	case isBase64ish(normalizedClue) && len(normalizedClue) > 0:
		// A fragment: every character is valid in some scheme's alphabet, but the length
		// matches none of the three full shapes above, so shape narrows nothing. Every
		// registered scheme stays a candidate — see the doc comment above.
		return allSchemeIDs()
	default:
		return nil
	}
}

// allSchemeIDs returns every registered scheme's id, in registry order. Reading directly from
// the registry (scheme.go) rather than hardcoding the four current ids keeps T35's "the registry
// is open — a new scheme is one entry, not a new call site" property true here too: a fifth
// registered scheme is automatically included in the fragment case above with no further change.
func allSchemeIDs() []domain.SchemeID {
	ids := make([]domain.SchemeID, len(registry))
	for i, s := range registry {
		ids[i] = s.ID
	}
	return ids
}

func isHexCaseInsensitive(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// isBase64ish reports whether s is entirely drawn from the base64 alphabet (after Normalize has
// already stripped "=" padding). It is deliberately permissive rather than a strict
// decode-and-verify — T36's routing job is narrowing a candidate set cheaply, not validating a
// clue's exact byte length; a value that passes this check but fails to match anything when
// actually compared is simply a non-match, the ordinary "no key found" outcome.
func isBase64ish(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '+' || r == '/' || r == '-' || r == '_': // std and URL-safe alphabets both
		default:
			return false
		}
	}
	return true
}
