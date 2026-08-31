package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/boweeb/hasp/internal/adapter/backup"
	"github.com/boweeb/hasp/internal/adapter/fswrite"
)

// noopFS is a WriteFS that fails any call — used by tests whose Change stubs never touch it, so
// a stray call is a loud test failure rather than a silent success.
type noopFS struct{ t *testing.T }

func (f noopFS) WriteFile(string, []byte, os.FileMode) error {
	f.t.Fatal("unexpected WriteFS.WriteFile call")
	return nil
}
func (f noopFS) Symlink(string, string) error {
	f.t.Fatal("unexpected WriteFS.Symlink call")
	return nil
}
func (f noopFS) Remove(string) error {
	f.t.Fatal("unexpected WriteFS.Remove call")
	return nil
}
func (f noopFS) Move(string, string) error {
	f.t.Fatal("unexpected WriteFS.Move call")
	return nil
}

// spyBackups records every path Snapshot was called with, in order — used to assert Applier's
// backup-before-apply wiring without depending on the real backup.Store's filesystem effects.
type spyBackups struct{ calls []string }

func (s *spyBackups) Snapshot(path string) error {
	s.calls = append(s.calls, path)
	return nil
}

// stubChange is a minimal Change used to test Applier's own control flow (backup timing, apply
// ordering) independent of any real concrete Change kind's Apply mechanics.
type stubChange struct {
	summary        string
	requiresBackup bool
	path           string // backupPath(); only consulted when requiresBackup is true
	applyErr       error
	applied        *bool // set true when Apply runs, if non-nil
}

func (c stubChange) Preview() Preview     { return Preview{Summary: c.summary} }
func (c stubChange) RequiresBackup() bool { return c.requiresBackup }
func (c stubChange) backupPath() string   { return c.path }
func (c stubChange) Apply(_ WriteFS) error {
	if c.applied != nil {
		*c.applied = true
	}
	return c.applyErr
}

func TestApplier_ZeroChangesIsANoop(t *testing.T) {
	dir := t.TempDir()
	a := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}

	result, err := a.Apply(Plan{Summary: "nothing to do"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("Applied = %+v, want empty", result.Applied)
	}
}

func TestApplier_BacksUpOnlyWhenRequired(t *testing.T) {
	spy := &spyBackups{}
	a := &Applier{FS: noopFS{t}, Backups: spy}

	var applied1, applied2 bool
	plan := Plan{Changes: []Change{
		stubChange{summary: "no backup", requiresBackup: false, applied: &applied1},
		stubChange{summary: "needs backup", requiresBackup: true, path: "/tmp/example", applied: &applied2},
	}}

	if _, err := a.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !applied1 || !applied2 {
		t.Fatalf("applied1=%v applied2=%v, want both true", applied1, applied2)
	}
	if len(spy.calls) != 1 || spy.calls[0] != "/tmp/example" {
		t.Errorf("Backups.Snapshot calls = %v, want exactly one call for /tmp/example", spy.calls)
	}
}

func TestApplier_BackupFailureAbortsBeforeApply(t *testing.T) {
	backupErr := errors.New("disk full")
	a := &Applier{FS: noopFS{t}, Backups: failingBackups{err: backupErr}}

	var applied bool
	plan := Plan{Changes: []Change{
		stubChange{summary: "needs backup", requiresBackup: true, path: "/tmp/example", applied: &applied},
	}}

	_, err := a.Apply(plan)
	if err == nil || !errors.Is(err, backupErr) {
		t.Fatalf("Apply error = %v, want it to wrap %v", err, backupErr)
	}
	if applied {
		t.Error("Change.Apply ran despite a failed backup")
	}
}

type failingBackups struct{ err error }

func (f failingBackups) Snapshot(string) error { return f.err }

// TestApplier_WitnessRace_Refused is the guard test for T30's preview/apply race: a Plan is built
// (capturing a Witness), the target file changes underneath it, and Apply must refuse the whole
// Plan before writing anything, with no backup taken.
func TestApplier_WitnessRace_Refused(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := "# >>> hasp:managed >>>\nInclude foo\n# <<< hasp:managed <<<\n"
	writeFile(t, configPath, original)

	witness, err := NewWitness(configPath)
	if err != nil {
		t.Fatalf("NewWitness: %v", err)
	}
	plan := Plan{
		Witnesses: []Witness{witness},
		Changes: []Change{
			WriteRegion{File: configPath, Marker: "managed", Before: []byte("Include foo\n"), After: []byte("Include bar\n")},
		},
	}

	// The race: something else touches the file between Plan() and Apply().
	mutated := "# >>> hasp:managed >>>\nInclude someone-else-changed-this\n# <<< hasp:managed <<<\n"
	writeFile(t, configPath, mutated)

	a := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	_, err = a.Apply(plan)
	if !errors.Is(err, ErrPlanStale) {
		t.Fatalf("Apply error = %v, want ErrPlanStale", err)
	}

	got, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("read %s: %v", configPath, readErr)
	}
	if string(got) != mutated {
		t.Errorf("config content = %q, want the mutation left in place (%q)", got, mutated)
	}

	if entries, statErr := os.ReadDir(filepath.Join(dir, ".hasp-backups")); statErr == nil && len(entries) != 0 {
		t.Errorf(".hasp-backups has %d entries, want none — a stale plan must never be backed up", len(entries))
	}
}

// TestApplier_WitnessRace_ByteIdenticalRewriteIsNotARace is the companion case: the file was
// rewritten between Plan() and Apply() but to byte-identical content. T30: "a file rewritten to
// byte-identical content is not a race," so Apply must proceed normally.
func TestApplier_WitnessRace_ByteIdenticalRewriteIsNotARace(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := "# >>> hasp:managed >>>\nInclude foo\n# <<< hasp:managed <<<\n"
	writeFile(t, configPath, original)

	witness, err := NewWitness(configPath)
	if err != nil {
		t.Fatalf("NewWitness: %v", err)
	}
	plan := Plan{
		Witnesses: []Witness{witness},
		Changes: []Change{
			WriteRegion{File: configPath, Marker: "managed", Before: []byte("Include foo\n"), After: []byte("Include bar\n")},
		},
	}

	// Rewritten to the exact same bytes, but with a distinctly later mtime — a touched mtime
	// alone must never trip the guard (T30: "hasp does not refuse on a touched mtime alone").
	if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
		t.Fatalf("rewrite %s: %v", configPath, err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(configPath, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	a := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	if _, err := a.Apply(plan); err != nil {
		t.Fatalf("Apply: %v, want a byte-identical rewrite to proceed normally", err)
	}

	got, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("read %s: %v", configPath, readErr)
	}
	want := "# >>> hasp:managed >>>\nInclude bar\n# <<< hasp:managed <<<\n"
	if string(got) != want {
		t.Errorf("config content = %q, want %q", got, want)
	}

	entries, err := os.ReadDir(filepath.Join(dir, ".hasp-backups"))
	if err != nil {
		t.Fatalf("read .hasp-backups: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf(".hasp-backups has %d entries, want exactly 1", len(entries))
	}
}
