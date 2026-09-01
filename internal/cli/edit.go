package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/domain"
)

// newEditCmd is the `edit` parent command (tdd.md §9's `edit` grid row). `edit key` and `edit
// host` are wired as of this phase (M3's third slice); `edit profile` is not — tdd.md §9's grid
// cell for `edit`/`profile` is rename-only (D13), not yet built.
func newEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Change an existing resource in place: rename, alias, replace material, or relocate",
	}
	cmd.AddCommand(newEditKeyCmd())
	cmd.AddCommand(newEditHostCmd())
	return cmd
}

// newEditKeyCmd wires `edit key <name-or-clue> [--name=...] [--profile=...] [--add-alias=...]
// [--remove-alias=...] [--replace-material=...]` (tdd.md §9's `edit` grid cell for `key`),
// mirroring adopt.go/release.go's own structure: resolve the target key via resolveKeyClue, build
// a Request, call the UseCase's Plan, then hand off to the shared runWritePlan write loop.
func newEditKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key <name-or-clue>",
		Short: "Rename, alias, replace material, or move a key between profiles",
		Args:  exactArgs(1),
	}
	nameFlag := cmd.Flags().String("name", "", "rename the key's real file in place (new basename)")
	profileFlag := cmd.Flags().String("profile", "", "move the key's real file into a different, already-existing profile directory (dotted path)")
	addAliasFlag := cmd.Flags().String("add-alias", "", "create an alias for the key at <profile>/<name> (bare <name> for a top-level alias) — D2's write path (T19)")
	removeAliasFlag := cmd.Flags().String("remove-alias", "", "remove an existing alias at <profile>/<name> (bare <name> for a top-level alias); never removes the real key file")
	replaceMaterialFlag := cmd.Flags().String("replace-material", "", "replace the key's private key bytes with the contents of <path> (T22)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		m, flags, err := buildMachine(cmd)
		if err != nil {
			return err
		}

		key, err := resolveKeyClue(m, args[0])
		if err != nil {
			return err
		}

		req := app.EditKeyRequest{KeyDir: flags.KeyDir, Key: key, NewName: *nameFlag}
		if *profileFlag != "" {
			req.NewProfile = domain.ParseProfilePath(*profileFlag)
		}
		if *addAliasFlag != "" {
			profile, leaf, perr := parseAliasArg(*addAliasFlag)
			if perr != nil {
				return perr
			}
			req.AddAliasProfile, req.AddAliasLeaf = profile, leaf
		}
		if *removeAliasFlag != "" {
			profile, leaf, perr := parseAliasArg(*removeAliasFlag)
			if perr != nil {
				return perr
			}
			req.RemoveAliasProfile, req.RemoveAliasLeaf = profile, leaf
		}
		req.ReplaceMaterialPath = *replaceMaterialFlag

		plan, err := (app.EditKeyUseCase{}).Plan(req)
		if err != nil {
			return err
		}

		_, applied, err := runWritePlan(cmd, flags, plan)
		if err != nil {
			return err
		}
		if !applied || flags.JSON {
			// --json's stdout is data only (tdd.md §10, T14), mirroring adopt/release's own RunE.
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "edited key %q\n", key.Name)
		return nil
	}
	return cmd
}

// newEditHostCmd wires `edit host <pattern> [--hostname=<h>] [--remove-hostname] [--user=<u>]
// [--remove-user] [--port=<p>] [--remove-port] [--key=<name-or-clue>] [--unbind] [--group=<g>]`
// (tdd.md §9's `edit` grid cell for `host`). Unlike edit key's four mutually exclusive
// sub-operations, edit host's three capabilities (directive changes, rebind, group move) may all
// be requested together in one call — see app.EditHostRequest's own doc comment.
func newEditHostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host <pattern>",
		Short: "Change directives, move between host groups, or rebind a managed Host stanza",
		Args:  exactArgs(1),
	}
	hostNameFlag := cmd.Flags().String("hostname", "", "set the HostName directive's value")
	removeHostNameFlag := cmd.Flags().Bool("remove-hostname", false, "remove the HostName directive if present")
	userFlag := cmd.Flags().String("user", "", "set the User directive's value")
	removeUserFlag := cmd.Flags().Bool("remove-user", false, "remove the User directive if present")
	portFlag := cmd.Flags().String("port", "", "set the Port directive's value")
	removePortFlag := cmd.Flags().Bool("remove-port", false, "remove the Port directive if present")
	keyFlag := cmd.Flags().String("key", "", "rebind: point IdentityFile at this key (name or clue)")
	unbindFlag := cmd.Flags().Bool("unbind", false, "rebind: remove the IdentityFile directive if present")
	groupFlag := cmd.Flags().String("group", "", "move the stanza to this host group (an explicitly passed empty string moves it to the default group, ~/.ssh/config itself)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *hostNameFlag != "" && *removeHostNameFlag {
			return fmt.Errorf("%w: --hostname and --remove-hostname are mutually exclusive", app.ErrUsage)
		}
		if *userFlag != "" && *removeUserFlag {
			return fmt.Errorf("%w: --user and --remove-user are mutually exclusive", app.ErrUsage)
		}
		if *portFlag != "" && *removePortFlag {
			return fmt.Errorf("%w: --port and --remove-port are mutually exclusive", app.ErrUsage)
		}
		if *keyFlag != "" && *unbindFlag {
			return fmt.Errorf("%w: --key and --unbind are mutually exclusive", app.ErrUsage)
		}

		m, flags, err := buildMachine(cmd)
		if err != nil {
			return err
		}

		req := app.EditHostRequest{KeyDir: flags.KeyDir, Pattern: args[0]}

		switch {
		case *removeHostNameFlag:
			empty := ""
			req.HostName = &empty
		case *hostNameFlag != "":
			v := *hostNameFlag
			req.HostName = &v
		}
		switch {
		case *removeUserFlag:
			empty := ""
			req.User = &empty
		case *userFlag != "":
			v := *userFlag
			req.User = &v
		}
		switch {
		case *removePortFlag:
			empty := ""
			req.Port = &empty
		case *portFlag != "":
			v := *portFlag
			req.Port = &v
		}

		if *keyFlag != "" {
			key, err := resolveKeyClue(m, *keyFlag)
			if err != nil {
				return err
			}
			location, err := singleRealLocation(key)
			if err != nil {
				return err
			}
			req.Key = location
		}
		req.Unbind = *unbindFlag

		// cobra's string flag defaults to "" whether or not --group was actually passed, which is
		// indistinguishable from "explicitly requested the default group" — Changed() is the only
		// way to recover "--group was not passed at all" here, mirroring app.EditHostRequest's own
		// NewGroup pointer convention (nil vs. a pointer to "").
		if cmd.Flags().Changed("group") {
			g := *groupFlag
			req.NewGroup = &g
		}

		plan, err := (app.EditHostUseCase{}).Plan(req)
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
		fmt.Fprintf(cmd.OutOrStdout(), "edited host %q\n", args[0])
		return nil
	}
	return cmd
}

// parseAliasArg splits "<profile>/<name>" into its profile path and leaf name — the shape T19's
// `--add-alias=personal/id_ed25519_foobarco` worked example uses (a dot-separated profile, then a
// slash, then the leaf name). `--remove-alias` deliberately reuses the identical shape (see
// EditKeyRequest's own doc comment in internal/app/editkey.go for why): a bare "<name>" (no slash)
// names a top-level alias, with a nil profile.
func parseAliasArg(arg string) (domain.ProfilePath, string, error) {
	idx := strings.LastIndex(arg, "/")
	if idx < 0 {
		return nil, arg, nil
	}
	profilePart, leaf := arg[:idx], arg[idx+1:]
	if profilePart == "" || leaf == "" {
		return nil, "", fmt.Errorf("%w: alias argument %q must be <profile>/<name> or a bare <name>", app.ErrUsage, arg)
	}
	return domain.ParseProfilePath(profilePart), leaf, nil
}
