package app

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

func TestShowKey_FoundWithBoundHost(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "id_ed25519")
	copyFixture(t, "ed25519-openssh-plain-nopub", realPath)
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile "+realPath+"\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	detail, ok := ShowKey(m, "id_ed25519")
	if !ok {
		t.Fatal("ShowKey: not found")
	}
	if len(detail.Hosts) != 1 || detail.Hosts[0].Patterns[0] != "work" {
		t.Errorf("Hosts = %+v, want [work]", detail.Hosts)
	}
}

func TestShowKey_NotFound(t *testing.T) {
	dir := t.TempDir()
	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if _, ok := ShowKey(m, "nope"); ok {
		t.Error("ShowKey: want not found")
	}
}

// flipCase inverts the case of every letter in s, leaving digits and punctuation untouched — a
// test helper for proving a case-insensitive comparison actually folds case, rather than merely
// tolerating a clue that never had any (item 6, M3.6.3 review).
func flipCase(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			b.WriteRune(unicode.ToLower(r))
		case unicode.IsLower(r):
			b.WriteRune(unicode.ToUpper(r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestFindKeys_FragmentPunctuationAndCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-pub", filepath.Join(dir, "id_ed25519"))
	copyFixture(t, "ed25519-openssh-plain-pub.pub", filepath.Join(dir, "id_ed25519.pub"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	full := m.Keys[0].Identity.Value() // "SHA256:xxxxx..."
	fragment := full[len(full)-8:]

	flipped := flipCase(fragment)
	if flipped == fragment {
		t.Fatalf("fixture fragment %q has no letters to flip; test proves nothing about case-folding", fragment)
	}

	// Mangle case (flipped, genuinely different from the real fingerprint's case) and inject
	// punctuation into the fragment — J2's requirement is both, not just punctuation (item 6).
	mangled := "  " + flipped[:2] + ":" + flipped[2:]

	got, _ := FindKeys(m, mangled)
	if len(got) != 1 {
		t.Fatalf("FindKeys(%q) = %d results, want 1", mangled, len(got))
	}
}

// TestCompareFold_PreservesPlusAndSlash guards item 3's narrowing (M3.6.3 review, T50): T50's own
// safety argument for folding punctuation away covers only "-" and "_" ("none of the four
// registered schemes ever emits '-' or '_' as real content"), but the SSH-native scheme's Compute
// output is standard, not URL-safe, base64 — meaning "+" and "/" are real content T50 never
// licenses folding. Two clues differing only in "+" vs "/" must therefore compare unequal.
func TestCompareFold_PreservesPlusAndSlash(t *testing.T) {
	a, b := compareFold("abc+def"), compareFold("abc/def")
	if a == b {
		t.Errorf("compareFold(%q) == compareFold(%q) == %q; '+' and '/' are real base64 content and must not fold together (T50)", "abc+def", "abc/def", a)
	}
}

func TestFindKeys_NoMatch(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if got, _ := FindKeys(m, "zzzzznotarealfragment"); len(got) != 0 {
		t.Errorf("got %d results, want 0", len(got))
	}
}

// TestFindKeys_SkipsUndecidableKeys is T48's own case: a fully undecidable key (legacy PEM,
// encrypted, no .pub — no public half and no private key derivable without a passphrase find
// never asks for, T39) never appears in matches, and — because a candidate-set clue with no shape
// hint (a plain letters-only fragment like "anything") admits every registered scheme, including
// aws-created-rsa — this key's missing private key surfaces as an explicit warning rather than a
// silent gap (D20).
func TestFindKeys_SkipsUndecidableKeys(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "id_rsa_old"))
	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	got, warnings := FindKeys(m, "anything")
	if len(got) != 0 {
		t.Errorf("got %d results for an undecidable-only machine, want 0", len(got))
	}
	if len(warnings) == 0 {
		t.Error("want a warning naming aws-created-rsa as unevaluable for this key (T48), got none")
	}
}
