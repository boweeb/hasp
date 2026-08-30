package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/boweeb/hasp/internal/domain"
)

func testdataKeysDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "keys")
}

func copyFixture(t *testing.T, fixtureName, destPath string) {
	t.Helper()
	src := filepath.Join(testdataKeysDir(t), fixtureName)
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixtureName, err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destPath, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDeriveKeys_SinglePlainKey(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-pub", filepath.Join(dir, "id_ed25519"))
	copyFixture(t, "ed25519-openssh-plain-pub.pub", filepath.Join(dir, "id_ed25519.pub"))

	keys, err := deriveKeys(dir)
	if err != nil {
		t.Fatalf("deriveKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("got %d keys, want 1: %+v", len(keys), keys)
	}
	k := keys[0]
	if k.Identity.Kind() != domain.IdentityFingerprint {
		t.Errorf("Identity.Kind() = %v, want IdentityFingerprint", k.Identity.Kind())
	}
	if k.Name != "id_ed25519" {
		t.Errorf("Name = %q, want %q", k.Name, "id_ed25519")
	}
	if len(k.Locations) != 1 || k.Locations[0].IsAlias {
		t.Errorf("Locations = %+v, want one non-alias location", k.Locations)
	}
	if !k.HasPublicHalf {
		t.Error("HasPublicHalf = false, want true")
	}
}

func TestDeriveKeys_AliasSymlinkMergesIntoOneKey(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "work", "id_ed25519_foobarco")
	copyFixture(t, "ed25519-openssh-plain-nopub", realPath)

	aliasPath := filepath.Join(dir, "id_ed25519_foobarco")
	if err := os.Symlink(realPath, aliasPath); err != nil {
		t.Fatal(err)
	}

	keys, err := deriveKeys(dir)
	if err != nil {
		t.Fatalf("deriveKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("got %d keys, want 1 (alias + target must merge into one Key): %+v", len(keys), keys)
	}
	k := keys[0]
	if len(k.Locations) != 2 {
		t.Fatalf("got %d locations, want 2 (real file + alias): %+v", len(k.Locations), k.Locations)
	}
	// Non-alias location sorts first; its basename is the Name.
	if k.Locations[0].IsAlias {
		t.Errorf("Locations[0] = %+v, want the non-alias (real file) location first", k.Locations[0])
	}
	if k.Name != "id_ed25519_foobarco" {
		t.Errorf("Name = %q, want %q", k.Name, "id_ed25519_foobarco")
	}

	var aliasCount int
	for _, loc := range k.Locations {
		if loc.IsAlias {
			aliasCount++
		}
	}
	if aliasCount != 1 {
		t.Errorf("got %d alias locations, want 1", aliasCount)
	}
}

func TestDeriveKeys_ConfirmedDuplicateSharesFingerprint(t *testing.T) {
	dir := t.TempDir()
	// Two independent copies of the same key material (not a symlink alias) — same fingerprint,
	// must merge into one Key with two non-alias Locations (T12's "confirmed duplicate").
	copyFixture(t, "rsa-openssh-plain-nopub", filepath.Join(dir, "id_rsa_old"))
	copyFixture(t, "rsa-openssh-plain-nopub", filepath.Join(dir, "legacy", "id_rsa"))

	keys, err := deriveKeys(dir)
	if err != nil {
		t.Fatalf("deriveKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("got %d keys, want 1 (two files sharing a fingerprint are one key, T12): %+v", len(keys), keys)
	}
	k := keys[0]
	if len(k.Locations) != 2 {
		t.Fatalf("got %d locations, want 2: %+v", len(k.Locations), k.Locations)
	}
	for _, loc := range k.Locations {
		if loc.IsAlias {
			t.Errorf("Locations = %+v, want both non-alias (these are independent copies, not symlinks)", k.Locations)
		}
	}
}

func TestDeriveKeys_UndecidablePairNeverDedupes(t *testing.T) {
	dir := t.TempDir()
	// Two independent copies of an undecidable key (legacy PEM, encrypted, no public half) —
	// byte-identical, but must NOT merge: hasp cannot prove they are the same secret (T12, P1).
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "id_rsa_old"))
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "legacy", "id_rsa"))

	keys, err := deriveKeys(dir)
	if err != nil {
		t.Fatalf("deriveKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2 (undecidable keys never dedupe, even with identical bytes): %+v", len(keys), keys)
	}
	for _, k := range keys {
		if k.Identity.Kind() != domain.IdentityPath {
			t.Errorf("Identity.Kind() = %v, want IdentityPath (undecidable)", k.Identity.Kind())
		}
		if len(k.Locations) != 1 {
			t.Errorf("Locations = %+v, want exactly 1 (no merge)", k.Locations)
		}
	}
}

func TestDeriveKeys_NonKeyFilesSkippedFailOpen(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("not a key\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	keys, err := deriveKeys(dir)
	if err != nil {
		t.Fatalf("deriveKeys: %v, want nil error — a non-key file must not abort the survey", err)
	}
	if len(keys) != 1 {
		t.Fatalf("got %d keys, want 1 (README silently skipped): %+v", len(keys), keys)
	}
}

func TestDeriveKeys_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	keys, err := deriveKeys(dir)
	if err != nil {
		t.Fatalf("deriveKeys: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("got %d keys in an empty dir, want 0", len(keys))
	}
}
