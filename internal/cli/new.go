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
// list/show/find/check are structured as parent+noun elsewhere in this package. `new key` and
// `new host` are wired as of this phase; `new profile` is a later M3 slice.
func newNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new key, host, or profile",
	}
	cmd.AddCommand(newNewKeyCmd())
	cmd.AddCommand(newNewHostCmd())
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

// newNewHostCmd wires `new host <pattern...>` (tdd.md §9's `new` grid cell, D9/T11's host-group
// semantics, design.md §5.5): creates a new "Host ..." stanza, in ~/.ssh/config's own managed
// region by default, or in a custom host group file (--group) hasp owns wholly.
func newNewHostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <pattern...>",
		Short: "Create a new Host stanza",
		Args:  minimumArgs(1),
	}
	keyFlag := cmd.Flags().String("key", "", "bind this host to a key (name or clue); omit for no explicit binding")
	groupFlag := cmd.Flags().String("group", "", "the host group to create this stanza in (default: ~/.ssh/config itself)")
	hostNameFlag := cmd.Flags().String("hostname", "", "the HostName directive's value")
	userFlag := cmd.Flags().String("user", "", "the User directive's value")
	portFlag := cmd.Flags().String("port", "", "the Port directive's value")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Always derive the full Machine, even when --key is omitted: --key-dir resolution and
		// validation is shared with the clue-resolution path below, and there is no cheaper
		// equivalent for a command that may need either (adopt.go's own `adopt key` does the same).
		m, flags, err := buildMachine(cmd)
		if err != nil {
			return err
		}

		var identityFile string
		if *keyFlag != "" {
			key, err := resolveKeyClue(m, *keyFlag)
			if err != nil {
				return err
			}
			location, err := singleRealLocation(key)
			if err != nil {
				return err
			}
			identityFile = location
		}

		plan, err := (app.NewHostUseCase{}).Plan(app.NewHostRequest{
			KeyDir:       flags.KeyDir,
			Patterns:     args,
			Group:        *groupFlag,
			HostName:     *hostNameFlag,
			User:         *userFlag,
			Port:         *portFlag,
			IdentityFile: identityFile,
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
		fmt.Fprintf(cmd.OutOrStdout(), "created host %q\n", hostPatternName(args))
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
