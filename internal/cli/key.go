package cli

import (
	"fmt"
	"strings"

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
			matches, warnings := app.FindKeys(m, args[0])
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "key.find", matches, warnings)
			}
			return renderKeyFindTable(cmd, matches, warnings)
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
	if _, err := fmt.Fprintln(w, "SEVERITY\tID\tSUBJECT\tMESSAGE"); err != nil {
		return err
	}
	for _, f := range findings {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s %s\t%s\n", f.Severity, f.ID, f.Subject.Kind, f.Subject.Name, f.Message); err != nil {
			return err
		}
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
	if _, err := fmt.Fprintln(w, "NAME\tALGORITHM\tFORMAT\tENCRYPTED\tPROFILES\tCOMMENT\tFINGERPRINT"); err != nil {
		return err
	}
	for _, k := range keys {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\t%s\t%s\n",
			k.Name, orDash(k.Algorithm), k.Format, k.Encrypted, profilesCell(k.Profiles), orDash(k.Comment), truncatedFingerprintCell(k)); err != nil {
			return err
		}
	}
	return w.Flush()
}

// renderKeyFindTable is `find key`'s human form (T36, T48): one row per key per matched scheme,
// carrying both which scheme matched and the origin evidence that match is — SCHEME, ORIGIN, and
// CONFIDENCE never collapse into a single "matched" column, exactly as T36 requires (a match
// under aws-created-rsa/aws-imported-rsa is `confirmed` provenance; a match under
// ssh-native-sha256/legacy-ssh-md5 is `possible` and proves only that this is the right key,
// nothing about how it came to exist). SCHEME matters as its own column, not merely implied by
// ORIGIN, because for those latter two schemes Origin.ID *is* the scheme id (originForSchemeMatch,
// internal/app/key.go) — but for the two AWS schemes Origin.ID names the inferred origin
// (`aws-ec2-created`/`aws-ec2-imported`), a different namespace than SchemeID (domain.Origin's own
// doc comment) — so a reader needs SCHEME to see which of the two possible MD5 candidates
// actually matched at the MD5 shape's own ambiguity (T35/T36). Warnings — a scheme the clue's
// shape admitted but that could not be evaluated for some keys, T48 — print to stderr after the
// table: they are advisory, not part of the answer stdout carries (tdd.md §10).
func renderKeyFindTable(cmd *cobra.Command, matches []app.FindMatch, warnings []string) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	if _, err := fmt.Fprintln(w, "NAME\tSCHEME\tORIGIN\tCONFIDENCE"); err != nil {
		return err
	}
	for _, match := range matches {
		for _, o := range match.Origins {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", match.Key.Name, schemeFromOrigin(o), o.ID, o.Confidence); err != nil {
				return err
			}
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	for _, warning := range warnings {
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "warning:", warning); err != nil {
			return err
		}
	}
	return nil
}

// schemeFromOrigin recovers the scheme id that produced o from its own Because evidence
// (originForSchemeMatch, internal/app/key.go, always sets Because[0] to "scheme=<id>") — the
// renderer's only way to show SCHEME distinctly from ORIGIN without app.FindMatch growing a
// parallel field that duplicates what Because already carries as machine-readable evidence (T37).
func schemeFromOrigin(o domain.Origin) string {
	for _, b := range o.Because {
		if scheme, ok := strings.CutPrefix(string(b), "scheme="); ok {
			return scheme
		}
	}
	return "-"
}

func renderKeyDetail(cmd *cobra.Command, d app.KeyDetail) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	k := d.Key
	if _, err := fmt.Fprintf(w, "Name:\t%s\n", k.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Identity:\t%s (%s)\n", k.Identity.Value(), k.Identity.Kind()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Algorithm:\t%s\n", orDash(k.Algorithm)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Format:\t%s\n", k.Format); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Encrypted:\t%v\n", k.Encrypted); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Public half:\t%v\n", k.HasPublicHalf); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Comment:\t%s\n", orDash(k.Comment)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Profiles:\t%s\n", profilesCell(k.Profiles)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Locations:"); err != nil {
		return err
	}
	for _, loc := range k.Locations {
		kind := "file"
		if loc.IsAlias {
			kind = "alias"
		}
		if _, err := fmt.Fprintf(w, "  %s\t(%s)\n", loc.Path, kind); err != nil {
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
	for _, h := range d.Hosts {
		kind := bindingKindFor(h, k.Identity)
		if _, err := fmt.Fprintf(w, "  %s\t(%s)\n", hostPatternName(h.Patterns), kind); err != nil {
			return err
		}
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
