// Package cli wires the cobra command tree, flags, and the human/JSON renderers (docs/tdd.md
// §9, §10). M0 wires only enough to prove the skeleton builds and hasp version works; the
// verb×noun grid arrives in M1.
package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd builds the "hasp" root command. Global flags (docs/tdd.md §9's table — --key-dir,
// --json, --yes, --verbose, --no-color) are wired starting M1, alongside the commands that
// consume them.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "hasp",
		Short:         "hasp manages SSH identity: keys, hosts, and profiles on one laptop.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newVersionCmd())

	return root
}
