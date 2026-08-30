package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigTree_MarkedRegionAndHumanInclude(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "work.sshconfig"), "Host work\n    HostName work.example.com\n")
	writeFile(t, filepath.Join(dir, "personal.sshconfig"), "Host personal\n    HostName home.example.com\n")

	root := filepath.Join(dir, "config")
	writeFile(t, root, ""+
		"# >>> hasp:managed >>>\n"+
		"Include "+filepath.Join(dir, "work.sshconfig")+"\n"+
		"# <<< hasp:managed <<<\n"+
		"\n"+
		"# a human-written include outside hasp's markers\n"+
		"Include "+filepath.Join(dir, "personal.sshconfig")+"\n",
	)

	got, err := LoadConfigTree(root)
	if err != nil {
		t.Fatalf("LoadConfigTree: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("got %d config files, want 3 (root, work.sshconfig, personal.sshconfig): %+v", len(got), pathsOf(got))
	}

	// File order is Include order (first-obtained-value-wins precedence, T11): root first, then
	// its two Includes in the order they appear, marked-region one first.
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	resolvedWork, _ := filepath.EvalSymlinks(filepath.Join(dir, "work.sshconfig"))
	resolvedPersonal, _ := filepath.EvalSymlinks(filepath.Join(dir, "personal.sshconfig"))

	want := []string{resolvedRoot, resolvedWork, resolvedPersonal}
	for i, w := range want {
		if got[i].Path != w {
			t.Errorf("got[%d].Path = %q, want %q", i, got[i].Path, w)
		}
	}
}

func TestLoadConfigTree_CycleSafe(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.sshconfig")
	b := filepath.Join(dir, "b.sshconfig")
	writeFile(t, a, "Include "+b+"\n")
	writeFile(t, b, "Include "+a+"\n")

	got, err := LoadConfigTree(a)
	if err != nil {
		t.Fatalf("LoadConfigTree: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d config files, want 2 (a, b) — cycle must not loop forever: %+v", len(got), pathsOf(got))
	}
}

func TestLoadConfigTree_MissingRoot(t *testing.T) {
	dir := t.TempDir()
	got, err := LoadConfigTree(filepath.Join(dir, "does-not-exist"))
	if err != nil {
		t.Fatalf("LoadConfigTree: %v, want nil error (fail-open, §11)", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d config files for a missing root, want 0", len(got))
	}
}

func TestLoadConfigTree_MissingIncludeTargetSkipped(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "config")
	writeFile(t, root, "Include "+filepath.Join(dir, "missing.sshconfig")+"\n")

	got, err := LoadConfigTree(root)
	if err != nil {
		t.Fatalf("LoadConfigTree: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d config files, want 1 (root only; dangling Include skipped): %+v", len(got), pathsOf(got))
	}
}

func TestLoadConfigTree_GlobInclude(t *testing.T) {
	dir := t.TempDir()
	confd := filepath.Join(dir, "conf.d")
	writeFile(t, filepath.Join(confd, "a.sshconfig"), "Host a\n    HostName a.example.com\n")
	writeFile(t, filepath.Join(confd, "b.sshconfig"), "Host b\n    HostName b.example.com\n")

	root := filepath.Join(dir, "config")
	writeFile(t, root, "Include "+filepath.Join(confd, "*.sshconfig")+"\n")

	got, err := LoadConfigTree(root)
	if err != nil {
		t.Fatalf("LoadConfigTree: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d config files, want 3 (root, a, b): %+v", len(got), pathsOf(got))
	}
}

func pathsOf(files []ConfigFile) []string {
	var out []string
	for _, f := range files {
		out = append(out, f.Path)
	}
	return out
}

func TestIncludeTargets_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available")
	}
	got := expandIncludeTarget("/irrelevant", "~/foo.sshconfig")
	want := filepath.Join(home, "foo.sshconfig")
	if len(got) != 1 || got[0] != want {
		t.Errorf("expandIncludeTarget = %v, want [%s]", got, want)
	}
}
