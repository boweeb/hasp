package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/boweeb/hasp/internal/domain"
)

// checkHosts runs every host-scoped detector.
func checkHosts(hosts []domain.Host, diagnostics []BindingDiagnostic, defects []MarkerDefectInfo) []Finding {
	var findings []Finding
	findings = append(findings, detectBindingDiagnostics(diagnostics)...)
	findings = append(findings, detectHostNoBinding(hosts)...)
	findings = append(findings, detectShadowedStanza(hosts)...)
	findings = append(findings, detectMarkerDefect(defects)...)
	findings = append(findings, detectStanzaInMultipleGroups(hosts)...)
	findings = append(findings, detectStanzaInNoGroup(hosts)...)
	return findings
}

func hostName(patterns []string) string { return strings.Join(patterns, " ") }

// detectBindingDiagnostics turns ResolveHosts's diagnostics (Stage 13) directly into findings —
// one BindingDiagnosticKind per finding id.
func detectBindingDiagnostics(diagnostics []BindingDiagnostic) []Finding {
	var findings []Finding
	for _, d := range diagnostics {
		name := hostName(d.HostPatterns)
		switch d.Kind {
		case DiagDanglingTarget:
			findings = append(findings, newFinding(
				FindingDanglingIdentityFile,
				Subject{Kind: "host", Name: name},
				fmt.Sprintf("IdentityFile %q does not resolve to a key hasp recognizes", d.RawValue),
				map[string]any{"value": d.RawValue, "resolved": d.ResolvedPath},
			))
		case DiagUnresolvableToken:
			findings = append(findings, newFinding(
				FindingUnresolvableToken,
				Subject{Kind: "host", Name: name},
				fmt.Sprintf("IdentityFile token %s is not one hasp resolves", d.RawValue),
				map[string]any{"token": d.RawValue},
			))
		case DiagRelativeIdentityFile:
			findings = append(findings, newFinding(
				FindingRelativeIdentityFile,
				Subject{Kind: "host", Name: name},
				fmt.Sprintf("IdentityFile %q is a bare relative path; hasp resolved it against the key directory, which ssh may resolve differently at connection time", d.RawValue),
				map[string]any{"value": d.RawValue, "resolved": d.ResolvedPath},
			))
		}
	}
	return findings
}

// detectHostNoBinding fires only when neither an explicit nor a resolvable implicit-default
// binding exists (T16) — Host.Bindings already carries both kinds, so an empty slice here means
// genuinely neither resolved.
func detectHostNoBinding(hosts []domain.Host) []Finding {
	var findings []Finding
	for _, h := range hosts {
		if len(h.Bindings) > 0 {
			continue
		}
		findings = append(findings, newFinding(
			FindingHostNoBinding,
			Subject{Kind: "host", Name: hostName(h.Patterns)},
			fmt.Sprintf("host %q has no explicit or implicit-default key binding", hostName(h.Patterns)),
			nil,
		))
	}
	return findings
}

// detectShadowedStanza fires on first-obtained-value-wins shadowing (T11): a later stanza
// declaring the exact same Host pattern set as an earlier one, in the order Hosts were derived
// (Include order — the same order that order *is* precedence, per scan.LoadConfigTree, Stage
// 10). v1 scope, stated plainly: exact pattern-set duplication only, not wildcard subsumption
// (e.g. "Host *.example.com" shadowing "Host db.example.com") — a substantially harder general
// matching problem, deferred.
func detectShadowedStanza(hosts []domain.Host) []Finding {
	seen := map[string]domain.Host{}
	var findings []Finding
	for _, h := range hosts {
		key := patternSetKey(h.Patterns)
		first, ok := seen[key]
		if !ok {
			seen[key] = h
			continue
		}
		findings = append(findings, newFinding(
			FindingShadowedStanza,
			Subject{Kind: "host", Name: hostName(h.Patterns)},
			fmt.Sprintf("stanza %q in %s is shadowed by an earlier declaration in %s (first-obtained-value-wins)", hostName(h.Patterns), h.HostGroup, first.HostGroup),
			map[string]any{"shadowedBy": first.HostGroup},
		))
	}
	return findings
}

func patternSetKey(patterns []string) string {
	sorted := append([]string(nil), patterns...)
	sort.Strings(sorted)
	return strings.Join(sorted, "\x00")
}

func detectMarkerDefect(defects []MarkerDefectInfo) []Finding {
	var findings []Finding
	for _, d := range defects {
		findings = append(findings, newFinding(
			FindingMarkerDefect,
			Subject{Kind: "host-group", Name: d.HostGroup},
			fmt.Sprintf("marker defect (%s) at line %d of %s", d.Kind, d.Line, d.HostGroup),
			map[string]any{"kind": d.Kind.String(), "line": d.Line, "detail": d.Detail},
		))
	}
	return findings
}

// detectStanzaInMultipleGroups and detectStanzaInNoGroup are reserved, always-empty detectors in
// M1. Both ids stay in the golden v1 set from the start (T29: "the finding schema fixed from the
// start") because they are named findings in that same table, but their real trigger — a stanza
// left in two host-group files, or in none, by a multi-file Plan that applied only partially — is
// a write-path artifact (T13) that does not exist before M2/M3's Plan/Applier. Real detection
// needs the metadata channel (T25) to track stanza identity across files; that channel's keyset
// is empty until then (tdd.md §7).
func detectStanzaInMultipleGroups(_ []domain.Host) []Finding { return nil }
func detectStanzaInNoGroup(_ []domain.Host) []Finding        { return nil }
