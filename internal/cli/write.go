// write.go is the generic write command-loop tdd.md §4 describes: given a built Plan, render its
// preview, gate on --yes/TTY confirmation, then apply. `new key` is the first verb to use it; a
// later phase's adopt/release/edit commands reuse runWritePlan rather than duplicate this logic.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/adapter/backup"
	"github.com/boweeb/hasp/internal/adapter/fswrite"
	"github.com/boweeb/hasp/internal/app"
	"github.com/boweeb/hasp/internal/cli/render"
)

// isStdinTTY reports whether os.Stdin is a real terminal — a package variable, not a bare call,
// so tests can force the "a TTY is attached" branch without a real pty (no such dependency is in
// the budget, tdd.md §2).
var isStdinTTY = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// runWritePlan is tdd.md §4's command-loop, steps 3-5 (steps 1-2 — parsing flags into a Request
// and calling UseCase.Plan — are each command's own job before calling this):
//
//  1. Render every Change's Preview() — this call IS the preview (P5); it happens whether or not
//     the write proceeds.
//  2. If --yes was passed, skip straight to apply.
//  3. Otherwise, if stdin is a TTY, prompt to confirm; if not, fail closed (app.ErrUsage, §11's
//     "a write is requested with no TTY and no --yes" row).
//  4. On confirmation, Applier.Apply(plan) runs against the real fswrite/backup adapters.
//
// applied reports whether Apply actually ran — false with a nil error means the user declined a
// TTY confirmation, which is a normal outcome (exit 0), not a usage error. A zero-Change Plan
// still flows through every step above; there is no special-cased no-op branch (tdd.md §4).
func runWritePlan(cmd *cobra.Command, flags globalFlags, plan app.Plan) (result app.Result, applied bool, err error) {
	if err := renderPlanPreview(cmd, flags, plan); err != nil {
		return app.Result{}, false, err
	}

	if !flags.Yes {
		confirmed, cErr := confirmApply(cmd)
		if cErr != nil {
			return app.Result{}, false, cErr
		}
		if !confirmed {
			return app.Result{}, false, nil
		}
	}

	applier := app.Applier{FS: fswrite.New(), Backups: backup.New(flags.KeyDir)}
	res, err := applier.Apply(plan)
	if err != nil {
		return app.Result{}, false, err
	}
	return res, true, nil
}

func renderPlanPreview(cmd *cobra.Command, flags globalFlags, plan app.Plan) error {
	if flags.JSON {
		return render.PlanPreviewJSON(cmd.OutOrStdout(), plan)
	}
	return render.PlanPreviewHuman(cmd.OutOrStdout(), plan, flags.NoColor)
}

// confirmApply prompts y/N on stdout/stdin when stdin is a TTY, and fails closed (app.ErrUsage,
// exit code 2) otherwise — tdd.md §11's "a write is requested with no TTY and no --yes" row.
func confirmApply(cmd *cobra.Command) (bool, error) {
	if !isStdinTTY() {
		return false, fmt.Errorf("%w: refusing to apply without a confirmation: stdin is not a terminal and --yes was not given", app.ErrUsage)
	}

	fmt.Fprint(cmd.OutOrStdout(), "Apply these changes? [y/N] ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	if strings.EqualFold(line, "y") || strings.EqualFold(line, "yes") {
		return true, nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), "not confirmed; nothing applied")
	return false, nil
}
