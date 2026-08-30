package app

import (
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"github.com/boweeb/hasp/internal/adapter/scan"
	"github.com/boweeb/hasp/internal/adapter/sshconfig"
	"github.com/boweeb/hasp/internal/domain"
)

// implicitDefaultNames is ssh_config(5)'s own default identity file list, verified against the
// man page on OpenSSH_10.5p1 (T16). id_dsa is deliberately absent — modern OpenSSH dropped it
// from the defaults, though a key at that path is still a legitimate, readable artifact (T1,
// T23).
var implicitDefaultNames = []string{
	"id_rsa", "id_ecdsa", "id_ecdsa_sk", "id_ed25519", "id_ed25519_sk", "id_mldsa44_ed25519",
}

// BindingDiagnosticKind enumerates the ways an IdentityFile value fails to resolve cleanly, or
// resolves in a way worth flagging even though it succeeded (T28, tdd.md §5). check's host
// detectors (Stage 15) turn these into findings.
type BindingDiagnosticKind int

const (
	DiagUnresolvableToken    BindingDiagnosticKind = iota // a %X token hasp does not resolve (only %d %h %r %u are, T28's table)
	DiagRelativeIdentityFile                              // a bare relative path, resolved against the key directory rather than ssh's cwd (T28)
	DiagDanglingTarget                                    // the resolved path does not exist, or is not a key hasp recognizes
)

// BindingDiagnostic names one binding-resolution problem on one Host stanza, carrying enough
// context for check to report it without re-deriving anything.
type BindingDiagnostic struct {
	HostPatterns []string
	HostGroup    string
	Kind         BindingDiagnosticKind
	RawValue     string // the raw IdentityFile value or "%X" token involved
	ResolvedPath string // best-effort resolved path, where applicable
}

// ResolveHosts builds every domain.Host from configFiles' HostBlocks, resolving each stanza's
// bindings — explicit IdentityFile lines and, when a stanza has none at all, ssh's own implicit
// default identity probing (T16) — against keys, the already-derived key list. keys must be
// derived first: a binding is only real once it identifies a key hasp actually found (tdd.md §5).
func ResolveHosts(configFiles []scan.ConfigFile, keyDir string, keys []domain.Key) ([]domain.Host, []BindingDiagnostic) {
	keysByPath := buildKeyPathIndex(keys)

	var hosts []domain.Host
	var diagnostics []BindingDiagnostic

	for _, cf := range configFiles {
		for _, found := range collectHostBlocks(cf.Body.Nodes, false) {
			bindings, diags := resolveBindingsForHost(found.block, keyDir, keysByPath)
			for i := range diags {
				diags[i].HostGroup = cf.Path
			}
			diagnostics = append(diagnostics, diags...)

			hosts = append(hosts, domain.Host{
				Patterns:  found.block.Patterns,
				HostGroup: cf.Path,
				Managed:   found.managed,
				Bindings:  bindings,
			})
		}
	}

	return hosts, diagnostics
}

// foundHostBlock pairs a parsed HostBlock with whether it sits inside a MarkedRegion — Managed,
// per D7/D14.
type foundHostBlock struct {
	block   *sshconfig.HostBlock
	managed bool
}

// collectHostBlocks walks nodes for Host-keyword blocks, recursing into any MarkedRegion.
// Match-keyword blocks are recognized structurally elsewhere (HostBlock grouping, Stage 5, so
// trailing directives are never misattributed) but produce no domain.Host in M1.
func collectHostBlocks(nodes []sshconfig.Node, insideRegion bool) []foundHostBlock {
	var out []foundHostBlock
	for _, n := range nodes {
		switch v := n.(type) {
		case *sshconfig.HostBlock:
			if strings.EqualFold(v.Header.Keyword, "Host") {
				out = append(out, foundHostBlock{block: v, managed: insideRegion})
			}
		case *sshconfig.MarkedRegion:
			out = append(out, collectHostBlocks(v.Body, true)...)
		}
	}
	return out
}

// buildKeyPathIndex maps every canonical (symlink-resolved) location of every derived key to
// that key's identity, so a resolved IdentityFile target can be matched back to a real Key.
func buildKeyPathIndex(keys []domain.Key) map[string]domain.KeyIdentity {
	index := map[string]domain.KeyIdentity{}
	for _, k := range keys {
		for _, loc := range k.Locations {
			canonical, err := filepath.EvalSymlinks(loc.Path)
			if err != nil {
				continue
			}
			index[canonical] = k.Identity
		}
	}
	return index
}

// resolveBindingsForHost resolves one HostBlock's bindings. Real ssh_config semantics: any
// IdentityFile directive at all — even "IdentityFile none", even one that fails to resolve to
// anything — replaces the built-in default identity list; ssh only falls back to defaults when
// no IdentityFile directive is present. "none" therefore needs no special-case suppression logic
// of its own: it is simply an IdentityFile directive whose value yields zero explicit bindings,
// and the presence check below already suppresses defaults for exactly that reason (T16).
func resolveBindingsForHost(hb *sshconfig.HostBlock, keyDir string, keysByPath map[string]domain.KeyIdentity) ([]domain.Binding, []BindingDiagnostic) {
	var rawValues []string
	hasIdentityFileDirective := false
	for _, n := range hb.Directives {
		d, ok := n.(*sshconfig.Directive)
		if !ok || !strings.EqualFold(d.Keyword, "IdentityFile") {
			continue
		}
		hasIdentityFileDirective = true
		rawValues = append(rawValues, d.Args()...)
	}

	var bindings []domain.Binding
	var diagnostics []BindingDiagnostic
	seen := map[string]bool{}

	addBinding := func(identity domain.KeyIdentity, kind domain.BindingKind) {
		key := identityMapKey(identity)
		if seen[key] {
			return
		}
		seen[key] = true
		bindings = append(bindings, domain.Binding{Key: identity, Kind: kind})
	}

	for _, raw := range rawValues {
		if strings.EqualFold(raw, "none") {
			continue // zero explicit bindings from this line, by design
		}

		expanded, tokenDiags := expandTokens(raw, hb)
		diagnostics = append(diagnostics, withHostContext(tokenDiags, hb)...)

		wasRelative := !strings.HasPrefix(expanded, "~") && !filepath.IsAbs(expanded)
		resolved := expandTilde(expanded)
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(keyDir, resolved)
		}

		if wasRelative {
			diagnostics = append(diagnostics, BindingDiagnostic{
				HostPatterns: hb.Patterns, Kind: DiagRelativeIdentityFile, RawValue: raw, ResolvedPath: resolved,
			})
		}

		canonical, err := filepath.EvalSymlinks(resolved)
		if err != nil {
			diagnostics = append(diagnostics, BindingDiagnostic{
				HostPatterns: hb.Patterns, Kind: DiagDanglingTarget, RawValue: raw, ResolvedPath: resolved,
			})
			continue
		}
		identity, ok := keysByPath[canonical]
		if !ok {
			diagnostics = append(diagnostics, BindingDiagnostic{
				HostPatterns: hb.Patterns, Kind: DiagDanglingTarget, RawValue: raw, ResolvedPath: canonical,
			})
			continue
		}
		addBinding(identity, domain.BindingExplicit)
	}

	if !hasIdentityFileDirective {
		for _, name := range implicitDefaultNames {
			canonical, err := filepath.EvalSymlinks(filepath.Join(keyDir, name))
			if err != nil {
				continue // absent; the ordinary case for most of the 6 names, never a diagnostic
			}
			if identity, ok := keysByPath[canonical]; ok {
				addBinding(identity, domain.BindingImplicitDefault)
			}
		}
	}

	return bindings, diagnostics
}

func withHostContext(diags []BindingDiagnostic, hb *sshconfig.HostBlock) []BindingDiagnostic {
	for i := range diags {
		diags[i].HostPatterns = hb.Patterns
	}
	return diags
}

// expandTokens resolves %d, %h, %r, and %u per tdd.md §5's table; every other %X sequence
// (including %%) is left as literal text and reported as DiagUnresolvableToken.
func expandTokens(raw string, hb *sshconfig.HostBlock) (string, []BindingDiagnostic) {
	var diags []BindingDiagnostic
	var out strings.Builder
	runes := []rune(raw)

	for i := 0; i < len(runes); i++ {
		if runes[i] != '%' || i+1 >= len(runes) {
			out.WriteRune(runes[i])
			continue
		}
		tok := runes[i+1]
		i++
		switch tok {
		case 'd':
			if home, err := os.UserHomeDir(); err == nil {
				out.WriteString(home)
				continue
			}
		case 'u':
			if u, err := user.Current(); err == nil {
				out.WriteString(u.Username)
				continue
			}
		case 'r':
			if v, ok := directiveValue(hb, "User"); ok {
				out.WriteString(v)
				continue
			}
			if u, err := user.Current(); err == nil {
				out.WriteString(u.Username)
				continue
			}
		case 'h':
			if v, ok := directiveValue(hb, "HostName"); ok {
				out.WriteString(v)
				continue
			}
			if len(hb.Patterns) == 1 && !strings.ContainsAny(hb.Patterns[0], "*?") {
				out.WriteString(hb.Patterns[0])
				continue
			}
		}
		out.WriteByte('%')
		out.WriteRune(tok)
		diags = append(diags, BindingDiagnostic{Kind: DiagUnresolvableToken, RawValue: "%" + string(tok)})
	}

	return out.String(), diags
}

// directiveValue returns the first argument of the first directive on hb matching keyword.
func directiveValue(hb *sshconfig.HostBlock, keyword string) (string, bool) {
	for _, n := range hb.Directives {
		d, ok := n.(*sshconfig.Directive)
		if !ok || !strings.EqualFold(d.Keyword, keyword) {
			continue
		}
		if args := d.Args(); len(args) > 0 {
			return args[0], true
		}
	}
	return "", false
}

// expandTilde expands a leading "~/" (or bare "~") to the local user's home directory. Real ssh
// also supports "~user/..."; hasp does not (T28 scopes resolution to the four tokens ssh_config
// itself names, and this is the same narrowing applied consistently).
func expandTilde(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

func identityMapKey(id domain.KeyIdentity) string {
	return id.Kind().String() + ":" + id.Value()
}

// attachHostProfiles computes Host.Profiles as the union of the profile sets of every key its
// bindings resolve to (T5) — unmanaged hosts get one too, since the computation only needs key
// location, not host management state (this is what lets M1 report host-profile membership
// before adopt exists at all, strengthening J1 and J3).
func attachHostProfiles(hosts []domain.Host, keys []domain.Key) []domain.Host {
	byIdentity := map[string][]domain.ProfilePath{}
	for _, k := range keys {
		byIdentity[identityMapKey(k.Identity)] = k.Profiles
	}

	out := make([]domain.Host, len(hosts))
	for i, h := range hosts {
		seen := map[string]bool{}
		var union []domain.ProfilePath
		for _, b := range h.Bindings {
			for _, p := range byIdentity[identityMapKey(b.Key)] {
				key := p.String()
				if seen[key] {
					continue
				}
				seen[key] = true
				union = append(union, p)
			}
		}
		sort.Slice(union, func(a, c int) bool { return union[a].String() < union[c].String() })
		h.Profiles = union
		out[i] = h
	}
	return out
}
