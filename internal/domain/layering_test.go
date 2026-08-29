package domain

import (
	"os/exec"
	"strings"
	"testing"
)

// TestLayeringGuard mechanically enforces T13's inward-dependency rule: internal/domain imports
// nothing outside the standard library. This is the "checked mechanically rather than left to
// convention" guard docs/roadmap.md §2 requires be wired before there is anything to violate it.
func TestLayeringGuard(t *testing.T) {
	modOut, err := exec.Command("go", "list", "-m").Output()
	if err != nil {
		t.Fatalf("go list -m: %v", err)
	}
	modulePath := strings.TrimSpace(string(modOut))

	depsOut, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps .: %v", err)
	}

	for _, dep := range strings.Split(strings.TrimSpace(string(depsOut)), "\n") {
		if dep == "" {
			continue
		}
		firstSegment := dep
		if i := strings.Index(dep, "/"); i >= 0 {
			firstSegment = dep[:i]
		}
		// The stdlib-vs-third-party tell: a dot in the first path segment (a domain name)
		// means it was fetched from somewhere other than GOROOT.
		if !strings.Contains(firstSegment, ".") {
			continue
		}
		if dep == modulePath || strings.HasPrefix(dep, modulePath+"/") {
			continue
		}
		t.Errorf("internal/domain must import nothing outside the standard library, but depends on %q (T13)", dep)
	}
}
