package fpscheme_test

import (
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fpscheme"
	"github.com/boweeb/hasp/internal/domain"
)

func TestNormalize(t *testing.T) {
	t.Run("colon-hex mixed case is lowercased and colons stripped", func(t *testing.T) {
		got := fpscheme.Normalize("97:47:11:3C:AF:56:47:B3:F9:A9:89:36:6D:CA:BE:0B:33:A0:05:F7")
		want := "9747113caf5647b3f9a989366dcabe0b33a005f7"
		if got != want {
			t.Errorf("Normalize() = %q, want %q", got, want)
		}
	})

	t.Run("internal and surrounding whitespace stripped", func(t *testing.T) {
		got := fpscheme.Normalize("97 47 11 3C AF 56 47 B3 F9 A9 89 36 6D CA BE 0B 33 A0 05 F7")
		want := "9747113caf5647b3f9a989366dcabe0b33a005f7"
		if got != want {
			t.Errorf("Normalize() = %q, want %q", got, want)
		}
	})

	t.Run("SHA256 prefix stripped, any case, base64 case preserved", func(t *testing.T) {
		got := fpscheme.Normalize("SHA256:lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA")
		want := "lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA"
		if got != want {
			t.Errorf("Normalize() = %q, want %q (base64 case must be preserved)", got, want)
		}
	})

	t.Run("lowercase sha256 prefix also stripped", func(t *testing.T) {
		got := fpscheme.Normalize("sha256:lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA")
		want := "lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA"
		if got != want {
			t.Errorf("Normalize() = %q, want %q", got, want)
		}
	})

	t.Run("MD5 prefix stripped", func(t *testing.T) {
		got := fpscheme.Normalize("MD5:34:29:f4:da:3c:db:49:4b:35:ba:c1:c2:cd:2e:75:a8")
		want := "3429f4da3cdb494b35bac1c2cd2e75a8"
		if got != want {
			t.Errorf("Normalize() = %q, want %q", got, want)
		}
	})

	t.Run("base64 padding stripped", func(t *testing.T) {
		got := fpscheme.Normalize("aGVsbG8=")
		want := "aGVsbG8"
		if got != want {
			t.Errorf("Normalize() = %q, want %q", got, want)
		}
	})

	t.Run("already-normalized value is a fixed point", func(t *testing.T) {
		in := "3429f4da3cdb494b35bac1c2cd2e75a8"
		got := fpscheme.Normalize(in)
		if got != in {
			t.Errorf("Normalize(%q) = %q, want unchanged", in, got)
		}
	})
}

func TestCandidateSchemes(t *testing.T) {
	cases := []struct {
		name string
		clue string
		want []domain.SchemeID
	}{
		{
			name: "59-char colon-hex (40 hex digits) -> aws-created-rsa alone",
			clue: "97:47:11:3c:af:56:47:b3:f9:a9:89:36:6d:ca:be:0b:33:a0:05:f7",
			want: []domain.SchemeID{domain.SchemeAWSCreatedRSA},
		},
		{
			name: "47-char colon-hex (32 hex digits) -> both MD5 schemes",
			clue: "a8:e7:45:95:5f:a3:f0:b1:79:6c:c2:f1:d2:80:57:ea",
			want: []domain.SchemeID{domain.SchemeAWSImportedRSA, domain.SchemeLegacySSHMD5},
		},
		{
			name: "base64 with SHA256: prefix -> ssh-native-sha256 alone",
			clue: "SHA256:lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA",
			want: []domain.SchemeID{domain.SchemeSSHNativeSHA256},
		},
		{
			name: "base64 without prefix -> ssh-native-sha256 alone",
			clue: "lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA",
			want: []domain.SchemeID{domain.SchemeSSHNativeSHA256},
		},
		{
			name: "punctuation outside both alphabets -> nil",
			clue: "not@a#fingerprint!",
			want: nil,
		},
		{
			name: "empty clue -> nil",
			clue: "",
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := fpscheme.Normalize(tc.clue)
			got := fpscheme.CandidateSchemes(normalized)
			assertSchemeSet(t, tc.clue, got, tc.want)
		})
	}
}

// allRegisteredSchemeIDs is every scheme id fpscheme currently registers, used below as the
// "shape narrows nothing" expected answer for a fragment clue. Read from fpscheme.Schemes()
// rather than hardcoded, so a fifth registered scheme keeps this test honest automatically.
func allRegisteredSchemeIDs() []domain.SchemeID {
	schemes := fpscheme.Schemes()
	ids := make([]domain.SchemeID, len(schemes))
	for i, s := range schemes {
		ids[i] = s.ID
	}
	return ids
}

func assertSchemeSet(t *testing.T, clue string, got, want []domain.SchemeID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("CandidateSchemes(Normalize(%q)) = %v, want %v", clue, got, want)
	}
	wantSet := map[domain.SchemeID]bool{}
	for _, id := range want {
		wantSet[id] = true
	}
	for _, id := range got {
		if !wantSet[id] {
			t.Errorf("CandidateSchemes(Normalize(%q)): unexpected candidate %s", clue, id)
		}
		delete(wantSet, id)
	}
	if len(wantSet) != 0 {
		t.Errorf("CandidateSchemes(Normalize(%q)): missing candidates: %v", clue, wantSet)
	}
}

// TestCandidateSchemes_FragmentsAreAmbiguous is the review-flagged defect fix: a clue whose shape
// matches none of T36's three *full* shapes must return every registered scheme, never a guess.
// Before the fix, isBase64ish's alphabet silently accepted hex digits (a subset of base64), and
// the base64 branch had no length gate at all — so any fragment fell through to the base64
// branch and was routed to ssh-native-sha256 alone, regardless of which scheme its full value
// actually belonged to. The five clues below are exactly the ones the review reported as
// misrouted; the fifth (a full 32-hex clue) was already correct and stays that way.
func TestCandidateSchemes_FragmentsAreAmbiguous(t *testing.T) {
	all := allRegisteredSchemeIDs()

	cases := []struct {
		name string
		clue string
		want []domain.SchemeID
	}{
		{
			// 8-char fragment of the legacy-ssh-md5 vector (rsa-pem-plain-pub,
			// tdd.md §12): "3429f4da3cdb494b35bac1c2cd2e75a8"[:8].
			name: "8-char hex fragment of legacy-ssh-md5 vector -> every scheme",
			clue: "3429f4da",
			want: all,
		},
		{
			name: "same fragment, colon-hex form -> every scheme",
			clue: "34:29:f4:da",
			want: all,
		},
		{
			// Fragment of the aws-created-rsa vector (rsa-pem-plain-pub):
			// "9747113caf5647b3f9a989366dcabe0b33a005f7"[:12], with stray internal
			// whitespace Normalize also strips.
			name: "whitespace-broken hex fragment of aws-created-rsa vector -> every scheme",
			clue: "97471 13c af56",
			want: all,
		},
		{
			// Fragment of the ssh-native/ED25519 vector (ed25519-openssh-plain-pub):
			// "lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA"[:11].
			name: "base64 fragment of ssh-native-sha256 vector -> every scheme",
			clue: "lqTGTP6KJSQ",
			want: all,
		},
		{
			// Full 32-hex clue (aws-imported-rsa vector, rsa-pem-plain-pub): already
			// correct before this fix, and must stay that way — the MD5 shape's
			// genuine two-way ambiguity, not the fragment case.
			name: "full 32-hex clue -> both MD5 schemes only, unchanged",
			clue: "a8e745955fa3f0b1796cc2f1d28057ea",
			want: []domain.SchemeID{domain.SchemeAWSImportedRSA, domain.SchemeLegacySSHMD5},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := fpscheme.Normalize(tc.clue)
			got := fpscheme.CandidateSchemes(normalized)
			assertSchemeSet(t, tc.clue, got, tc.want)
		})
	}
}

// TestCandidateSchemes_FragmentRoutingIsNeverExclusive is the regression that matters most: a
// fragment of any one of the four committed vectors (tdd.md §12) must route to a candidate set
// that still *contains* the scheme that vector actually belongs to. In particular, a fragment of
// the legacy-ssh-md5 vector must never be routed away from legacy-ssh-md5 — the exact failure the
// review reported (a legacy-MD5 fragment landing on ssh-native-sha256 alone).
func TestCandidateSchemes_FragmentRoutingIsNeverExclusive(t *testing.T) {
	cases := []struct {
		name       string
		fullClue   string // T35/tdd.md §12's committed vector, colons and prefix included
		wantScheme domain.SchemeID
	}{
		{
			name:       "aws-created-rsa vector fragment",
			fullClue:   "97:47:11:3c:af:56:47:b3:f9:a9:89:36:6d:ca:be:0b:33:a0:05:f7",
			wantScheme: domain.SchemeAWSCreatedRSA,
		},
		{
			name:       "aws-imported-rsa vector fragment",
			fullClue:   "a8:e7:45:95:5f:a3:f0:b1:79:6c:c2:f1:d2:80:57:ea",
			wantScheme: domain.SchemeAWSImportedRSA,
		},
		{
			name:       "legacy-ssh-md5 vector fragment",
			fullClue:   "34:29:f4:da:3c:db:49:4b:35:ba:c1:c2:cd:2e:75:a8",
			wantScheme: domain.SchemeLegacySSHMD5,
		},
		{
			name:       "ssh-native-sha256 vector fragment",
			fullClue:   "SHA256:lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA",
			wantScheme: domain.SchemeSSHNativeSHA256,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			normalizedFull := fpscheme.Normalize(tc.fullClue)
			// A prefix short enough to miss every full shape (40, 32, and 43 characters
			// respectively) for every vector in this table.
			fragment := normalizedFull[:8]
			got := fpscheme.CandidateSchemes(fragment)

			found := false
			for _, id := range got {
				if id == tc.wantScheme {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("CandidateSchemes(%q) = %v, must contain %s (the scheme %q's full form belongs to)",
					fragment, got, tc.wantScheme, tc.fullClue)
			}
		})
	}
}
