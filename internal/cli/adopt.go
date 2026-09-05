package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/domain"
)

// newAdoptCmd is the `adopt` parent command (tdd.md §9's `adopt` grid row), mirroring how
// list/show/find/new are structured as parent+noun elsewhere in this package. `adopt key`,
// `adopt profile`, and `adopt host` are all wired as of this phase (M3).
func newAdoptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adopt",
		Short: "Move an unmanaged resource into managed territory",
	}
	cmd.AddCommand(newAdoptKeyCmd())
	cmd.AddCommand(newAdoptProfileCmd())
	cmd.AddCommand(newAdoptHostCmd())
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
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "adopted key %q into profile %q\n", key.Name, *profileFlag); err != nil {
			return err
		}
		return nil
	}
	return cmd
}

// newAdoptProfileCmd wires `adopt profile <name>` (tdd.md §9's `adopt` grid cell: "Add a `.hasp`
// marker to an existing directory that already holds keys", D13). Simpler than `adopt key`: no
// clue resolution against a derived Machine is needed, since the target directory is built
// entirely from --key-dir and the dotted profile name, mirroring `new key`'s own RunE (new.go).
func newAdoptProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profile <name>",
		Short: "Mark an existing directory that already holds keys as a managed profile (D13)",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := readGlobalFlags(cmd)
			if err != nil {
				return err
			}
			if err := requireKeyDir(flags); err != nil {
				return err
			}

			plan, err := (app.AdoptProfileUseCase{}).Plan(app.AdoptProfileRequest{
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
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "adopted profile %q\n", args[0]); err != nil {
				return err
			}
			return nil
		},
	}
}

// newAdoptHostCmd wires `adopt host <pattern>` (tdd.md §9's `adopt` grid cell: "Wrap an existing
// hand-written stanza in hasp's markers", D7, D14). Mirrors newAdoptKeyCmd's own RunE shape
// (buildMachine, then Plan, then runWritePlan) even though AdoptHostRequest needs no clue
// resolution against the derived Machine — Pattern is matched exactly against the raw CST inside
// app.AdoptHostUseCase.Plan itself (mirroring ShowHost's own exact-match semantics), not resolved
// here the way resolveKeyClue resolves a key name-or-clue.
func newAdoptHostCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "host <pattern>",
		Short: "Wrap an existing hand-written Host stanza in hasp's markers (D7, D14)",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}

			plan, err := (app.AdoptHostUseCase{}).Plan(app.AdoptHostRequest{
				KeyDir:  flags.KeyDir,
				Pattern: args[0],
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
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "adopted host %q\n", args[0]); err != nil {
				return err
			}
			return nil
		},
	}
}
