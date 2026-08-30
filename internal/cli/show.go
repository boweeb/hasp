package cli

import "github.com/spf13/cobra"

// newShowCmd is the "show" verb parent — "show me this one thing in full" (design.md §6.2).
func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show full detail for one resource",
	}
	cmd.AddCommand(newShowKeyCmd())
	cmd.AddCommand(newShowHostCmd())
	return cmd
}
