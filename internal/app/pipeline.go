package app

import (
	"path/filepath"
	"sort"

	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/adapter/scan"
	"github.com/boweeb/hasp/internal/domain"
)

// Machine is the single derived read-model snapshot every read use case (list/show/find/check)
// is built on — computed once per invocation, covering keys/hosts/profiles/bindings together
// (T13's single read aggregate, required by P7's one-screen answer and licensed by P8's scale).
type Machine struct {
	Keys               []domain.Key
	Profiles           []domain.Profile
	Hosts              []domain.Host
	BindingDiagnostics []BindingDiagnostic
}

// DeriveOptions carries every input the derivation pipeline needs. The key directory is always
// injected explicitly here — never defaulted inside app or domain (tdd.md §12); cmd/hasp resolves
// --key-dir's default exactly once, in the cli layer.
type DeriveOptions struct {
	KeyDir string
	// ConfigRoot is the host-group root to start Include resolution from — typically
	// KeyDir/config (~/.ssh/config). Defaults to that when empty.
	ConfigRoot string
}

// Derive runs the scan -> classify -> resolve -> project pipeline (tdd.md §5) once, producing
// the Machine every read use case reads from.
func Derive(opts DeriveOptions) (Machine, error) {
	keys, err := deriveKeys(opts.KeyDir)
	if err != nil {
		return Machine{}, err
	}
	profiles, err := deriveProfiles(opts.KeyDir)
	if err != nil {
		return Machine{}, err
	}
	keys = attachKeyProfiles(keys, profiles)

	configRoot := opts.ConfigRoot
	if configRoot == "" {
		configRoot = filepath.Join(opts.KeyDir, "config")
	}
	configFiles, err := scan.LoadConfigTree(configRoot)
	if err != nil {
		return Machine{}, err
	}

	hosts, diagnostics := ResolveHosts(configFiles, opts.KeyDir, keys)
	hosts = attachHostProfiles(hosts, keys)

	return Machine{
		Keys:               keys,
		Profiles:           profiles,
		Hosts:              hosts,
		BindingDiagnostics: diagnostics,
	}, nil
}

// physicalFile groups every candidate path that resolves to the same on-disk file — a real file
// plus zero or more alias symlinks pointing at it (D5).
type physicalFile struct {
	resolvedPath string
	locations    []domain.KeyLocation
}

// inspectedFile is a physicalFile that keyfile.Inspect successfully classified as a key.
type inspectedFile struct {
	info      keyfile.Info
	locations []domain.KeyLocation
	resolved  string
}

// deriveKeys implements the key side of the derivation pipeline: scan for candidates, classify
// each with keyfile.Inspect (fail-open on anything that isn't actually a key, §11), then group by
// KeyIdentity — fingerprint when derivable, else the resolved path (T12) — so that two files
// sharing a fingerprint become one Key with multiple Locations, while two undecidable files never
// merge, because hasp cannot prove they are the same secret.
func deriveKeys(keyDir string) ([]domain.Key, error) {
	candidates, err := scan.Keys(keyDir)
	if err != nil {
		return nil, err
	}

	byResolved := map[string]*physicalFile{}
	var resolvedOrder []string
	for _, c := range candidates {
		pf, ok := byResolved[c.ResolvedPath]
		if !ok {
			pf = &physicalFile{resolvedPath: c.ResolvedPath}
			byResolved[c.ResolvedPath] = pf
			resolvedOrder = append(resolvedOrder, c.ResolvedPath)
		}
		pf.locations = append(pf.locations, domain.KeyLocation{Path: c.Path, IsAlias: c.IsSymlink})
	}

	var files []inspectedFile
	for _, resolved := range resolvedOrder {
		pf := byResolved[resolved]
		info, err := keyfile.Inspect(resolved)
		if err != nil {
			continue // not actually a key; fail-open (§11) — a survey never aborts over one bad file
		}
		files = append(files, inspectedFile{info: info, locations: pf.locations, resolved: resolved})
	}

	type identityGroup struct {
		identity domain.KeyIdentity
		files    []inspectedFile
	}
	groups := map[string]*identityGroup{}
	var groupOrder []string
	for _, f := range files {
		var identity domain.KeyIdentity
		var groupKey string
		if f.info.Fingerprint != "" {
			identity = domain.NewFingerprintIdentity(f.info.Fingerprint)
			groupKey = "fp:" + string(f.info.Fingerprint)
		} else {
			identity = domain.NewPathIdentity(f.resolved)
			groupKey = "path:" + f.resolved // unique per file; never collides, so never merges
		}
		g, ok := groups[groupKey]
		if !ok {
			g = &identityGroup{identity: identity}
			groups[groupKey] = g
			groupOrder = append(groupOrder, groupKey)
		}
		g.files = append(g.files, f)
	}

	keys := make([]domain.Key, 0, len(groupOrder))
	for _, groupKey := range groupOrder {
		g := groups[groupKey]
		keys = append(keys, projectKey(g.identity, g.files))
	}
	return keys, nil
}

// locationRef pairs one KeyLocation with the inspectedFile it came from, so projectKey can pick
// a single "primary" file for the key's Name and scalar facts (algorithm, comment, ...) once
// every location is sorted.
type locationRef struct {
	loc  domain.KeyLocation
	file *inspectedFile
}

// projectKey turns one identity group (one or more physical files sharing a KeyIdentity) into a
// domain.Key. Locations sort non-alias (real file) first, then lexicographically — the first
// entry after sorting is the "primary" file: its path's basename becomes Name (design.md §5.1:
// a key's handle is "by default the filename it was discovered under," which means a real file,
// not an alias) and its Info supplies every scalar fact. For a confirmed duplicate (two real
// files sharing a fingerprint, T12) this is a deterministic tie-break, not a claim that the
// non-chosen copy's own comment or public-half state is somehow less true — domain.Key simply
// has one scalar slot per fact, and check's duplicate-key-confirmed finding (Stage 14) is what
// surfaces the fact that there even was a choice to make.
func projectKey(identity domain.KeyIdentity, files []inspectedFile) domain.Key {
	var refs []locationRef
	for i := range files {
		for _, loc := range files[i].locations {
			refs = append(refs, locationRef{loc: loc, file: &files[i]})
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].loc.IsAlias != refs[j].loc.IsAlias {
			return !refs[i].loc.IsAlias
		}
		return refs[i].loc.Path < refs[j].loc.Path
	})

	locations := make([]domain.KeyLocation, len(refs))
	for i, r := range refs {
		locations[i] = r.loc
	}

	primary := refs[0].file
	name := domain.KeyName(filepath.Base(refs[0].loc.Path))

	return domain.Key{
		Identity:      identity,
		Locations:     locations,
		Name:          name,
		Algorithm:     primary.info.Algorithm,
		Bits:          primary.info.Bits,
		Comment:       primary.info.Comment,
		Format:        primary.info.Format,
		Encrypted:     primary.info.Encrypted,
		HasPublicHalf: primary.info.HasPublicHalf,
	}
}
