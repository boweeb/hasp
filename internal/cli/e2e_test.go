// End-to-end command-line tests via rogpeppe/go-internal/testscript (tdd.md §12): each .txtar
// case under testdata/script/ sets up a synthetic HOME/key-dir via env directives, runs hasp
// commands, and asserts against golden stdout/stderr. testscript.RunMain re-execs this same test
// binary as a subprocess whenever a script does "exec hasp ...", dispatching to the registered
// function below — the full read -> render -> exit-code cycle, exercised exactly as a real
// installed binary would be, with no separate build step.
package cli_test

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
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
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Cmds: map[string]func(*testscript.TestScript, bool, []string){
			"assert-exit-code": assertExitCode,
		},
	})
}

func assertExitCode(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("assert-exit-code does not support negation")
	}
	if len(args) < 2 {
		ts.Fatalf("usage: assert-exit-code <code> <command> [args...]")
	}
	expected, err := strconv.Atoi(args[0])
	if err != nil {
		ts.Fatalf("invalid exit code %q: %v", args[0], err)
	}

	err = ts.Exec(args[1], args[2:]...)
	actual := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			ts.Fatalf("run %q: %v", args[1], err)
		}
		actual = exitErr.ExitCode()
	}
	if actual != expected {
		ts.Fatalf("%q exited with code %d, want %d", args[1], actual, expected)
	}
}
