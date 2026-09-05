package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestLayeringGuard mechanically enforces T32's build-time-only dependency rule: cmd/hasp's own
// dependency graph must never carry github.com/magefile/mage. A magefile importing it is fine;
// cmd/hasp importing it is a regression. This extends T13's existing internal/domain layering
// guard (internal/domain/layering_test.go) to the mage dependency budget.
func TestLayeringGuard(t *testing.T) {
	depsOut, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps .: %v", err)
	}

	const forbidden = "github.com/magefile/mage"

	for _, dep := range strings.Split(strings.TrimSpace(string(depsOut)), "\n") {
		if dep == "" {
			continue
		}
		if dep == forbidden || strings.HasPrefix(dep, forbidden+"/") {
			t.Errorf("cmd/hasp must never depend on %q, but it does (T32)", forbidden)
		}
	}
}
