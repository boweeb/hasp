package app

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/sshconfig"
	"github.com/boweeb/hasp/internal/domain"
)

// hostBlockFixture builds a *sshconfig.HostBlock for expandTokens tests, by parsing a small
// "Host <patterns>\n<indented directives>" snippet through the real CST parser rather than
// hand-constructing the struct — the same "test against the real thing" discipline the rest of
// this package's tests already follow.
func hostBlockFixture(t *testing.T, patterns []string, directives string) *sshconfig.HostBlock {
	t.Helper()
	var b strings.Builder
	b.WriteString("Host ")
	b.WriteString(strings.Join(patterns, " "))
	b.WriteString("\n")
	for _, line := range strings.Split(strings.TrimRight(directives, "\n"), "\n") {
		if line == "" {
			continue
		}
		b.WriteString("    ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	f := sshconfig.Parse([]byte(b.String()))
	hb, ok := f.Nodes[0].(*sshconfig.HostBlock)
	if !ok {
		t.Fatalf("fixture did not parse to a HostBlock: %+v", f.Nodes)
	}
	return hb
}

func TestDerive_ExplicitBinding(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile "+filepath.Join(dir, "id_ed25519")+"\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts) != 1 {
		t.Fatalf("got %d hosts, want 1: %+v", len(m.Hosts), m.Hosts)
	}
	h := m.Hosts[0]
	if len(h.Bindings) != 1 || h.Bindings[0].Kind != domain.BindingExplicit {
		t.Fatalf("Bindings = %+v, want one BindingExplicit", h.Bindings)
	}
	if h.Bindings[0].Key.Value() != m.Keys[0].Identity.Value() {
		t.Errorf("bound key identity = %v, want %v", h.Bindings[0].Key, m.Keys[0].Identity)
	}
}

func TestDerive_ImplicitDefaultBinding(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    HostName work.example.com\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts) != 1 {
		t.Fatalf("got %d hosts, want 1", len(m.Hosts))
	}
	h := m.Hosts[0]
	if len(h.Bindings) != 1 || h.Bindings[0].Kind != domain.BindingImplicitDefault {
		t.Fatalf("Bindings = %+v, want one BindingImplicitDefault (id_ed25519 with no IdentityFile line)", h.Bindings)
	}
}

func TestDerive_IdentityFileNoneSuppressesDefaults(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile none\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts[0].Bindings) != 0 {
		t.Errorf("Bindings = %+v, want none — IdentityFile none suppresses even implicit defaults", m.Hosts[0].Bindings)
	}
}

func TestDerive_IdentitiesOnlyDoesNotSuppressDefaults(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentitiesOnly yes\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts[0].Bindings) != 1 || m.Hosts[0].Bindings[0].Kind != domain.BindingImplicitDefault {
		t.Errorf("Bindings = %+v, want one BindingImplicitDefault — IdentitiesOnly must not suppress it (T16)", m.Hosts[0].Bindings)
	}
}

func TestDerive_DanglingIdentityFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile "+filepath.Join(dir, "does_not_exist")+"\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts[0].Bindings) != 0 {
		t.Errorf("Bindings = %+v, want none", m.Hosts[0].Bindings)
	}
	if len(m.BindingDiagnostics) != 1 || m.BindingDiagnostics[0].Kind != DiagDanglingTarget {
		t.Fatalf("BindingDiagnostics = %+v, want one DiagDanglingTarget", m.BindingDiagnostics)
	}
}

func TestDerive_RelativeIdentityFileResolvesAgainstKeyDir(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-nopub", filepath.Join(dir, "id_ed25519"))
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile id_ed25519\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts[0].Bindings) != 1 {
		t.Fatalf("Bindings = %+v, want one resolved binding (against the key directory, T28)", m.Hosts[0].Bindings)
	}

	var found bool
	for _, d := range m.BindingDiagnostics {
		if d.Kind == DiagRelativeIdentityFile {
			found = true
		}
	}
	if !found {
		t.Errorf("BindingDiagnostics = %+v, want a DiagRelativeIdentityFile flagging the divergence from ssh's own cwd-relative resolution (T28)", m.BindingDiagnostics)
	}
}

func TestDerive_UnresolvableToken(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config"), "Host work\n    IdentityFile ~/.ssh/id_%k\n")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	var found bool
	for _, d := range m.BindingDiagnostics {
		if d.Kind == DiagUnresolvableToken && d.RawValue == "%k" {
			found = true
		}
	}
	if !found {
		t.Errorf("BindingDiagnostics = %+v, want a DiagUnresolvableToken for %%k", m.BindingDiagnostics)
	}
}

func TestExpandTokens_AllFour(t *testing.T) {
	home := mustHomeDir(t)
	u := mustUsername(t)

	hb := hostBlockFixture(t, []string{"work"}, "HostName work.example.com\nUser jesse\n")
	tests := []struct {
		token string
		want  string
	}{
		{"%d", home},
		{"%u", u},
		{"%r", "jesse"}, // stanza's own User directive wins
		{"%h", "work.example.com"},
	}
	for _, tt := range tests {
		got, diags := expandTokens(tt.token, hb)
		if got != tt.want {
			t.Errorf("expandTokens(%q) = %q, want %q", tt.token, got, tt.want)
		}
		if len(diags) != 0 {
			t.Errorf("expandTokens(%q) diags = %v, want none", tt.token, diags)
		}
	}
}

func TestExpandTokens_RFallsBackToU(t *testing.T) {
	u := mustUsername(t)
	hb := hostBlockFixture(t, []string{"work"}, "HostName work.example.com\n") // no User directive
	got, _ := expandTokens("%r", hb)
	if got != u {
		t.Errorf("expandTokens(%%r) = %q, want %q (falls back to %%u)", got, u)
	}
}

func TestExpandTokens_HWithWildcardPatternUnresolvable(t *testing.T) {
	hb := hostBlockFixture(t, []string{"*.example.com"}, "") // no HostName, wildcard pattern
	got, diags := expandTokens("%h", hb)
	if got != "%h" {
		t.Errorf("expandTokens(%%h) = %q, want literal %%h (wildcard pattern has no single value)", got)
	}
	if len(diags) != 1 || diags[0].Kind != DiagUnresolvableToken {
		t.Errorf("diags = %v, want one DiagUnresolvableToken", diags)
	}
}

func mustHomeDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available")
	}
	return home
}

func mustUsername(t *testing.T) string {
	t.Helper()
	u, err := user.Current()
	if err != nil {
		t.Skip("no current user available")
	}
	return u.Username
}
