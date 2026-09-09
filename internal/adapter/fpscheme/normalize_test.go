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
			if len(got) != len(tc.want) {
				t.Fatalf("CandidateSchemes(Normalize(%q)) = %v, want %v", tc.clue, got, tc.want)
			}
			wantSet := map[domain.SchemeID]bool{}
			for _, id := range tc.want {
				wantSet[id] = true
			}
			for _, id := range got {
				if !wantSet[id] {
					t.Errorf("unexpected candidate %s", id)
				}
				delete(wantSet, id)
			}
			if len(wantSet) != 0 {
				t.Errorf("missing candidates: %v", wantSet)
			}
		})
	}
}
