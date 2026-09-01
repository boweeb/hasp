package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/domain"
)

// newReleaseCmd is the `release` parent command (tdd.md §9's `release` grid row), the inverse of
// `adopt` (D14). `release key` and `release profile` are wired as of this phase; `release host` is
// a later M2/M3 slice.
func newReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Move a managed resource back out of managed territory (inverse of adopt)",
	}
	cmd.AddCommand(newReleaseKeyCmd())
	cmd.AddCommand(newReleaseProfileCmd())
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

// newReleaseProfileCmd wires `release profile <name>`, the inverse of `adopt profile` (D14, D18).
// Mirrors `adopt profile`'s own RunE: the target directory is built entirely from --key-dir and
// the dotted profile name, no clue resolution against a derived Machine is needed.
func newReleaseProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profile <name>",
		Short: "Remove a profile directory's .hasp marker, showing its contents first (D15, D18)",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := readGlobalFlags(cmd)
			if err != nil {
				return err
			}
			if err := requireKeyDir(flags); err != nil {
				return err
			}

			plan, err := (app.ReleaseProfileUseCase{}).Plan(app.ReleaseProfileRequest{
				KeyDir:  flags.KeyDir,
				Profile: domain.ParseProfilePath(args[0]),
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
			fmt.Fprintf(cmd.OutOrStdout(), "released profile %q\n", args[0])
			return nil
		},
	}
}
