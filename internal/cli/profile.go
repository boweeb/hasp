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
		Use:   "profile",
		Short: "List every directory carrying a .hasp marker, with key/host counts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			profiles := app.ListProfiles(m)
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "profile.list", profiles, nil)
			}
			return renderProfileTable(cmd, profiles)
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
		Use:   "profile <clue>",
		Short: "Match by name fragment",
		Args:  exactArgs(1),
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
		Use:   "profile",
		Short: "Report empty or unmarked profile directories",
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

func renderProfileTable(cmd *cobra.Command, profiles []domain.Profile) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	fmt.Fprintln(w, "PROFILE\tMANAGED\tDIR")
	for _, p := range profiles {
		fmt.Fprintf(w, "%s\t%v\t%s\n", p.Path, p.Managed, p.Dir)
	}
	return w.Flush()
}

func renderProfileDetail(cmd *cobra.Command, d app.ProfileDetail) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	fmt.Fprintf(w, "Profile:\t%s\n", d.Profile.Path)
	fmt.Fprintf(w, "Dir:\t%s\n", d.Profile.Dir)
	fmt.Fprintln(w, "Keys:")
	if len(d.Keys) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, kr := range d.Keys {
		fmt.Fprintf(w, "  %s\t(from %s)\n", kr.Key.Name, kr.FromProfile)
	}
	fmt.Fprintln(w, "Hosts:")
	if len(d.Hosts) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, hr := range d.Hosts {
		fmt.Fprintf(w, "  %s\t(from %s)\n", hostPatternName(hr.Host.Patterns), hr.FromProfile)
	}
	return w.Flush()
}
