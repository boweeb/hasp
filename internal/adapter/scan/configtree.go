package scan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/boweeb/hasp/internal/adapter/sshconfig"
)

// ConfigFile is one host-group file in the Include graph, already parsed. Path is the resolved
// (symlink-followed) canonical path — a symlinked ~/.ssh/config (common under dotfiles
// management) is read through to the real file it points at, matching P1: the data lives where
// it actually lives, and canonical paths are what cycle detection needs to be reliable.
type ConfigFile struct {
	Path string
	Body *sshconfig.File
}

// LoadConfigTree reads root (typically ~/.ssh/config) and recursively follows every Include
// directive it finds — inside a marked region or not, hasp-written or hand-written, no
// distinction — in file order, which is exactly the first-obtained-value-wins precedence order
// real ssh_config itself uses (T11). This is purely a read-side traversal: it needs none of
// T11's write-time Include-ordering machinery, only the order that already exists on disk.
//
// A missing root, a missing or unreadable Include target, or a cycle are all fail-open (§11):
// LoadConfigTree never errors for reasons like these — a machine with no config file yet, or a
// dangling Include, must not abort the survey (J1).
func LoadConfigTree(root string) ([]ConfigFile, error) {
	visited := map[string]bool{}
	var out []ConfigFile
	loadConfigFile(root, visited, &out)
	return out, nil
}

func loadConfigFile(path string, visited map[string]bool, out *[]ConfigFile) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return // missing or unreadable; fail-open
	}
	if visited[resolved] {
		return // cycle guard
	}
	visited[resolved] = true

	raw, err := os.ReadFile(resolved)
	if err != nil {
		return // fail-open
	}

	body := sshconfig.Parse(raw)
	*out = append(*out, ConfigFile{Path: resolved, Body: body})

	for _, inc := range includeTargets(body) {
		for _, target := range expandIncludeTarget(filepath.Dir(resolved), inc) {
			loadConfigFile(target, visited, out)
		}
	}
}

// includeTargets collects every Include directive's raw arguments from body's top level and
// from inside its MarkedRegion, if any — but not from inside a HostBlock, which is not a shape
// hasp itself ever writes an Include into and is out of scope to follow for M1.
func includeTargets(body *sshconfig.File) []string {
	var out []string
	var walk func(nodes []sshconfig.Node)
	walk = func(nodes []sshconfig.Node) {
		for _, n := range nodes {
			switch v := n.(type) {
			case *sshconfig.Directive:
				if strings.EqualFold(v.Keyword, "Include") {
					out = append(out, v.Args()...)
				}
			case *sshconfig.MarkedRegion:
				walk(v.Body)
			}
		}
	}
	walk(body.Nodes)
	return out
}

// expandIncludeTarget resolves one Include argument against ssh_config(5)'s own rules: "~/" is
// the user's home directory, a relative path is relative to the directory containing the
// current file (baseDir), and the result may be a glob — "will be expanded and processed in
// lexical order" (T11's own quote of the man page). A pattern matching nothing is returned
// as-is, a single literal target, so the caller's read attempt fails open in the usual way
// rather than silently vanishing here.
func expandIncludeTarget(baseDir, raw string) []string {
	expanded := raw
	switch {
	case strings.HasPrefix(expanded, "~/"):
		if home, err := os.UserHomeDir(); err == nil {
			expanded = filepath.Join(home, expanded[2:])
		}
	case !filepath.IsAbs(expanded):
		expanded = filepath.Join(baseDir, expanded)
	}

	matches, err := filepath.Glob(expanded)
	if err != nil || len(matches) == 0 {
		return []string{expanded}
	}
	sort.Strings(matches)
	return matches
}
