// End-to-end command-line tests via rogpeppe/go-internal/testscript (tdd.md §12): each .txtar
// case under testdata/script/ sets up a synthetic HOME/key-dir via env directives, runs hasp
// commands, and asserts against golden stdout/stderr. testscript.RunMain re-execs this same test
// binary as a subprocess whenever a script does "exec hasp ...", dispatching to the registered
// function below — the full read -> render -> exit-code cycle, exercised exactly as a real
// installed binary would be, with no separate build step.
package cli_test

import (
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/boweeb/hasp/internal/cli"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"hasp": func() { os.Exit(cli.Run()) },
	})
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{Dir: "testdata/script"})
}
