// Command hasp is the entry point. It delegates entirely to internal/cli.Run, which the
// testscript harness (internal/cli/e2e_test.go) also calls, so the two never drift.
package main

import (
	"os"

	"github.com/boweeb/hasp/internal/cli"
)

func main() {
	os.Exit(cli.Run())
}
