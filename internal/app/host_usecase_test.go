package app

import (
	"path/filepath"
	"testing"
)

func TestListHosts_FilterByGroup(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "work.sshconfig"), "Host work\n    HostName work.example.com\n")
	writeFile(t, filepath.Join(dir, "config"), "Include "+filepath.Join(dir, "work.sshconfig")+"\n\nHost personal\n    HostName home.example.com\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts) != 2 {
		t.Fatalf("got %d hosts, want 2", len(m.Hosts))
	}

	got := ListHosts(m, "work")
	if len(got) != 1 || got[0].Patterns[0] != "work" {
		t.Errorf("ListHosts(work) = %+v, want [work]", got)
	}

	got = ListHosts(m, "config")
	if len(got) != 1 || got[0].Patterns[0] != "personal" {
		t.Errorf("ListHosts(config) = %+v, want [personal]", got)
	}

	if got := ListHosts(m, ""); len(got) != 2 {
		t.Errorf("ListHosts(\"\") = %+v, want all 2", got)
	}
}

func TestShowHost_ExplicitVsImplicitLabelledDistinctly(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    HostName work.example.com\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	h, ok := ShowHost(m, "work")
	if !ok {
		t.Fatal("ShowHost: not found")
	}
	if len(h.Bindings) != 1 {
		t.Fatalf("Bindings = %+v, want 1", h.Bindings)
	}
	if h.Bindings[0].Kind.String() != "implicit-default" {
		t.Errorf("Bindings[0].Kind = %v, want implicit-default", h.Bindings[0].Kind)
	}
}

func TestFindHosts_ByPatternFragment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "Host work-prod\n    HostName prod.example.com\n\nHost personal\n    HostName home.example.com\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	got := FindHosts(m, "prod")
	if len(got) != 1 || got[0].Patterns[0] != "work-prod" {
		t.Errorf("FindHosts(prod) = %+v, want [work-prod]", got)
	}
}

func TestFindHosts_ByBoundKeyName(t *testing.T) {
	dir := t.TempDir()
	realPath := filepath.Join(dir, "id_ed25519_foobarco")
	copyFixture(t, "ed25519-openssh-plain-nopub", realPath)
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile "+realPath+"\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	got := FindHosts(m, "foobarco")
	if len(got) != 1 || got[0].Patterns[0] != "work" {
		t.Errorf("FindHosts(foobarco) = %+v, want [work]", got)
	}
}
