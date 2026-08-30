package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/boweeb/hasp/internal/app"
)

// ExitCode maps the error rootCmd.Execute() returns to one of tdd.md §10's four exit codes.
// Cobra itself never calls os.Exit.
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, app.ErrFindings):
		return 1
	case errors.Is(err, app.ErrUsage):
		return 2
	default:
		return 3
	}
}

// Run builds and executes the root command, printing any error to stderr — except check's own
// findings, which are already reported through the normal render path — and returns the process
// exit code. cmd/hasp's main() and the testscript harness both call this one function, so
// "what does a failure look like" is defined exactly once.
func Run() int {
	root := NewRootCmd()
	err := root.Execute()
	if err != nil && !errors.Is(err, app.ErrFindings) {
		fmt.Fprintln(os.Stderr, "hasp:", err)
	}
	return ExitCode(err)
}
