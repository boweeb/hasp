package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
)

// newReleaseCmd is the `release` parent command (tdd.md §9's `release` grid row), the inverse of
// `adopt` (D14). Only `release key` is wired in this phase; `release host` and `release profile`
// are later M2/M3 slices.
func newReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Move a managed resource back out of managed territory (inverse of adopt)",
	}
	cmd.AddCommand(newReleaseKeyCmd())
	return cmd
}

func newReleaseKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "key <name-or-clue>",
		Short: "Move a managed key back out of its profile directory (inverse of adopt, D14)",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}

			key, err := resolveKeyClue(m, args[0])
			if err != nil {
				return err
			}
			source, alias, err := adoptedLocations(key)
			if err != nil {
				return err
			}

			plan, err := (app.ReleaseKeyUseCase{}).Plan(app.ReleaseKeyRequest{
				KeyDir: flags.KeyDir,
				Alias:  alias,
				Source: source,
			})
			if err != nil {
				return err
			}

			_, applied, err := runWritePlan(cmd, flags, plan)
			if err != nil {
				return err
			}
			if !applied || flags.JSON {
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "released key %q\n", key.Name)
			return nil
		},
	}
}
