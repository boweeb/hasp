package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/cli/render"
	"github.com/boweeb/hasp/internal/domain"
)

func newListKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "key",
		Short: "List every key found, managed and unmanaged",
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			keys := app.ListKeys(m)
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "key.list", keys, nil)
			}
			return renderKeyTable(cmd, keys)
		},
	}
}

func newShowKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "key <name>",
		Short: "Show full detail for one key: identity, locations, profiles, and bound hosts",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			detail, ok := app.ShowKey(m, args[0])
			if !ok {
				return fmt.Errorf("key %q not found", args[0])
			}
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "key.show", detail, nil)
			}
			return renderKeyDetail(cmd, detail)
		},
	}
}

func newFindKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "key <fingerprint-fragment>",
		Short: "Identify a key from a fingerprint fragment, ignoring punctuation and case",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			keys := app.FindKeys(m, args[0])
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "key.find", keys, nil)
			}
			return renderKeyTable(cmd, keys)
		},
	}
}

func newCheckKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "key",
		Short: "Report duplicate, incomplete, or unaffiliated keys",
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			findings := findingsForSubjectKind(app.Check(m), "key")
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

func findingsForSubjectKind(findings []app.Finding, kind string) []app.Finding {
	var out []app.Finding
	for _, f := range findings {
		if f.Subject.Kind == kind {
			out = append(out, f)
		}
	}
	return out
}

func renderFindings(cmd *cobra.Command, flags globalFlags, findings []app.Finding) error {
	if flags.JSON {
		return render.JSON(cmd.OutOrStdout(), "check.report", findings, nil)
	}
	return renderFindingsTable(cmd, findings)
}

func renderFindingsTable(cmd *cobra.Command, findings []app.Finding) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	if len(findings) == 0 {
		_, err := fmt.Fprintln(w, "clean — no findings")
		if err != nil {
			return err
		}
		return w.Flush()
	}
	fmt.Fprintln(w, "SEVERITY\tID\tSUBJECT\tMESSAGE")
	for _, f := range findings {
		fmt.Fprintf(w, "%s\t%s\t%s %s\t%s\n", f.Severity, f.ID, f.Subject.Kind, f.Subject.Name, f.Message)
	}
	return w.Flush()
}

// renderKeyTable's column order puts the fixed-width facts (name, algorithm, format, encrypted,
// profiles, comment) before the one long, variable-width column (fingerprint) — a long trailing
// column disrupts scanning far less than one in the middle, and the fingerprint itself is
// truncated here (full value: `show key`, or --json) so a personal-scale inventory (P8) stays a
// one-screen answer (P7) even at a dozen-plus keys.
func renderKeyTable(cmd *cobra.Command, keys []domain.Key) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	fmt.Fprintln(w, "NAME\tALGORITHM\tFORMAT\tENCRYPTED\tPROFILES\tCOMMENT\tFINGERPRINT")
	for _, k := range keys {
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\t%s\t%s\n",
			k.Name, orDash(k.Algorithm), k.Format, k.Encrypted, profilesCell(k.Profiles), orDash(k.Comment), truncatedFingerprintCell(k))
	}
	return w.Flush()
}

func renderKeyDetail(cmd *cobra.Command, d app.KeyDetail) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	k := d.Key
	fmt.Fprintf(w, "Name:\t%s\n", k.Name)
	fmt.Fprintf(w, "Identity:\t%s (%s)\n", k.Identity.Value(), k.Identity.Kind())
	fmt.Fprintf(w, "Algorithm:\t%s\n", orDash(k.Algorithm))
	fmt.Fprintf(w, "Format:\t%s\n", k.Format)
	fmt.Fprintf(w, "Encrypted:\t%v\n", k.Encrypted)
	fmt.Fprintf(w, "Public half:\t%v\n", k.HasPublicHalf)
	fmt.Fprintf(w, "Comment:\t%s\n", orDash(k.Comment))
	fmt.Fprintf(w, "Profiles:\t%s\n", profilesCell(k.Profiles))
	fmt.Fprintln(w, "Locations:")
	for _, loc := range k.Locations {
		kind := "file"
		if loc.IsAlias {
			kind = "alias"
		}
		fmt.Fprintf(w, "  %s\t(%s)\n", loc.Path, kind)
	}
	fmt.Fprintln(w, "Hosts:")
	if len(d.Hosts) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, h := range d.Hosts {
		kind := bindingKindFor(h, k.Identity)
		fmt.Fprintf(w, "  %s\t(%s)\n", hostPatternName(h.Patterns), kind)
	}
	return w.Flush()
}

func bindingKindFor(h domain.Host, identity domain.KeyIdentity) domain.BindingKind {
	target := identityMapKeyForRender(identity)
	for _, b := range h.Bindings {
		if identityMapKeyForRender(b.Key) == target {
			return b.Kind
		}
	}
	return domain.BindingExplicit
}

func identityMapKeyForRender(id domain.KeyIdentity) string {
	return id.Kind().String() + ":" + id.Value()
}

// truncatedFingerprintCell shows enough of a SHA256 fingerprint to eyeball at a glance in a
// table, without the full ~52-character value dominating every row. At this project's scale
// (P8 — tens of keys, never thousands) a short prefix is not a meaningful collision risk; a
// consumer who needs the exact value has `show key <name>` or --json.
func truncatedFingerprintCell(k domain.Key) string {
	if k.Identity.Kind() != domain.IdentityFingerprint {
		return "unknown"
	}
	return truncatedIdentityValue(k.Identity)
}

func profilesCell(paths []domain.ProfilePath) string {
	if len(paths) == 0 {
		return "(none)"
	}
	s := ""
	for i, p := range paths {
		if i > 0 {
			s += ", "
		}
		s += p.String()
	}
	return s
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
