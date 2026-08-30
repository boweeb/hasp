package cli

import "github.com/spf13/cobra"

// newCheckCmd is the "check" verb parent — "tell me what's wrong or untidy" (design.md §6.2).
// check is advisory everywhere and the only verb permitted to exit 1 (T14). The bare "hasp
// check" aggregate (no noun) is wired in Stage 20 alongside profile's commands.
func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Report untidiness across managed and unmanaged resources alike",
	}
	cmd.AddCommand(newCheckKeyCmd())
	cmd.AddCommand(newCheckHostCmd())
	return cmd
}
