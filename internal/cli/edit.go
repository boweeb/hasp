package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/domain"
)

// newEditCmd is the `edit` parent command (tdd.md §9's `edit` grid row) — new as of this phase;
// Phase 3a/3b only added `adopt`/`release`. `edit key` is wired here; `edit host`/`edit profile`
// are later M2/M3 slices.
func newEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Change an existing resource in place: rename, alias, replace material, or relocate",
	}
	cmd.AddCommand(newEditKeyCmd())
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
