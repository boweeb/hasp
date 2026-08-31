package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/backup"
	"github.com/boweeb/hasp/internal/adapter/fswrite"
	"github.com/boweeb/hasp/internal/adapter/scan"
	"github.com/boweeb/hasp/internal/domain"
)

func TestReleaseProfileUseCase_Plan_RejectsEmptyRequest(t *testing.T) {
	uc := ReleaseProfileUseCase{}
	cases := []ReleaseProfileRequest{
		{Profile: domain.ProfilePath{"work"}}, // no KeyDir
		{KeyDir: "/tmp"},                      // no Profile
	}
	for i, req := range cases {
		if _, err := uc.Plan(req); !errors.Is(err, ErrUsage) {
			t.Errorf("case %d: err = %v, want ErrUsage", i, err)
		}
	}
}

// TestReleaseProfileUseCase_Plan_SingleChange confirms the Plan collapses to exactly one Remove
// Change, targeting the marker file itself, with RequiresBackup() true (P4).
func TestReleaseProfileUseCase_Plan_SingleChange(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	mkManagedProfileDir(t, profileDir)

	uc := ReleaseProfileUseCase{}
	plan, err := uc.Plan(ReleaseProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	c, ok := plan.Changes[0].(Remove)
	if !ok {
		t.Fatalf("Changes[0] = %T, want Remove", plan.Changes[0])
	}
	want := filepath.Join(profileDir, markerFileName)
	if c.Path != want {
		t.Errorf("Path = %s, want %s", c.Path, want)
	}
	if !c.RequiresBackup() {
		t.Error("Remove.RequiresBackup() = false, want true (P4)")
	}
}

// TestReleaseProfileUseCase_Plan_PreviewShowsPriorContent is D15 elaboration 4 / D18's own
// requirement, asserted directly against the rendered Preview: removing a marker the user has
// written a note into must never be a silent loss.
func TestReleaseProfileUseCase_Plan_PreviewShowsPriorContent(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	note := "# managed by hasp\n# a note I wrote myself\n"
	writeFile(t, filepath.Join(profileDir, markerFileName), note)

	uc := ReleaseProfileUseCase{}
	plan, err := uc.Plan(ReleaseProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	preview := plan.Changes[0].Preview()
	if len(preview.Diff) == 0 {
		t.Fatal("Preview().Diff is empty, want the marker's prior content rendered as removed lines")
	}
	var got string
	for _, line := range preview.Diff {
		if line.Kind != DiffRemoved {
			t.Errorf("Diff line %+v: Kind = %v, want DiffRemoved", line, line.Kind)
		}
		got += line.Text + "\n"
	}
	if got != note {
		t.Errorf("Preview().Diff reconstructs to %q, want %q", got, note)
	}
}

func TestReleaseProfileUseCase_Plan_RefusesUnmanaged(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}

	uc := ReleaseProfileUseCase{}
	_, err := uc.Plan(ReleaseProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan against an unmanaged directory: err = %v, want ErrUsage", err)
	}
}

// TestReleaseProfileUseCase_Plan_RejectsPathTraversal mirrors AdoptProfileUseCase's own guard.
func TestReleaseProfileUseCase_Plan_RejectsPathTraversal(t *testing.T) {
	uc := ReleaseProfileUseCase{}
	cases := []struct {
		name    string
		profile domain.ProfilePath
	}{
		{"traversal segment", domain.ProfilePath{"work", ".."}},
		{"embedded separator", domain.ProfilePath{"work/../../evil"}},
		{"empty segment", domain.ProfilePath{""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			_, err := uc.Plan(ReleaseProfileRequest{KeyDir: dir, Profile: tc.profile})
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("Plan error = %v, want ErrUsage", err)
			}
		})
	}
}

// TestReleaseProfileUseCase_RoundTrip applies adopt then release through the real Applier and
// BackupStore, and confirms: the backup fires for release (T8), the directory's non-marker
// contents are byte-for-byte unchanged (D18's content-preservation spirit), and the directory is
// no longer reported managed by scan.Profiles afterward.
func TestReleaseProfileUseCase_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(profileDir, "id_ed25519")
	writeFile(t, keyPath, "irreplaceable key bytes")

	adoptPlan, err := (AdoptProfileUseCase{}).Plan(AdoptProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("adopt Plan: %v", err)
	}
	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	if _, err := applier.Apply(adoptPlan); err != nil {
		t.Fatalf("adopt Apply: %v", err)
	}

	releasePlan, err := (ReleaseProfileUseCase{}).Plan(ReleaseProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("release Plan: %v", err)
	}
	if _, err := applier.Apply(releasePlan); err != nil {
		t.Fatalf("release Apply: %v", err)
	}

	if _, statErr := os.Lstat(filepath.Join(profileDir, markerFileName)); !os.IsNotExist(statErr) {
		t.Errorf(".hasp marker still present after release, statErr = %v", statErr)
	}
	got, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("key file missing after release: %v", err)
	}
	if string(got) != "irreplaceable key bytes" {
		t.Errorf("key content = %q, want unchanged", got)
	}

	backupDir := filepath.Join(dir, ".hasp-backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil || len(entries) == 0 {
		t.Errorf("no backup found under %s after release profile (P4): %v", backupDir, err)
	}

	candidates, err := scan.Profiles(dir)
	if err != nil {
		t.Fatalf("scan.Profiles: %v", err)
	}
	for _, c := range candidates {
		if c.Dir == profileDir && c.Managed {
			t.Errorf("profile %s still reported Managed after release", profileDir)
		}
	}
}

// TestReleaseProfileUseCase_Plan_WitnessCatchesMarkerEditedAfterPlan is the regression test for a
// reviewer finding: without a Witness, a note the user adds to .hasp during the preview-to-confirm
// window would be silently deleted on Apply, having never been shown in the preview it was
// supposedly guaranteed by (D15 elaboration 4 / D18: "never a silent loss"). This proves the
// Witness Plan now carries actually closes that race — mirroring the same pattern
// TestApplier_WitnessRace_Refused already establishes for the general mechanism (T30), applied to
// this specific real use case rather than a synthetic Change.
func TestReleaseProfileUseCase_Plan_WitnessCatchesMarkerEditedAfterPlan(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	mkManagedProfileDir(t, profileDir)
	marker := filepath.Join(profileDir, markerFileName)

	plan, err := (ReleaseProfileUseCase{}).Plan(ReleaseProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// Simulate the user adding a note to the marker during the preview-to-confirm window — after
	// Plan() already captured the marker's prior content and Witness, before Apply ever runs.
	edited := "# a note the user just wrote, never shown in the preview above\n"
	if err := os.WriteFile(marker, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	if _, err := applier.Apply(plan); !errors.Is(err, ErrPlanStale) {
		t.Fatalf("Apply after marker edited mid-window: err = %v, want ErrPlanStale", err)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("marker missing after refused Apply: %v", err)
	}
	if string(got) != edited {
		t.Errorf("marker content = %q, want the edit to survive unchanged: %q", got, edited)
	}

	backupDir := filepath.Join(dir, ".hasp-backups")
	if entries, readErr := os.ReadDir(backupDir); readErr == nil && len(entries) != 0 {
		t.Errorf("a backup was written for a Plan that never applied: %v", entries)
	}
}
