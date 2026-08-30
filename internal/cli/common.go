package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
)

// hostPatternName joins a Host's patterns into the space-separated form it was written as
// ("Host a b c") — the closest thing a Host stanza has to a single display name.
func hostPatternName(patterns []string) string {
	return strings.Join(patterns, " ")
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
