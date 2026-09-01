package render

import (
	"fmt"
	"io"

	"github.com/boweeb/hasp/internal/app"
)

// ANSI SGR codes for the write-plan diff renderer (tdd.md §10) — kept minimal and local to this
// package rather than a terminal-color library, matching P8's dependency discipline for what is
// purely cosmetic output.
const (
	ansiReset = "\x1b[0m"
	ansiGreen = "\x1b[32m"
	ansiRed   = "\x1b[31m"
)

// PlanPreviewData is the "plan.preview" envelope's Data payload (tdd.md §10): the Plan's own
// summary, plus every Change's Preview, in Plan order. Diff stays omitted (not null, not []) for
// any Change whose Preview().Diff is nil, since app.Preview.Diff already carries `json:",omitempty"`.
type PlanPreviewData struct {
	Summary string        `json:"summary"`
	Changes []app.Preview `json:"changes"`
}

// PlanPreviewJSON marshals plan as a "plan.preview"-kind envelope (tdd.md §10). This IS the
// preview call (P5, tdd.md §4 step 3) — it happens whether or not the write proceeds, and a
// zero-Change plan renders with an empty Changes array through this exact same call, not a
// special-cased branch.
func PlanPreviewJSON(w io.Writer, plan app.Plan) error {
	changes := make([]app.Preview, len(plan.Changes))
	for i, c := range plan.Changes {
		changes[i] = c.Preview()
	}
	return JSON(w, "plan.preview", PlanPreviewData{Summary: plan.Summary, Changes: changes}, nil)
}

// PlanPreviewHuman prints plan.Summary, then every Change's Preview: a summary line, and — only
// where Preview().Diff != nil — a unified-diff-style block beneath it, colorized unless noColor
// (tdd.md §10). A zero-Change plan prints as "nothing to do" — the same call, an empty loop body,
// no special-cased branch.
func PlanPreviewHuman(w io.Writer, plan app.Plan, noColor bool) error {
	if _, err := fmt.Fprintln(w, plan.Summary); err != nil {
		return err
	}
	if len(plan.Changes) == 0 {
		_, err := fmt.Fprintln(w, "  (nothing to do)")
		return err
	}
	for _, c := range plan.Changes {
		p := c.Preview()
		if _, err := fmt.Fprintf(w, "  - %s\n", p.Summary); err != nil {
			return err
		}
		for _, line := range p.Diff {
			if err := printDiffLine(w, line, noColor); err != nil {
				return err
			}
		}
	}
	return nil
}

func printDiffLine(w io.Writer, line app.DiffLine, noColor bool) error {
	// DiffElided is a synthetic summary line, not real file content — render it with no +/-/space
	// prefix and no diff coloring, so it reads unambiguously as "hasp collapsed this" rather than as
	// a line that ever existed in before/after (T26; windowing fix requirement 2).
	if line.Kind == app.DiffElided {
		_, err := fmt.Fprintln(w, "    "+line.Text)
		return err
	}

	prefix, code := " ", ""
	switch line.Kind {
	case app.DiffAdded:
		prefix, code = "+", ansiGreen
	case app.DiffRemoved:
		prefix, code = "-", ansiRed
	}
	text := prefix + line.Text
	if !noColor && code != "" {
		text = code + text + ansiReset
	}
	_, err := fmt.Fprintln(w, "    "+text)
	return err
}
