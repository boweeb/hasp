package app

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/boweeb/hasp/internal/adapter/fpscheme"
	"github.com/boweeb/hasp/internal/adapter/keyfile"
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
	Key   domain.Key    `json:"key"`
	Hosts []domain.Host `json:"hosts"`
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

// FindMatch is find key's answer for one key (D20, T36, T48): the key itself, plus the match
// evidence — which scheme(s) the clue matched under, expressed as T37's own Origin shape.
// Which scheme matched is the origin evidence for only two of the four registered schemes
// (T36): a match under aws-created-rsa or aws-imported-rsa is deterministic proof of provenance
// (Origin.ID is the AWS-EC2 created/imported id, Confidence confirmed); a match under
// ssh-native-sha256 or legacy-ssh-md5 proves nothing about provenance — it only proves this is
// the right key — so Origin.ID names the scheme itself (there is no provenance claim to make)
// and Confidence stays possible. The renderer must not flatten the two into a single "matched"
// verdict (T36's own words).
type FindMatch struct {
	Key     domain.Key      `json:"key"`
	Origins []domain.Origin `json:"origins"`
}

// FindKeys identifies every key matching a fingerprint clue, across every scheme the clue's
// shape admits (J2, D20, T36) — not only hasp's own SSH-native scheme. Normalization and shape
// routing are internal/adapter/fpscheme's own (Normalize, CandidateSchemes, M3.6.1); FindKeys
// consumes them rather than reimplementing either, and computes only the schemes a clue's shape
// admits, not every scheme for every key, so the common case (an SSH-native clue) computes the
// fewest schemes it can (T36) — see candidatesNeedMaterial below for what that shape also buys in
// disk I/O, which is no longer the same claim (tdd.md §18, item 4, M3.6.3 review).
//
// find key never prompts for a passphrase (T39, T48, the consent boundary D19 draws): the
// aws-created-rsa scheme, which needs the decrypted private key, is only ever evaluated against
// a key that is already readable with none — an unencrypted key, legitimate under D19 because
// the user explicitly handed hasp a SHA-1-shaped clue. An encrypted key can never satisfy that
// scheme through find, and rather than let that read as a silent "you don't have this key" —
// exactly the failure D20 exists to eliminate — the second return value names the scheme and how
// many keys could not be evaluated against it (T48).
func FindKeys(m Machine, clue string) (matches []FindMatch, warnings []string) {
	normalized := fpscheme.Normalize(clue)
	if normalized == "" {
		return nil, nil
	}
	candidates := fpscheme.CandidateSchemes(normalized)
	if len(candidates) == 0 {
		return nil, nil
	}

	if !candidatesNeedMaterial(candidates) {
		// The only shape that reaches here is the full-length base64 SHA-256 clue, whose sole
		// candidate is ssh-native-sha256 (fpscheme.CandidateSchemes). D19/P3 license reading
		// private key material only for an operation that needs it (item 4, M3.6.3 review); this
		// shape does not, because ssh-native-sha256's Compute is byte-for-byte
		// ssh.FingerprintSHA256 over the public key — exactly the value deriveKeys already stored
		// as k.Identity (keyfile.Inspect, pipeline.go) — so no key's file is reopened here at
		// all, restoring FindKeys' pre-M3.6.3 zero-disk-I/O behavior for this shape rather than
		// merely reducing it. A key with no derivable fingerprint at all (Identity.Kind() ==
		// IdentityPath, T1's undecidable row) is skipped, not reopened: Inspect already tried the
		// identical derivation OpenMaterial(path, nil) would repeat and came back empty, so
		// reopening it can only reproduce that same "no public key" answer, never a match.
		id := candidates[0]
		for _, k := range m.Keys {
			if k.Identity.Kind() != domain.IdentityFingerprint {
				continue
			}
			value := k.Identity.Value()
			if value == "" || !strings.Contains(compareFold(value), compareFold(normalized)) {
				continue
			}
			matches = append(matches, FindMatch{Key: k, Origins: []domain.Origin{originForSchemeMatch(id)}})
		}
		return matches, nil
	}

	unevaluable := map[domain.SchemeID]int{}
	for _, k := range m.Keys {
		path := primaryKeyPath(k)
		if path == "" {
			continue
		}
		// No passphrase, ever (T39, T48) — find degrades to an honest "could not evaluate"
		// rather than interrupting the user; §11's fail-open stance for a bad file applies here
		// exactly as it does to the rest of a plain read.
		mat, err := keyfile.OpenMaterial(path, nil)
		if err != nil {
			continue
		}

		var origins []domain.Origin
		for _, id := range candidates {
			sf, ok := fpscheme.ComputeOne(id, mat)
			if !ok {
				continue
			}
			if sf.Reason == domain.ReasonPrivateKeyUnavailable {
				unevaluable[id]++
			}
			if sf.Value == "" || !strings.Contains(compareFold(sf.Value), compareFold(normalized)) {
				continue
			}
			origins = append(origins, originForSchemeMatch(id))
		}
		if len(origins) > 0 {
			matches = append(matches, FindMatch{Key: k, Origins: origins})
		}
	}

	return matches, unevaluableWarnings(candidates, unevaluable)
}

// candidatesNeedMaterial reports whether evaluating clue's candidate schemes against a key
// requires opening that key's file at all (item 4, M3.6.3 review; D19, P3, P8: hasp may read
// private key material only for an operation that needs it, and P8's one-laptop scale is what
// makes the pre-fix N-reads-per-invocation regression cheap in absolute terms without making it
// correct in principle). True whenever a candidate is declared RequiresDecryptedPrivateKey
// (fpscheme.Schemes' Requires field, M3.6.1) — aws-created-rsa, the one scheme find can still
// never actually satisfy for an encrypted key (T48) — or is an AWS scheme at all:
// aws-imported-rsa is declared RequiresPublicHalf, but what it hashes (the PKIX/SPKI DER of the
// public key) is not the value k.Identity already carries (that value is always
// ssh-native-sha256's own SHA-256-over-SSH-wire-format encoding, T1), so Identity cannot stand in
// for it the way it can for ssh-native-sha256 itself. legacy-ssh-md5 never appears as a candidate
// without aws-imported-rsa alongside it (fpscheme.CandidateSchemes' MD5-shape and fragment
// routing), so naming the two AWS schemes here is sufficient without naming it too.
func candidatesNeedMaterial(candidates []domain.SchemeID) bool {
	requires := make(map[domain.SchemeID]fpscheme.MaterialRequirement, len(fpscheme.Schemes()))
	for _, s := range fpscheme.Schemes() {
		requires[s.ID] = s.Requires
	}
	for _, id := range candidates {
		if requires[id] == fpscheme.RequiresDecryptedPrivateKey {
			return true
		}
		switch id {
		case domain.SchemeAWSCreatedRSA, domain.SchemeAWSImportedRSA:
			return true
		}
	}
	return false
}

// compareFold strips exactly ":", whitespace, "-", and "_" and lowercases letters, so a scheme's
// computed value and a user-typed clue compare equal despite two independent kinds of formatting
// noise (T50): case (fpscheme.Normalize deliberately leaves base64 case exactly as given —
// folding case there would silently corrupt which bytes a SHA-256 value names, D20's own
// base64-padding note) and separator punctuation a human pastes in for readability (a hyphen
// grouping a base64 fragment into pronounceable chunks, say), which Normalize also does not strip
// beyond a literal colon, for the identical reason: "-" and "_" are meaningful content in the
// URL-safe base64 alphabet, and CandidateSchemes' shape routing (T36) needs Normalize's output at
// its true, uncorrupted length to route a full-length clue correctly.
//
// The stripped set is deliberately narrower than "every non-alphanumeric rune" (item 3, M3.6.3
// review): T50's own safety argument for folding covers only "-" and "_" — none of the four
// registered schemes (T35) ever emits either as real content, the two colon-hex schemes use only
// hex digits and colons, so this fold can never conflate two genuinely distinct values those
// schemes compute. That argument does not extend to "+" and "/": T50's own text notes the
// SSH-native scheme emits *standard*, not URL-safe, base64, so "+" and "/" are real content in
// its Compute output, and folding them away would be exactly the kind of silent conflation T50
// warns against, just for a different pair of characters — stripping only what the written
// argument actually covers keeps the code matching the invariant it cites rather than
// overreaching past it. compareFold is find's own comparison step, never CandidateSchemes'
// routing input or a scheme's own Compute output — both of those still see Normalize's
// shape-preserving form.
func compareFold(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == ':' || r == '-' || r == '_' || unicode.IsSpace(r):
			continue
		case unicode.IsUpper(r):
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// originForSchemeMatch turns one matched scheme id into T36's origin evidence — see FindMatch's
// own doc comment for why the two branches name different kinds of thing in Origin.ID.
func originForSchemeMatch(id domain.SchemeID) domain.Origin {
	because := []domain.ReasonToken{domain.ReasonToken("scheme=" + string(id))}
	switch id {
	case domain.SchemeAWSCreatedRSA:
		return domain.Origin{ID: domain.OriginAWSEC2Created, Confidence: domain.ConfidenceConfirmed, Because: because}
	case domain.SchemeAWSImportedRSA:
		return domain.Origin{ID: domain.OriginAWSEC2Imported, Confidence: domain.ConfidenceConfirmed, Because: because}
	default:
		// ssh-native-sha256 and legacy-ssh-md5: no provenance claim to make (T36), so the
		// origin's own id is the scheme's id rather than an invented AWS-EC2 guess.
		return domain.Origin{ID: string(id), Confidence: domain.ConfidencePossible, Because: because}
	}
}

// unevaluableWarnings is T48's explicit-warning-over-silent-miss decision made concrete: for
// every candidate scheme the clue's shape admitted, report how many keys could not be evaluated
// against it because the decrypted private key was unavailable — never a silent gap in the
// result (D20).
func unevaluableWarnings(candidates []domain.SchemeID, unevaluable map[domain.SchemeID]int) []string {
	var out []string
	for _, id := range candidates {
		n := unevaluable[id]
		if n == 0 {
			continue
		}
		// tdd.md §10 makes the envelope's warnings array a public contract, frozen at v1.0.0 —
		// this string must not name a scheduling artifact (a milestone/chunk id) that is
		// meaningless to an end user and wrong the moment the chunk is renumbered (item 5,
		// M3.6.3 review). Citing T39 is fine: internal/cli/profile.go:45 already does.
		out = append(out, fmt.Sprintf(
			"%s could not be evaluated for %d encrypted key(s): find key never prompts for a passphrase (T39) — decrypt the key yourself to compare by hand, or use hasp's --investigate flag, once available, to attempt decryption interactively",
			id, n))
	}
	return out
}

// primaryKeyPath returns k's primary (first non-alias) location's path, falling back to the
// first location at all if every one happens to be an alias — projectKey (pipeline.go) always
// sorts a real, non-alias location first when one exists, so the fallback is defensive rather
// than expected to fire.
func primaryKeyPath(k domain.Key) string {
	for _, loc := range k.Locations {
		if !loc.IsAlias {
			return loc.Path
		}
	}
	if len(k.Locations) > 0 {
		return k.Locations[0].Path
	}
	return ""
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
