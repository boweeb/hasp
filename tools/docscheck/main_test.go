package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// entry builds a well-formed decision file.
func entry(id, body string) string {
	return "---\nid: decision-" + id + "\ntitle: 'A title'\ndate: '2026-11-25'\nstatus: 'Accepted'\n---\n" + body
}

const goodBody = "### Context\n\nWhy.\n\n### Decision\n\nWhat.\n\n### Consequence\n\nCost.\n"

// baseline is a minimal tree on which every check passes; each case overrides or adds files.
func baseline() map[string]string {
	return map[string]string{
		"docs/design.md":                         "# Design\n\n## 4. Principles\n\nP1 holds. Reality wins every time they disagree.\n",
		"docs/decision-log.md":                   "# Decision log\n\nRules live here.\n",
		"docs/tech-decision-log.md":              "# Tech decision log\n\nRules live here.\n",
		"backlog/decisions/decision-D1.md":       entry("D1", goodBody),
		"backlog/decisions/decision-D2.md":       entry("D2", goodBody),
		"backlog/decisions/decision-T1.md":       entry("T1", goodBody),
		"README.md":                              "# Root\n\nSee [the design](docs/design.md#4-principles).\n",
		"docs/guide.md":                          "# Guide\n\nSee [D1](../backlog/decisions/decision-D1.md) and [T1](../backlog/decisions/decision-T1.md).\n",
		"backlog/tasks/task-0001 - Something.md": "---\nid: TASK-0001\n---\n\nFinding U4.F1 is cited by T1.\n",
	}
}

func runTree(t *testing.T, overrides map[string]string, removals ...string) []string {
	t.Helper()
	root := t.TempDir()
	files := baseline()
	for k, v := range overrides {
		files[k] = v
	}
	for _, k := range removals {
		delete(files, k)
	}
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	findings, err := run(root)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range findings {
		out = append(out, f.String())
	}
	return out
}

func TestBaselineIsClean(t *testing.T) {
	if got := runTree(t, nil); len(got) != 0 {
		t.Fatalf("baseline must be clean, got:\n%s", strings.Join(got, "\n"))
	}
}

// Each case injects exactly one real defect and asserts the check that owns it reports it. A guard
// that passes either way is worthless (docs/tdd.md §17, the M3.6.4 lesson), so every row here was
// also confirmed to fail when the check it names is disabled.
func TestDefectsAreDetected(t *testing.T) {
	cases := []struct {
		name      string
		overrides map[string]string
		removals  []string
		want      string
	}{
		{"unresolved citation in docs", map[string]string{"docs/guide.md": "# Guide\n\nSee T9.\n"}, nil, "citation T9 does not resolve"},
		{"unresolved citation in a task", map[string]string{"backlog/tasks/task-0001 - Something.md": "Cites D77.\n"}, nil, "citation D77 does not resolve"},
		{"F-log reference", map[string]string{"docs/guide.md": "# Guide\n\nSee F3.\n"}, nil, "F-log reference"},
		{"missing Consequence", map[string]string{"backlog/decisions/decision-D2.md": entry("D2", "### Context\n\nWhy.\n")}, nil, "no line starting with \"### Consequence\""},
		{"gap in IDs", map[string]string{"backlog/decisions/decision-T3.md": entry("T3", goodBody)}, nil, "T2 is missing"},
		{"gap: D1 removed", nil, []string{"backlog/decisions/decision-D1.md"}, "D1 is missing"},
		{"stray F-ref beside a UAT finding ID", map[string]string{"backlog/tasks/task-0001 - Something.md": "Finding U4.F1 and stray F3.\n"}, nil, "F-log reference \"F3\""},
		{"duplicate number via zero-padded filename", map[string]string{"backlog/decisions/decision-D02.md": entry("D02", goodBody)}, nil, "appears in 2 files (duplicate)"},
		{"zero-padded filename", map[string]string{"backlog/decisions/decision-D02.md": entry("D02", goodBody)}, nil, "must not be zero-padded"},
		{"huge ID gap is capped, not enumerated", map[string]string{"backlog/decisions/decision-D999999999.md": entry("D999999999", goodBody)}, nil, "more are missing"},
		{"unresolved citation in a root file", map[string]string{"CLAUDE.md": "# Agents\n\nSee T99.\n"}, nil, "citation T99 does not resolve"},
		{"broken %20 link to a spaced task filename", map[string]string{"docs/guide.md": "# Guide\n\nSee [x](../backlog/tasks/task-0099%20-%20Nope.md).\n"}, nil, "does not exist"},
		{"broken angle-bracket link", map[string]string{"docs/guide.md": "# Guide\n\nSee [x](<../backlog/tasks/task-0099 - Nope.md>).\n"}, nil, "does not exist"},
		{"id disagrees with filename", map[string]string{"backlog/decisions/decision-D2.md": strings.Replace(entry("D2", goodBody), "decision-D2", "decision-D9", 1)}, nil, "does not match filename"},
		{"missing date", map[string]string{"backlog/decisions/decision-D2.md": strings.Replace(entry("D2", goodBody), "date: '2026-11-25'\n", "", 1)}, nil, "is not an ISO date"},
		{"missing status", map[string]string{"backlog/decisions/decision-D2.md": strings.Replace(entry("D2", goodBody), "status: 'Accepted'\n", "", 1)}, nil, "has no status"},
		{"backlog decision create output", map[string]string{"backlog/decisions/decision-1 - Minted.md": entry("1", goodBody)}, nil, "must be decision-D<n>.md"},
		{"broken link file", map[string]string{"docs/guide.md": "# Guide\n\nSee [x](nope.md).\n"}, nil, "link target file \"nope.md\" does not exist"},
		{"broken link fragment", map[string]string{"docs/guide.md": "# Guide\n\nSee [x](design.md#no-such-heading).\n"}, nil, "does not resolve in docs/design.md"},
		{"broken relative link out of a decision file", map[string]string{"backlog/decisions/decision-D2.md": entry("D2", goodBody+"\nSee [x](../../docs/missing.md).\n")}, nil, "does not exist"},
		{"broken link to a decision file", map[string]string{"docs/guide.md": "# Guide\n\nSee [D5](../backlog/decisions/decision-D5.md).\n"}, nil, "decision-D5.md"},
		{"mangled quote, relative-path link source", map[string]string{"backlog/decisions/decision-D2.md": entry("D2", goodBody+"\nPer [the design](../../docs/design.md), *\"Reality loses every time they disagree.\"*\n")}, nil, "does not appear verbatim"},
		{"mangled quote, bare filename link source", map[string]string{"docs/guide.md": "# Guide\n\nPer [the design](design.md), *\"Reality loses every time they disagree.\"*\n"}, nil, "does not appear verbatim"},
		{"mangled quote attributed to a decision file", map[string]string{"docs/guide.md": "# Guide\n\nPer [D1](../backlog/decisions/decision-D1.md), *\"Cost is unbounded.\"*\n"}, nil, "does not appear verbatim"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := strings.Join(runTree(t, c.overrides, c.removals...), "\n")
			if !strings.Contains(got, c.want) {
				t.Fatalf("expected a finding containing %q, got:\n%s", c.want, got)
			}
		})
	}
}

// Exemptions must stay narrow: the UAT finding ID shape is not an F-log reference, and a faithful
// quote through a relative-path link still passes.
func TestExemptionsAndFaithfulQuotes(t *testing.T) {
	got := runTree(t, map[string]string{
		// Backlog tasks quote text from outside the repository, so Check D must not read them.
		"backlog/tasks/task-0002 - Another.md":     "Finding U2.F1 and U5.F7. Per [the design](../../docs/design.md), *\"Reality loses every time they disagree.\"*\n",
		"backlog/tasks/task-0003 - Spaced name.md": "---\nid: TASK-0003\n---\n\nbody\n",
		"docs/guide.md":                    "# Guide\n\nSee [a](../backlog/tasks/task-0003%20-%20Spaced%20name.md) and [b](<../backlog/tasks/task-0003 - Spaced name.md>).\n",
		"backlog/decisions/decision-D2.md": entry("D2", goodBody+"\nPer [the design](../../docs/design.md), *\"Reality wins every time they disagree.\"*\n"),
	})
	if len(got) != 0 {
		t.Fatalf("expected no findings, got:\n%s", strings.Join(got, "\n"))
	}
}
