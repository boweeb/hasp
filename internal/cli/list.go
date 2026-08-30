package cli

import "github.com/spf13/cobra"

// newListCmd is the "list" verb parent — "show me everything of this kind" (design.md §6.2).
// Noun subcommands are attached as they land (Stages 18-20).
func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List every resource of one kind",
	}
	cmd.AddCommand(newListKeyCmd())
	cmd.AddCommand(newListHostCmd())
	cmd.AddCommand(newListProfileCmd())
	return cmd
}
