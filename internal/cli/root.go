// Package cli wires the cobra command tree, flags, and the human/JSON renderers (docs/tdd.md
// §9, §10).
package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
)

// globalFlags is every flag in tdd.md §9's table, read back out of a command's flag set once
// cobra has parsed it — persistent flags registered on root are visible on every subcommand's
// own Flags() once the tree is executing.
type globalFlags struct {
	KeyDir  string
	JSON    bool
	Verbose bool
	NoColor bool
	Yes     bool
}

// NewRootCmd builds the "hasp" root command and every global flag. --key-dir's default is
// resolved here, once, at the outermost layer — never inside internal/app or internal/domain
// (tdd.md §12) — via os.UserHomeDir(), the same stdlib call T7 uses for the settings path,
// deliberately avoiding an XDG-paths dependency (T7's non-dependency list).
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "hasp",
		Short:         "hasp manages SSH identity: keys, hosts, and profiles on one laptop.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			flags, err := readGlobalFlags(cmd)
			if err != nil {
				return err
			}
			configureLogging(flags.Verbose)
			return nil
		},
	}

	root.PersistentFlags().String("key-dir", defaultKeyDir(), "key directory (default ~/.ssh)")
	root.PersistentFlags().Bool("json", false, "machine-readable output")
	root.PersistentFlags().Bool("verbose", false, "enable diagnostic logging to stderr")
	root.PersistentFlags().Bool("no-color", false, "disable ANSI color in human output")
	// --yes: consent to apply a Plan non-interactively (tdd.md §9, §4's command-loop step 4). Was
	// a stub through M1, since nothing wrote yet; from M2's `new key` onward it gates every write
	// command's confirm step (internal/cli/write.go's runWritePlan) — no TTY and no --yes fails
	// closed (tdd.md §11).
	root.PersistentFlags().Bool("yes", false, "consent to apply a write non-interactively")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newShowCmd())
	root.AddCommand(newFindCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newNewCmd())

	return root
}

func defaultKeyDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh")
}

// requireKeyDir is called by every noun command's RunE (Stages 18-20) — version/help/completion
// need no key directory at all, so the check lives here rather than in root's
// PersistentPreRunE, which runs unconditionally for every command in the tree.
func requireKeyDir(flags globalFlags) error {
	if flags.KeyDir == "" {
		return fmt.Errorf("%w: --key-dir was not given and no home directory could be detected", app.ErrUsage)
	}
	return nil
}

func readGlobalFlags(cmd *cobra.Command) (globalFlags, error) {
	keyDir, err := cmd.Flags().GetString("key-dir")
	if err != nil {
		return globalFlags{}, err
	}
	jsonOut, err := cmd.Flags().GetBool("json")
	if err != nil {
		return globalFlags{}, err
	}
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return globalFlags{}, err
	}
	noColor, err := cmd.Flags().GetBool("no-color")
	if err != nil {
		return globalFlags{}, err
	}
	yes, err := cmd.Flags().GetBool("yes")
	if err != nil {
		return globalFlags{}, err
	}
	return globalFlags{KeyDir: keyDir, JSON: jsonOut, Verbose: verbose, NoColor: noColor, Yes: yes}, nil
}

// configureLogging wires log/slog to stderr only when --verbose is set (T14): stdout carries
// data and nothing else, which is what makes the pipeline promise (hasp ... --json | jq) literal
// rather than aspirational. Off by default means routed to io.Discard, not merely "unconfigured"
// — slog's own zero-value default handler is not guaranteed silent, so this is explicit either
// way.
func configureLogging(verbose bool) {
	out := io.Discard
	if verbose {
		out = os.Stderr
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(out, nil)))
}

// exactArgs wraps cobra.ExactArgs so a missing/extra positional argument reports as app.ErrUsage
// (exit code 2, T14) rather than falling through to the generic exit code 3.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(n)(cmd, args); err != nil {
			return fmt.Errorf("%w: %v", app.ErrUsage, err)
		}
		return nil
	}
}
