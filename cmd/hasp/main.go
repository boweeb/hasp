// Command hasp is the entry point. It wires internal/cli and maps the error rootCmd.Execute()
// returns to a process exit code.
//
// docs/tdd.md §10 defines a full four-code taxonomy (0 clean, 1 check findings, 2 usage, 3
// error) for M1 onward, once there are real commands to distinguish. M0 has only `version`,
// which never fails in a way that needs more than "success (0) vs generic error (3)" — the
// richer taxonomy is wired as the verbs that need it (check, and fail-closed preconditions)
// arrive.
package main

import (
	"fmt"
	"os"

	"github.com/boweeb/hasp/internal/cli"
)

func main() {
	os.Exit(run())
}

func run() int {
	root := cli.NewRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "hasp:", err)
		return 3
	}
	return 0
}
