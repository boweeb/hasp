package app

import (
	"strings"
	"unicode"

	"github.com/boweeb/hasp/internal/domain"
)

// ListKeys returns every derived key (J1, J3) — the Machine's own list is already the answer;
// this wrapper exists for symmetry with ShowKey/FindKeys and as the one place list-time
// presentation logic (sorting, filtering) would grow if M1's one-screen answer (P7) ever needed
// it.
func ListKeys(m Machine) []domain.Key {
	return m.Keys
}

// KeyDetail is show key's full-detail answer (J8): the key itself, plus every host that binds
// it — a reverse lookup domain.Key alone can't answer, since a Key has no back-reference to the
// hosts that reference it.
type KeyDetail struct {
	Key   domain.Key
	Hosts []domain.Host
}

// ShowKey finds a key by its stable handle (KeyName) and reports every host bound to it.
func ShowKey(m Machine, name string) (KeyDetail, bool) {
	for _, k := range m.Keys {
		if string(k.Name) != name {
			continue
		}
		return KeyDetail{Key: k, Hosts: hostsBindingKey(m.Hosts, k.Identity)}, true
	}
	return KeyDetail{}, false
}

func hostsBindingKey(hosts []domain.Host, identity domain.KeyIdentity) []domain.Host {
	target := identityMapKey(identity)
	var out []domain.Host
	for _, h := range hosts {
		for _, b := range h.Bindings {
			if identityMapKey(b.Key) == target {
				out = append(out, h)
				break
			}
		}
	}
	return out
}

// FindKeys identifies a key from a fingerprint fragment, normalizing punctuation and case (J2):
// "SHA256:abc123", "sha256abc123", and "AB C1 23" (a fragment) all match the same underlying
// value once punctuation is stripped and case is folded.
func FindKeys(m Machine, clue string) []domain.Key {
	normalized := normalizeFingerprintClue(clue)
	if normalized == "" {
		return nil
	}
	var out []domain.Key
	for _, k := range m.Keys {
		if k.Identity.Kind() != domain.IdentityFingerprint {
			continue // J2 is a fingerprint clue; an undecidable key has none to match against
		}
		if strings.Contains(normalizeFingerprintClue(k.Identity.Value()), normalized) {
			out = append(out, k)
		}
	}
	return out
}

// normalizeFingerprintClue lowercases and strips everything but letters and digits, so
// "SHA256:AbC1:23", "sha256-abc1-23", and "abc123" all normalize identically.
func normalizeFingerprintClue(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
