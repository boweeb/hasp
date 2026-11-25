// Package cli wires the cobra command tree, flags, and the human/JSON renderers (docs/tdd.md
// §9, §10).
package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

	markHelpOnly(root)
	root.SetUsageTemplate(hideRunnableUsageLine(root.UsageTemplate()))

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
	for _, parent := range []*cobra.Command{
		newListCmd(),
		newShowCmd(),
		newFindCmd(),
		newCheckCmd(),
		newNewCmd(),
		newAdoptCmd(),
		newReleaseCmd(),
		newEditCmd(),
	} {
		if parent.Run == nil && parent.RunE == nil {
			markHelpOnly(parent)
		} else {
			parent.Args = rejectUnexpectedArgs
		}
		root.AddCommand(parent)
	}

	// Both run before preserveUnknownHelpTopic adds cobra's help command, which takes a topic as
	// its positional args and must keep doing so (T55).
	root.SetFlagErrorFunc(flagUsageError)
	rejectArgsWhereUndeclared(root)

	preserveUnknownHelpTopic(root)

	return root
}

// flagUsageError makes every cobra flag parse error — unknown flag, bad flag value, missing flag
// value — a usage error (exit 2, tdd.md §10's "bad flags/arguments") on any command: cobra
// consults the root's FlagErrorFunc when a child sets none. cobra's own message text is kept.
func flagUsageError(_ *cobra.Command, err error) error {
	return fmt.Errorf("%w: %w", app.ErrUsage, err)
}

// rejectArgsWhereUndeclared gives every Runnable command in the tree that declares no Args
// validator rejectUnexpectedArgs, so a leaf that takes no positionals refuses one instead of
// silently ignoring it: cobra skips its legacyArgs check entirely for a command with no
// subcommands. Commands that declare Args (exactArgs, minimumArgs, help-only parents) keep theirs.
func rejectArgsWhereUndeclared(cmd *cobra.Command) {
	if cmd.Runnable() && cmd.Args == nil {
		cmd.Args = rejectUnexpectedArgs
	}
	for _, sub := range cmd.Commands() {
		rejectArgsWhereUndeclared(sub)
	}
}

// preserveUnknownHelpTopic keeps `hasp help bogus` behaving as it did before root had Args.
// cobra's built-in help command detects an unknown topic via root.Find returning an error, but
// Find only calls legacyArgs for a root without Args, so with Args set it returns (root, ["bogus"])
// and the help command would print root help to stdout instead of "Unknown help topic" on stderr.
// The wrapper reproduces cobra's own unknown-topic output (same calls, same stream) and delegates
// every other case unchanged, including `help list bogus`.
func preserveUnknownHelpTopic(root *cobra.Command) {
	root.InitDefaultHelpCmd()
	help, _, err := root.Find([]string{"help"})
	if err != nil || help == root || help.Run == nil {
		panic("cli: cobra's default help command not found")
	}
	orig := help.Run
	help.Run = func(c *cobra.Command, args []string) {
		if found, rest, err := c.Root().Find(args); err == nil && found == c.Root() && len(rest) > 0 {
			c.Printf("Unknown help topic %#q\n", args)
			cobra.CheckErr(c.Root().Usage())
			return
		}
		orig(c, args)
	}
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

// helpOnlyAnnotation marks a command whose only job is to show its own help and to reject unknown
// positional arguments (root and every verb parent without a real RunE). Cobra runs Args
// validation only on Runnable commands, so such a command is given a RunE (showHelp) purely to
// make it validate — see markHelpOnly. The annotation lets the usage template and the doc
// generator treat it as if it were not Runnable, so help and docs/cli stay byte-identical to a
// tree with no such RunE (backlog/decisions/decision-T54.md).
const helpOnlyAnnotation = "hasp.help-only"

// markHelpOnly makes cmd a help-only command: unknown args are a usage error, no args shows help,
// and the annotation is set in the same place as the RunE so the two cannot drift.
func markHelpOnly(cmd *cobra.Command) {
	cmd.Args = rejectUnexpectedArgs
	cmd.RunE = showHelp
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[helpOnlyAnnotation] = "true"
}

// StripHelpOnlyRunners clears Run/RunE on every help-only command in the tree rooted at root so
// cmd.Runnable() reports false for it, as it did before those commands were given showHelp. It is
// for tools/gendocs only: cobra/doc's generators print a "hasp list [flags]" synopsis for any
// Runnable command, which would misdescribe a help-only parent. Never call it on a tree that will
// be executed.
func StripHelpOnlyRunners(root *cobra.Command) {
	if _, ok := root.Annotations[helpOnlyAnnotation]; ok {
		root.Run, root.RunE = nil, nil
	}
	for _, c := range root.Commands() {
		StripHelpOnlyRunners(c)
	}
}

// runnableUsageCond is the exact opening of cobra's default usage template that prints the
// "hasp list [flags]" line; hideRunnableUsageLine narrows it.
const runnableUsageCond = "Usage:{{if .Runnable}}"

// hideRunnableUsageLine returns tmpl with the Usage line suppressed for help-only commands.
// It panics if cobra's template no longer contains the expected text, so a cobra upgrade that
// changes it fails every test rather than silently reintroducing the misleading line.
func hideRunnableUsageLine(tmpl string) string {
	if !strings.Contains(tmpl, runnableUsageCond) {
		panic("cli: cobra usage template no longer contains " + runnableUsageCond)
	}
	return strings.Replace(tmpl, runnableUsageCond,
		"Usage:{{if and .Runnable (not (index .Annotations \""+helpOnlyAnnotation+"\"))}}", 1)
}

// rejectUnexpectedArgs makes a leftover positional argument a usage error (exit 2). On a command
// with subcommands the token is an unknown noun: cobra's own legacyArgs validator only checks the
// root, and cobra.NoArgs omits the "Did you mean" suggestions legacyArgs adds, so they are restored
// here in cobra's format. On a leaf (T55) the token is just an extra argument, and cobra.NoArgs's
// "unknown command" wording would mislead, so the message says so plainly.
func rejectUnexpectedArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && !cmd.HasSubCommands() {
		return fmt.Errorf("%w: unexpected argument %q for %q", app.ErrUsage, args[0], cmd.CommandPath())
	}
	if err := cobra.NoArgs(cmd, args); err != nil {
		return fmt.Errorf("%w: %v%s", app.ErrUsage, err, suggestionsText(cmd, args[0]))
	}
	return nil
}

// suggestionsText mirrors cobra's unexported Command.findSuggestions, including its bump of a
// non-positive SuggestionsMinimumDistance to 2 (SuggestionsFor uses the raw field).
func suggestionsText(cmd *cobra.Command, arg string) string {
	if cmd.DisableSuggestions {
		return ""
	}
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	suggestions := cmd.SuggestionsFor(arg)
	if len(suggestions) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\nDid you mean this?\n")
	for _, s := range suggestions {
		fmt.Fprintf(&sb, "\t%v\n", s)
	}
	return sb.String()
}

func showHelp(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
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

// minimumArgs wraps cobra.MinimumNArgs the same way exactArgs wraps cobra.ExactArgs, for the one
// command in this tree whose positional args are variadic rather than fixed (`new host`'s one-or-
// more Host patterns, T14): without this, a too-few-args refusal falls through to cobra's raw,
// unprefixed error text and the generic exit code 3 instead of app.ErrUsage's exit code 2 and
// `hasp: usage error: ...` shape (M3 close-out review finding 1).
func minimumArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(n)(cmd, args); err != nil {
			return fmt.Errorf("%w: %v", app.ErrUsage, err)
		}
		return nil
	}
}
