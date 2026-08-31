package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/domain"
)

// newAdoptCmd is the `adopt` parent command (tdd.md §9's `adopt` grid row), mirroring how
// list/show/find/new are structured as parent+noun elsewhere in this package. Only `adopt key` is
// wired in this phase; `adopt host` and `adopt profile` are later M2/M3 slices.
func newAdoptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adopt",
		Short: "Move an unmanaged resource into managed territory",
	}
	cmd.AddCommand(newAdoptKeyCmd())
	return cmd
}

func newAdoptKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key <name-or-clue>",
		Short: "Move an unmanaged key into a managed profile directory, leaving a top-level alias (D13)",
		Args:  exactArgs(1),
	}
	profileFlag := cmd.Flags().String("profile", "", "the managed profile to adopt the key into (dotted path, e.g. work.foobarco)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		m, flags, err := buildMachine(cmd)
		if err != nil {
			return err
		}
		if *profileFlag == "" {
			return fmt.Errorf("%w: --profile is required", app.ErrUsage)
		}

		key, err := resolveKeyClue(m, args[0])
		if err != nil {
			return err
		}
		source, err := singleRealLocation(key)
		if err != nil {
			return err
		}

		plan, err := (app.AdoptKeyUseCase{}).Plan(app.AdoptKeyRequest{
			KeyDir:  flags.KeyDir,
			Profile: domain.ParseProfilePath(*profileFlag),
			Source:  source,
		})
		if err != nil {
			return err
		}

		_, applied, err := runWritePlan(cmd, flags, plan)
		if err != nil {
			return err
		}
		if !applied || flags.JSON {
			// --json's stdout is data only (tdd.md §10, T14): the "plan.preview" envelope
			// runWritePlan already rendered is the entire machine-readable answer.
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "adopted key %q into profile %q\n", key.Name, *profileFlag)
		return nil
	}
	return cmd
}
