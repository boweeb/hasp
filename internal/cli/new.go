package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/adapter/settings"
	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/domain"
)

// newNewCmd is the `new` parent command (tdd.md §9's `new` grid row), mirroring how
// list/show/find/check are structured as parent+noun elsewhere in this package. Only `new key`
// is wired in this phase; `new host` and `new profile` are later M2 slices.
func newNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new key, host, or profile",
	}
	cmd.AddCommand(newNewKeyCmd())
	return cmd
}

func newNewKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key <name>",
		Short: "Generate a new ed25519 keypair (never overwrites an existing file)",
		Args:  exactArgs(1),
	}
	passphrase := cmd.Flags().Bool("passphrase", false, "prompt interactively for a passphrase (no local echo)")
	noPassphrase := cmd.Flags().Bool("no-passphrase", false, "generate without a passphrase, explicitly")
	passphraseStdin := cmd.Flags().Bool("passphrase-stdin", false, "read the passphrase from one line of stdin")
	profileFlag := cmd.Flags().String("profile", "", "place the key in this profile (dotted path, e.g. work.foobarco)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		flags, err := readGlobalFlags(cmd)
		if err != nil {
			return err
		}
		if err := requireKeyDir(flags); err != nil {
			return err
		}

		pf := passphraseFlags{Passphrase: *passphrase, NoPassphrase: *noPassphrase, PassphraseStdin: *passphraseStdin}
		settingsMode, err := loadNewKeySettingsMode()
		if err != nil {
			return err
		}
		mode, err := resolvePassphraseMode(pf, os.LookupEnv, settingsMode, isStdinTTY())
		if err != nil {
			return err
		}

		secret, err := obtainPassphrase(mode)
		if err != nil {
			return err
		}
		defer zeroBytes(secret)

		plan, err := (app.NewKeyUseCase{}).Plan(app.NewKeyRequest{
			Name:       args[0],
			KeyDir:     flags.KeyDir,
			Profile:    domain.ParseProfilePath(*profileFlag),
			Passphrase: secret,
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
			// runWritePlan already rendered is the entire machine-readable answer, so a plain-text
			// confirmation line never follows it on the same stream.
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "created key %q\n", args[0])
		return nil
	}
	return cmd
}

// loadNewKeySettingsMode reads settings.toml's [new_key] default_passphrase_mode (tdd.md §8) —
// "" if the file is absent or the key is unset, which falls through resolvePassphraseMode's chain
// to its next tier exactly as a missing setting should.
func loadNewKeySettingsMode() (string, error) {
	path, err := settings.Path()
	if err != nil {
		return "", fmt.Errorf("resolve settings path: %w", err)
	}
	s, err := settings.Load(path)
	if err != nil {
		return "", err
	}
	return s.NewKey.DefaultPassphraseMode, nil
}
