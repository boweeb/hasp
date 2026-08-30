package app

import (
	"path/filepath"
	"testing"
)

func TestDerive_HostProfilesUnionFromBindings(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "work", "id_ed25519")
	copyFixture(t, "ed25519-openssh-plain-nopub", realPath)
	writeFile(t, filepath.Join(dir, "work", ".hasp"), "# marker\n")

	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile "+realPath+"\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts) != 1 {
		t.Fatalf("got %d hosts, want 1", len(m.Hosts))
	}
	profiles := m.Hosts[0].Profiles
	if len(profiles) != 1 || profiles[0].String() != "work" {
		t.Errorf("Host.Profiles = %v, want [work] (T5)", profiles)
	}
}

func TestDerive_HostWithNoBindingHasNoProfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    HostName work.example.com\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts[0].Bindings) != 0 {
		t.Fatalf("Bindings = %+v, want none (no keys exist to bind implicitly)", m.Hosts[0].Bindings)
	}
	if len(m.Hosts[0].Profiles) != 0 {
		t.Errorf("Profiles = %v, want none — a host bound to no key belongs to no profile", m.Hosts[0].Profiles)
	}
}

func TestDerive_UnmanagedHostStillGetsComputedProfile(t *testing.T) {
	// A stanza outside any marker still gets its profile computed — this is what lets M1's
	// read-only survey report host-profile membership before adopt exists at all (T5).
	dir := t.TempDir()
	realPath := filepath.Join(dir, "work", "id_ed25519")
	copyFixture(t, "ed25519-openssh-plain-nopub", realPath)
	writeFile(t, filepath.Join(dir, "work", ".hasp"), "# marker\n")
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile "+realPath+"\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if m.Hosts[0].Managed {
		t.Fatal("expected the stanza to be unmanaged (no marked region in this fixture)")
	}
	if len(m.Hosts[0].Profiles) != 1 || m.Hosts[0].Profiles[0].String() != "work" {
		t.Errorf("Profiles = %v, want [work] even though the host is unmanaged", m.Hosts[0].Profiles)
	}
}

// TestDerive_HostAfterRegionIsUnmanaged documents a structural consequence of ssh_config's own
// grammar, not a hasp limitation: once any Host/Match line has appeared, every later line
// belongs to some Host/Match stanza (there is no way back to "top level" in the format itself),
// so a begin/end MarkedRegion's closing marker can never come after a Host block it is meant to
// contain — the marker would be swallowed into that stanza's own directive list and reported as
// DefectNestedInHostBlock instead (T18, exercised directly in sshconfig's own tests). This is
// exactly why hasp's marked region in the co-owned config only ever wraps Include lines (T11):
// a Host stanza hasp itself manages directly lives in a wholly-owned host-group file instead,
// under D7's *other* marker convention (a single header comment, no begin/end pair) — which is
// unreachable in M1 because nothing before M3 creates such a file. A Host block that follows a
// clean region in the same file is simply outside it, and correctly reports Managed=false.
func TestDerive_HostAfterRegionIsUnmanaged(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"),
		"# >>> hasp:managed >>>\n"+
			"Include "+filepath.Join(dir, "work.sshconfig")+"\n"+
			"# <<< hasp:managed <<<\n"+
			"\n"+
			"Host personal-legacy\n"+
			"    HostName legacy.example.com\n",
	)
	writeFile(t, filepath.Join(dir, "work.sshconfig"), "Host work\n    HostName work.example.com\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts) != 2 {
		t.Fatalf("got %d hosts, want 2 (work, personal-legacy): %+v", len(m.Hosts), m.Hosts)
	}
	for _, h := range m.Hosts {
		if h.Managed {
			t.Errorf("host %v: Managed = true, want false — neither stanza sits inside a begin/end region", h.Patterns)
		}
	}
}
