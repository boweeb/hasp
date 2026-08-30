package cli

import "github.com/spf13/cobra"

// newFindCmd is the "find" verb parent — "which thing matches this clue?" (design.md §6.2).
func newFindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "find",
		Short: "Identify a resource from a partial clue",
	}
	cmd.AddCommand(newFindKeyCmd())
	return cmd
}
