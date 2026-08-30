package cli

import (
	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
)

// newCheckCmd is the "check" verb parent — "tell me what's wrong or untidy" (design.md §6.2).
// check is advisory everywhere and the only verb permitted to exit 1 (T14). Run bare, with no
// noun subcommand, it aggregates every finding across all three nouns in one report — the same
// findings a script would get by unioning "check key"/"check host"/"check profile", but in one
// call and one exit code.
func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Report untidiness across managed and unmanaged resources alike",
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			findings := app.Check(m)
			if err := renderFindings(cmd, flags, findings); err != nil {
				return err
			}
			if len(findings) > 0 {
				return app.ErrFindings
			}
			return nil
		},
	}
	cmd.AddCommand(newCheckKeyCmd())
	cmd.AddCommand(newCheckHostCmd())
	cmd.AddCommand(newCheckProfileCmd())
	return cmd
}
