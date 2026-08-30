package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/cli/render"
	"github.com/boweeb/hasp/internal/domain"
)

func newListHostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host",
		Short: "List every host stanza, scoped to one group with --group",
	}
	group := cmd.Flags().String("group", "", "scope to one host group's short name (e.g. \"work\" for work.sshconfig)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		m, flags, err := buildMachine(cmd)
		if err != nil {
			return err
		}
		hosts := app.ListHosts(m, *group)
		if flags.JSON {
			return render.JSON(cmd.OutOrStdout(), "host.list", hosts, nil)
		}
		return renderHostTable(cmd, hosts)
	}
	return cmd
}

func newShowHostCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "host <pattern>",
		Short: "Show full stanza detail: resolved bindings, host group, derived profiles",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			h, ok := app.ShowHost(m, args[0])
			if !ok {
				return fmt.Errorf("host %q not found", args[0])
			}
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "host.show", h, nil)
			}
			return renderHostDetail(cmd, h)
		},
	}
}

func newFindHostCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "host <clue>",
		Short: "Match by pattern fragment, or by the name/fingerprint of a bound key",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			hosts := app.FindHosts(m, args[0])
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "host.find", hosts, nil)
			}
			return renderHostTable(cmd, hosts)
		},
	}
}

func newCheckHostCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "host",
		Short: "Report dangling, unresolvable, shadowed, or unbound host stanzas",
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			findings := findingsForSubjectKinds(app.Check(m), "host", "host-group")
			if err := renderFindings(cmd, flags, findings); err != nil {
				return err
			}
			if len(findings) > 0 {
				return app.ErrFindings
			}
			return nil
		},
	}
}

func findingsForSubjectKinds(findings []app.Finding, kinds ...string) []app.Finding {
	want := map[string]bool{}
	for _, k := range kinds {
		want[k] = true
	}
	var out []app.Finding
	for _, f := range findings {
		if want[f.Subject.Kind] {
			out = append(out, f)
		}
	}
	return out
}

// renderHostTable, like renderKeyTable, puts the fixed-width columns first and the one
// variable-width column (bindings) last, and keeps that column short (P7) — a host bound to
// several implicit-default keys at once must not turn one row into a wall of fingerprints.
func renderHostTable(cmd *cobra.Command, hosts []domain.Host) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	fmt.Fprintln(w, "PATTERN\tGROUP\tPROFILES\tMANAGED\tBINDINGS")
	for _, h := range hosts {
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n",
			hostPatternName(h.Patterns), hostGroupName(h.HostGroup), profilesCell(h.Profiles), h.Managed, bindingsCell(h.Bindings))
	}
	return w.Flush()
}

func renderHostDetail(cmd *cobra.Command, h domain.Host) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	fmt.Fprintf(w, "Pattern:\t%s\n", hostPatternName(h.Patterns))
	fmt.Fprintf(w, "Group:\t%s\n", hostGroupName(h.HostGroup))
	fmt.Fprintf(w, "Managed:\t%v\n", h.Managed)
	fmt.Fprintf(w, "Profiles:\t%s\n", profilesCell(h.Profiles))
	fmt.Fprintln(w, "Bindings:")
	if len(h.Bindings) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, b := range h.Bindings {
		fmt.Fprintf(w, "  %s\t(%s)\n", b.Key.Value(), b.Kind)
	}
	return w.Flush()
}

// bindingsCell summarizes a host's bindings for the list/find table: at most 2 shown by name
// with a truncated identity, collapsing to a count beyond that (T16's implicit-default probing
// alone can add up to 6 bindings to a single stanza) — full detail is `show host`'s job, not
// list's (P7: the default is the useful answer, not a data dump).
func bindingsCell(bindings []domain.Binding) string {
	if len(bindings) == 0 {
		return "(none)"
	}
	const maxShown = 2
	if len(bindings) > maxShown {
		return fmt.Sprintf("%d bindings (see show host)", len(bindings))
	}
	s := ""
	for i, b := range bindings {
		if i > 0 {
			s += ", "
		}
		s += truncatedIdentityValue(b.Key) + " (" + b.Kind.String() + ")"
	}
	return s
}
