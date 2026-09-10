package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/domain"
)

// hostPatternName joins a Host's patterns into the space-separated form it was written as
// ("Host a b c") — the closest thing a Host stanza has to a single display name.
func hostPatternName(patterns []string) string {
	return strings.Join(patterns, " ")
}

// hostGroupName derives a host group's short display name from its file path, mirroring
// app.hostGroupName (unexported there, so this small rendering-only helper is duplicated here
// rather than exported across the layer boundary for one line of logic): "work.sshconfig" ->
// "work" (D9).
func hostGroupName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".sshconfig")
}

// truncatedIdentityValue shortens a KeyIdentity's display value in table cells (P7): a full
// SHA256 fingerprint is ~52 characters, and a table with several such values per row stops being
// a one-screen answer. Full values remain available via `show`/--json.
func truncatedIdentityValue(id domain.KeyIdentity) string {
	if id.Kind() != domain.IdentityFingerprint {
		return id.Value()
	}
	const prefixLen = len("SHA256:") + 12
	v := id.Value()
	if len(v) <= prefixLen {
		return v
	}
	return v[:prefixLen] + "…"
}

// buildMachine reads the global flags off cmd, validates --key-dir is resolvable, and runs the
// full derivation pipeline once (T13) — the shared setup every noun command's RunE needs before
// it can list/show/find/check anything.
func buildMachine(cmd *cobra.Command) (app.Machine, globalFlags, error) {
	flags, err := readGlobalFlags(cmd)
	if err != nil {
		return app.Machine{}, globalFlags{}, err
	}
	if err := requireKeyDir(flags); err != nil {
		return app.Machine{}, globalFlags{}, err
	}
	m, err := app.Derive(app.DeriveOptions{KeyDir: flags.KeyDir})
	if err != nil {
		return app.Machine{}, globalFlags{}, err
	}
	return m, flags, nil
}

// resolveKeyClue finds exactly one key matching name-or-clue: an exact KeyName match first (the
// same lookup app.ShowKey performs), falling back to app.FindKeys' fingerprint-fragment match
// (J2) when no exact name matches. Shared by `adopt key` and `release key`, both of which take a
// name-or-clue exactly as `show key`/`find key` already do — this is internal/app's own
// ShowKey/FindKeys lookups, at the point a CLI argument needs to become a concrete key before an
// AdoptKeyRequest/ReleaseKeyRequest can be built (tdd.md §3's Request-only framing: internal/app
// never resolves a "which key does the user mean" clue itself).
func resolveKeyClue(m app.Machine, clue string) (domain.Key, error) {
	if detail, ok := app.ShowKey(m, clue); ok {
		return detail.Key, nil
	}
	// resolveKeyClue backs adopt (adopt.go:43), release (release.go:37), edit (edit.go:48,162),
	// and new --key (new.go:117) — every one of them a single-key lookup, not `find key`'s own
	// multi-match report. On the case-0 path below, D20's failure mode (a false negative reading
	// as "you don't have this key") reappears one layer in: an encrypted candidate key against a
	// scheme that needs the decrypted private key (aws-created-rsa, T48) yields zero matches with
	// no signal distinguishing "checked, doesn't match" from "never evaluated." T48's warnings
	// carry exactly that signal, so a not-found here folds them into the error rather than
	// discarding them; case 1 and the multi-match default need no such folding since they already
	// have a positive or disambiguating answer.
	matches, warnings := app.FindKeys(m, clue)
	switch len(matches) {
	case 0:
		if len(warnings) > 0 {
			return domain.Key{}, fmt.Errorf("key %q not found (%s)", clue, strings.Join(warnings, "; "))
		}
		return domain.Key{}, fmt.Errorf("key %q not found", clue)
	case 1:
		return matches[0].Key, nil
	default:
		return domain.Key{}, fmt.Errorf("%q matches %d keys; use a more specific clue", clue, len(matches))
	}
}

// singleRealLocation returns k's one non-alias (real, canonical) location, or an error if k has
// zero or more than one — adopt operates on exactly one real file at a time (D5's alias-vs-real
// distinction), and a key already spread across multiple real locations (T12's
// unconfirmed-duplicate shape) is not something adopt knows how to move.
func singleRealLocation(k domain.Key) (string, error) {
	var real []string
	for _, loc := range k.Locations {
		if !loc.IsAlias {
			real = append(real, loc.Path)
		}
	}
	switch len(real) {
	case 1:
		return real[0], nil
	case 0:
		return "", fmt.Errorf("key %q has no real (non-alias) location", k.Name)
	default:
		return "", fmt.Errorf("key %q has %d real locations; not a shape adopt can move", k.Name, len(real))
	}
}

// adoptedLocations returns the (source, alias) pair release needs: exactly one real location and
// exactly one alias location, the shape adopt produces (tdd.md §9's `release` grid cell). Any
// other shape — no alias at all, or more than one of either — is not something release knows how
// to invert cleanly, and is reported as a usage error rather than guessed at.
func adoptedLocations(k domain.Key) (source, alias string, err error) {
	var reals, aliases []string
	for _, loc := range k.Locations {
		if loc.IsAlias {
			aliases = append(aliases, loc.Path)
		} else {
			reals = append(reals, loc.Path)
		}
	}
	if len(reals) != 1 || len(aliases) != 1 {
		return "", "", fmt.Errorf("key %q is not in the simple adopted shape (1 real location + 1 alias); found %d real, %d alias", k.Name, len(reals), len(aliases))
	}
	return reals[0], aliases[0], nil
}
