package cli

import (
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
