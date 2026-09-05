package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestLayeringGuard mechanically enforces T32's build-time-only dependency rule: cmd/hasp's own
// dependency graph must never carry github.com/magefile/mage, nor tools/gendocs's
// github.com/spf13/cobra/doc chain (its go-md2man/blackfriday transitive deps are covered by
// forbidding the doc subpackage itself). A magefile or dev-time tool importing either is fine;
// cmd/hasp importing either is a regression. This extends T13's existing internal/domain layering
// guard (internal/domain/layering_test.go) to the mage and gendocs dependency budgets.
func TestLayeringGuard(t *testing.T) {
	depsOut, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps .: %v", err)
	}

	forbidden := []string{"github.com/magefile/mage", "github.com/spf13/cobra/doc"}

	for _, dep := range strings.Split(strings.TrimSpace(string(depsOut)), "\n") {
		if dep == "" {
			continue
		}
		for _, f := range forbidden {
			if dep == f || strings.HasPrefix(dep, f+"/") {
				t.Errorf("cmd/hasp must never depend on %q, but it does (T32)", f)
			}
		}
	}
}
