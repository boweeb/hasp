package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/boweeb/hasp/internal/adapter/sshagent"
	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/cli/render"
	"github.com/boweeb/hasp/internal/domain"
)

// investigateFlagUsage is shared by list key and show key's own --investigate registration so the
// help text is identical wherever it appears.
const investigateFlagUsage = "opt-in investigation mode (tdd.md §18): every registered fingerprint scheme, a confidence-graded origin, and ssh-agent-sourced facts, at the cost of speed and possibly a passphrase prompt"

// --investigate is registered as a LOCAL flag on exactly these two commands, deliberately not on
// root as a persistent flag alongside --json/--verbose/etc. tdd.md §9's own global-flags table
// lists it there for exposition (it is documented beside the other flags, and its effect is
// described once rather than twice), but its own text is explicit that it is "absent from every
// other verb×noun cell" — a persistent root flag would instead make cobra accept (and the
// generated CLI reference, docs/cli/, T34, advertise) `--investigate` on every command in the
// tree, including ones tdd.md §9's grid never mentions it for (`new key`, `check host`, ...),
// which would be a documentation lie the moment `docs/cli` regenerates. Registering it locally on
// only these two RunE closures is what keeps the surface matching the table's prose rather than
// its literal section heading.
func newListKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "List every key found, managed and unmanaged",
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			investigate, err := cmd.Flags().GetBool("investigate")
			if err != nil {
				return err
			}
			if !investigate {
				// P7/criterion 7: the default read path takes no new branch, opens no new
				// file, and does no new work at all when the flag is absent — this is that
				// exact pre-M3.6 code path, untouched.
				keys := app.ListKeys(m)
				if flags.JSON {
					return render.JSON(cmd.OutOrStdout(), "key.list", keys, nil)
				}
				return renderKeyTable(cmd, keys)
			}

			req := newInvestigateRequest()
			defer req.Gate.Close()
			investigated := app.InvestigateKeys(m, req)
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "key.list", investigated, nil)
			}
			if err := renderKeyTable(cmd, m.Keys); err != nil {
				return err
			}
			return renderInvestigatedKeys(cmd, investigated)
		},
	}
	cmd.Flags().Bool("investigate", false, investigateFlagUsage)
	return cmd
}

func newShowKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key <name>",
		Short: "Show full detail for one key: identity, locations, profiles, and bound hosts",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, flags, err := buildMachine(cmd)
			if err != nil {
				return err
			}
			investigate, err := cmd.Flags().GetBool("investigate")
			if err != nil {
				return err
			}
			if !investigate {
				detail, ok := app.ShowKey(m, args[0])
				if !ok {
					return fmt.Errorf("key %q not found", args[0])
				}
				if flags.JSON {
					return render.JSON(cmd.OutOrStdout(), "key.show", detail, nil)
				}
				return renderKeyDetail(cmd, detail)
			}

			req := newInvestigateRequest()
			defer req.Gate.Close()
			detail, ok := app.InvestigateKey(m, args[0], req)
			if !ok {
				return fmt.Errorf("key %q not found", args[0])
			}
			if flags.JSON {
				return render.JSON(cmd.OutOrStdout(), "key.show", detail, nil)
			}
			return renderInvestigatedKeyDetail(cmd, detail)
		},
	}
	cmd.Flags().Bool("investigate", false, investigateFlagUsage)
	return cmd
}

// newInvestigateRequest builds --investigate's app.InvestigateRequest exactly as tdd.md §18 and
// T38/T39 specify: the agent's currently loaded keys (sshagent.List is fail-open by its own
// contract — no agent, a dead socket, or a protocol error all silently produce a nil slice, never
// an error this function has to handle), and a passphrase gate whose prompt is wired in only when
// a real terminal is attached to stdin — never when stdin is a pipe or /dev/null, which is
// roadmap.md §5.6 exit criterion 5's own no-TTY-degrades case reaching PassphraseGate.Derive as
// a nil PassphraseFunc rather than a callback that would hang or error against a non-interactive
// fd. Every caller must `defer req.Gate.Close()` immediately after this call returns — the same
// pattern internal/cli/new.go:61 uses for `new key`'s own passphrase buffer
// (`defer zeroBytes(secret)`) — so a return added later on an error path can never accidentally
// skip zeroing D19's held passphrase.
func newInvestigateRequest() app.InvestigateRequest {
	facts := sshagent.List(os.Getenv("SSH_AUTH_SOCK"))
	gate := &app.PassphraseGate{}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		gate.Passphrase = promptPassphrase
	}
	return app.InvestigateRequest{AgentFacts: facts, Gate: gate}
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

// renderInvestigatedKeys is list key --investigate's human form (tdd.md §9's list-key cell,
// T35-T39): the ordinary key table (renderKeyTable, unchanged, printed by the caller before this
// runs) followed by one investigation block per key. P7's one-screen answer deliberately does not
// bind under this flag — tdd.md §9's own words: "deliberately high-friction, and accepted as
// such" — so every registered scheme is shown in full for every key, never truncated for table
// brevity the way the plain table's FINGERPRINT column already is.
func renderInvestigatedKeys(cmd *cobra.Command, keys []app.InvestigatedKey) error {
	for _, k := range keys {
		if err := renderInvestigation(cmd, k); err != nil {
			return err
		}
	}
	return nil
}

// renderInvestigation prints one key's --investigate section: every registered scheme in registry
// order (roadmap.md §5.6 criterion 2 — no scheme silently omitted, criterion 3's confidence value
// always one of domain.AllConfidences()), its origin evidence ("(none)" when the array is empty,
// per originsForInvestigate's own rule), and any agent-sourced comment, explicitly labelled as
// such. Shared by list key --investigate (once per key, after the plain table) and show key
// --investigate (appended below the existing detail block) so the two renderers can never diverge
// in *content* — only human vs. --json form (tdd.md §10, T13).
func renderInvestigation(cmd *cobra.Command, k app.InvestigatedKey) error {
	w := render.NewTabWriter(cmd.OutOrStdout())
	if _, err := fmt.Fprintf(w, "\nInvestigation: %s\n", k.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  SCHEME\tVALUE\tCONFIDENCE\tREASON"); err != nil {
		return err
	}
	for _, sf := range k.Schemes {
		value := sf.Value
		if value == "" {
			value = "unknown"
		}
		if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", sf.Scheme, value, sf.Confidence, orDash(string(sf.Reason))); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "  Origins:"); err != nil {
		return err
	}
	if len(k.Origins) == 0 {
		if _, err := fmt.Fprintln(w, "    (none)"); err != nil {
			return err
		}
	}
	for _, o := range k.Origins {
		if _, err := fmt.Fprintf(w, "    %s\t%s\n", o.ID, o.Confidence); err != nil {
			return err
		}
	}
	if k.CommentSource == domain.FactSourceAgent {
		if _, err := fmt.Fprintf(w, "  Comment (agent-sourced):\t%s\n", orDash(k.AgentComment)); err != nil {
			return err
		}
	}
	return w.Flush()
}

// renderInvestigatedKeyDetail is show key --investigate's human form: the existing detail block
// (renderKeyDetail, unchanged) with --investigate's own section appended below it (tdd.md §9's
// show-key cell: "the same surface as list key --investigate, scoped to one key").
func renderInvestigatedKeyDetail(cmd *cobra.Command, d app.InvestigatedKeyDetail) error {
	if err := renderKeyDetail(cmd, app.KeyDetail{Key: d.Key.Key, Hosts: d.Hosts}); err != nil {
		return err
	}
	return renderInvestigation(cmd, d.Key)
}
