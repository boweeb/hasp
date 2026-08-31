package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/boweeb/hasp/internal/app"
)

// swapStdin replaces os.Stdin for the duration of the test with a pipe fed by content, and
// restores the original on cleanup — the standard technique for driving code that reads directly
// from os.Stdin (write.go's confirmApply, tdd.md §4 step 4) without a real terminal.
func swapStdin(t *testing.T, content string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	original := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = original })
}

func newTestCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{Use: "test"}
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	return cmd, out
}

func TestRunWritePlan_YesSkipsPromptAndApplies(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "example")
	plan := app.Plan{
		Summary: "write example",
		Changes: []app.Change{app.WriteKeyFile{Path: target, Contents: []byte("hello"), Mode: 0o600}},
	}

	cmd, out := newTestCmd()
	flags := globalFlags{KeyDir: dir, Yes: true}

	// isStdinTTY must never be consulted when --yes is set — force it to panic if it is, so an
	// accidental prompt shows up as a hard test failure rather than a silent hang.
	orig := isStdinTTY
	isStdinTTY = func() bool { t.Fatal("isStdinTTY consulted despite --yes"); return false }
	defer func() { isStdinTTY = orig }()

	result, applied, err := runWritePlan(cmd, flags, plan)
	if err != nil {
		t.Fatalf("runWritePlan: %v", err)
	}
	if !applied {
		t.Fatal("applied = false, want true")
	}
	if len(result.Applied) != 1 {
		t.Errorf("result.Applied = %+v, want 1 entry", result.Applied)
	}
	if _, statErr := os.Stat(target); statErr != nil {
		t.Errorf("target file was not written: %v", statErr)
	}
	if out.Len() == 0 {
		t.Error("no preview was rendered")
	}
}

func TestRunWritePlan_NoTTYNoYes_FailsClosed(t *testing.T) {
	dir := t.TempDir()
	plan := app.Plan{
		Summary: "write example",
		Changes: []app.Change{app.WriteKeyFile{Path: filepath.Join(dir, "example"), Contents: []byte("hello"), Mode: 0o600}},
	}

	cmd, _ := newTestCmd()
	flags := globalFlags{KeyDir: dir, Yes: false}

	orig := isStdinTTY
	isStdinTTY = func() bool { return false }
	defer func() { isStdinTTY = orig }()

	_, applied, err := runWritePlan(cmd, flags, plan)
	if !errors.Is(err, app.ErrUsage) {
		t.Fatalf("runWritePlan error = %v, want ErrUsage (tdd.md §11: no TTY, no --yes -> fail closed)", err)
	}
	if applied {
		t.Error("applied = true, want false")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "example")); statErr == nil {
		t.Error("target file was written despite the fail-closed refusal")
	}
}

func TestRunWritePlan_TTYConfirmed_Applies(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "example")
	plan := app.Plan{
		Summary: "write example",
		Changes: []app.Change{app.WriteKeyFile{Path: target, Contents: []byte("hello"), Mode: 0o600}},
	}

	cmd, _ := newTestCmd()
	flags := globalFlags{KeyDir: dir, Yes: false}

	orig := isStdinTTY
	isStdinTTY = func() bool { return true }
	defer func() { isStdinTTY = orig }()
	swapStdin(t, "y\n")

	_, applied, err := runWritePlan(cmd, flags, plan)
	if err != nil {
		t.Fatalf("runWritePlan: %v", err)
	}
	if !applied {
		t.Fatal("applied = false, want true")
	}
	if _, statErr := os.Stat(target); statErr != nil {
		t.Errorf("target file was not written: %v", statErr)
	}
}

func TestRunWritePlan_TTYDeclined_DoesNotApply(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "example")
	plan := app.Plan{
		Summary: "write example",
		Changes: []app.Change{app.WriteKeyFile{Path: target, Contents: []byte("hello"), Mode: 0o600}},
	}

	cmd, _ := newTestCmd()
	flags := globalFlags{KeyDir: dir, Yes: false}

	orig := isStdinTTY
	isStdinTTY = func() bool { return true }
	defer func() { isStdinTTY = orig }()
	swapStdin(t, "n\n")

	_, applied, err := runWritePlan(cmd, flags, plan)
	if err != nil {
		t.Fatalf("runWritePlan: %v, want nil (declining is not a usage error)", err)
	}
	if applied {
		t.Error("applied = true, want false")
	}
	if _, statErr := os.Stat(target); statErr == nil {
		t.Error("target file was written despite declining the confirm prompt")
	}
}

func TestRunWritePlan_ZeroChanges_RendersNothingToDoAndApplies(t *testing.T) {
	dir := t.TempDir()
	plan := app.Plan{Summary: "nothing to do"}

	cmd, out := newTestCmd()
	flags := globalFlags{KeyDir: dir, Yes: true}

	result, applied, err := runWritePlan(cmd, flags, plan)
	if err != nil {
		t.Fatalf("runWritePlan: %v", err)
	}
	if !applied {
		t.Error("applied = false, want true (Apply on a zero-Change Plan is a legitimate no-op)")
	}
	if len(result.Applied) != 0 {
		t.Errorf("result.Applied = %+v, want empty", result.Applied)
	}
	if out.Len() == 0 {
		t.Error("no preview was rendered for the zero-Change plan")
	}
}
