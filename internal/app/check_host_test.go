package app

import (
	"path/filepath"
	"testing"
)

func TestCheck_DanglingIdentityFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile "+filepath.Join(dir, "missing")+"\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !hasFinding(Check(m), FindingDanglingIdentityFile) {
		t.Error("want a dangling-identityfile finding")
	}
}

func TestCheck_UnresolvableToken(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile ~/.ssh/id_%k\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !hasFinding(Check(m), FindingUnresolvableToken) {
		t.Error("want an unresolvable-token finding")
	}
}

func TestCheck_RelativeIdentityFile(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile id_ed25519\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !hasFinding(Check(m), FindingRelativeIdentityFile) {
		t.Error("want a relative-identityfile finding")
	}
}

func TestCheck_HostNoBinding(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    HostName work.example.com\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !hasFinding(Check(m), FindingHostNoBinding) {
		t.Error("want a host-no-binding finding (no keys exist at all)")
	}
}

func TestCheck_HostNoBinding_AbsentWhenImplicitDefaultResolves(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    HostName work.example.com\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if hasFinding(Check(m), FindingHostNoBinding) {
		t.Error("want no host-no-binding finding — the implicit default id_ed25519 resolves")
	}
}

func TestCheck_ShadowedStanza(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.sshconfig"), "Host db\n    HostName a.example.com\n")
	writeFile(t, filepath.Join(dir, "b.sshconfig"), "Host db\n    HostName b.example.com\n")
	writeFile(t, filepath.Join(dir, "config"), ""+
		"Include "+filepath.Join(dir, "a.sshconfig")+"\n"+
		"Include "+filepath.Join(dir, "b.sshconfig")+"\n",
	)

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	findings := Check(m)
	if !hasFinding(findings, FindingShadowedStanza) {
		t.Fatalf("want a shadowed-stanza finding: %+v", findings)
	}
	for _, f := range findings {
		if f.ID != FindingShadowedStanza {
			continue
		}
		if f.Subject.Name != "db" {
			t.Errorf("shadowed subject = %q, want %q", f.Subject.Name, "db")
		}
	}
}

func TestCheck_NoShadowForDistinctPatterns(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "Host a\n    HostName a.example.com\n\nHost b\n    HostName b.example.com\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if hasFinding(Check(m), FindingShadowedStanza) {
		t.Error("want no shadowed-stanza finding for two distinct Host patterns")
	}
}

func TestCheck_MarkerDefect(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "# <<< hasp:managed <<<\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !hasFinding(Check(m), FindingMarkerDefect) {
		t.Error("want a marker-defect finding for an end-before-begin marker")
	}
}

func TestCheck_ReservedFindingsNeverFire(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "work", "id_ed25519")
	copyFixture(t, "ed25519-openssh-plain-pub", realPath)
	copyFixture(t, "ed25519-openssh-plain-pub.pub", realPath+".pub")
	writeFile(t, filepath.Join(dir, "work", ".hasp"), "# marker\n")
	writeFile(t, filepath.Join(dir, "config"),
		"# >>> hasp:managed >>>\n"+
			"Include "+filepath.Join(dir, "work.sshconfig")+"\n"+
			"# <<< hasp:managed <<<\n"+
			"\nHost work\n    IdentityFile "+realPath+"\n",
	)
	writeFile(t, filepath.Join(dir, "work.sshconfig"), "Host other\n    HostName other.example.com\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	findings := Check(m)
	if hasFinding(findings, FindingStanzaInMultipleGroups) || hasFinding(findings, FindingStanzaInNoGroup) {
		t.Errorf("reserved findings fired in M1: %+v", findings)
	}
}

func TestAllFindingIDs_GoldenList(t *testing.T) {
	want := []FindingID{
		FindingDanglingIdentityFile,
		FindingDuplicateKeyConfirmed,
		FindingDuplicateKeyUnconfirmed,
		FindingEmptyProfileDir,
		FindingFingerprintUnknown,
		FindingHostNoBinding,
		FindingKeyMissingPublicHalf,
		FindingKeyNoProfile,
		FindingMarkerDefect,
		FindingRelativeIdentityFile,
		FindingShadowedStanza,
		FindingStanzaInMultipleGroups,
		FindingStanzaInNoGroup,
		FindingUnmarkedProfileDir,
		FindingUnresolvableToken,
	}
	got := AllFindingIDs()
	if len(got) != len(want) {
		t.Fatalf("got %d finding ids, want %d (T29's v1 set): got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllFindingIDs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestCheck_EveryFindingHasAKnownID asserts every finding Check can ever produce, across every
// detector's own test fixtures in this file and check_test.go, carries an id from the golden set
// — the same discipline as the settings field-set test T7 describes, applied here first.
func TestCheck_EveryFindingHasAKnownID(t *testing.T) {
	known := map[FindingID]bool{}
	for _, id := range AllFindingIDs() {
		known[id] = true
	}

	dir := t.TempDir()
	copyFixture(t, "rsa-openssh-plain-nopub", filepath.Join(dir, "id_rsa_old"))
	copyFixture(t, "rsa-openssh-plain-nopub", filepath.Join(dir, "legacy", "id_rsa"))
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "old1"))
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "old2"))
	writeFile(t, filepath.Join(dir, "empty-profile", ".hasp"), "# marker\n")
	writeFile(t, filepath.Join(dir, "unmarked", "id_ed25519"), "placeholder")
	writeFile(t, filepath.Join(dir, "config"),
		"# <<< hasp:managed <<<\n"+
			"Host work\n    IdentityFile "+filepath.Join(dir, "missing")+"\n"+
			"\nHost work2\n    IdentityFile ~/.ssh/id_%k\n"+
			"\nHost work3\n    IdentityFile id_rsa_old\n",
	)

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	for _, f := range Check(m) {
		if !known[f.ID] {
			t.Errorf("finding with unknown id %q produced", f.ID)
		}
	}
}
