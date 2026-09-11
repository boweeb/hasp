package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/cli/render"
	"github.com/boweeb/hasp/internal/domain"
)

func newListProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "profile",
		Short:   "List every directory carrying a .hasp marker, with key/host counts",
		Aliases: []string{"profiles"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			profiles := app.ListProfiles(m)
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "profile.list", profiles, nil)
			}
			return renderProfileSummaryTable(cmd, profiles)
		},
	}
}

func newShowProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile <name>",
		Short: "Show a profile's keys and hosts, aggregated across child profiles by default",
		Args:  exactArgs(1),
	}
	noRecurse := cmd.Flags().Bool("no-recurse", false, "scope to the named profile alone, without descendants")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		m, flags, err := buildMachine(cmd)
		if err != nil {
			return err
		}
		detail, ok := app.ShowProfile(m, domain.ParseProfilePath(args[0]), !*noRecurse)
		if !ok {
			return fmt.Errorf("profile %q not found (it may be an unmarked container directory, T21)", args[0])
		}
		if flags.JSON {
			return render.JSON(cmd.OutOrStdout(), "profile.show", detail, nil)
		}
		return renderProfileDetail(cmd, detail)
	}
	return cmd
}

func newFindProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "profile <clue>",
		Short:   "Match by name fragment",
		Aliases: []string{"profiles"},
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			profiles := app.FindProfiles(m, args[0])
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "profile.find", profiles, nil)
			}
			return renderProfileTable(cmd, profiles)
		},
	}
}

func newCheckProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "profile",
		Short:   "Report empty or unmarked profile directories",
		Aliases: []string{"profiles"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			findings := findingsForSubjectKind(app.Check(m), "profile")
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

// renderProfileSummaryTable is list profile's table — includes the key/host counts tdd.md §9
// documents ("Every directory carrying .hasp, with key/host counts") and the command's own
// --help text already promised.
func renderProfileSummaryTable(cmd *cobra.Command, summaries []app.ProfileSummary) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	if _, err := fmt.Fprintln(w, "PROFILE\tMANAGED\tKEYS\tHOSTS\tDIR"); err != nil {
		return err
	}
	for _, s := range summaries {
		if _, err := fmt.Fprintf(w, "%s\t%v\t%d\t%d\t%s\n", s.Profile.Path, s.Profile.Managed, s.KeyCount, s.HostCount, s.Profile.Dir); err != nil {
			return err
		}
	}
	return w.Flush()
}

// renderProfileTable is find profile's table — no counts, matching tdd.md §9's grid ("Match by
// name fragment," nothing about counts for find).
func renderProfileTable(cmd *cobra.Command, profiles []domain.Profile) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	if _, err := fmt.Fprintln(w, "PROFILE\tMANAGED\tDIR"); err != nil {
		return err
	}
	for _, p := range profiles {
		if _, err := fmt.Fprintf(w, "%s\t%v\t%s\n", p.Path, p.Managed, p.Dir); err != nil {
			return err
		}
	}
	return w.Flush()
}

func renderProfileDetail(cmd *cobra.Command, d app.ProfileDetail) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	if _, err := fmt.Fprintf(w, "Profile:\t%s\n", d.Profile.Path); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Dir:\t%s\n", d.Profile.Dir); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Keys:"); err != nil {
		return err
	}
	if len(d.Keys) == 0 {
		if _, err := fmt.Fprintln(w, "  (none)"); err != nil {
			return err
		}
	}
	for _, kr := range d.Keys {
		if _, err := fmt.Fprintf(w, "  %s\t(from %s)\n", kr.Key.Name, kr.FromProfile); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "Hosts:"); err != nil {
		return err
	}
	if len(d.Hosts) == 0 {
		if _, err := fmt.Fprintln(w, "  (none)"); err != nil {
			return err
		}
	}
	for _, hr := range d.Hosts {
		if _, err := fmt.Fprintf(w, "  %s\t(from %s)\n", hostPatternName(hr.Host.Patterns), hr.FromProfile); err != nil {
			return err
		}
	}
	return w.Flush()
}
