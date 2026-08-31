package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fswrite"
	"github.com/boweeb/hasp/internal/adapter/scan"
	"github.com/boweeb/hasp/internal/domain"
)

func TestAdoptProfileUseCase_Plan_RejectsEmptyRequest(t *testing.T) {
	uc := AdoptProfileUseCase{}
	cases := []AdoptProfileRequest{
		{Profile: domain.ProfilePath{"work"}}, // no KeyDir
		{KeyDir: "/tmp"},                      // no Profile
	}
	for i, req := range cases {
		if _, err := uc.Plan(req); !errors.Is(err, ErrUsage) {
			t.Errorf("case %d: err = %v, want ErrUsage", i, err)
		}
	}
}

// TestAdoptProfileUseCase_Plan_SingleChange confirms the Plan collapses to exactly one
// CreateMarker Change (tdd.md §9's `adopt` grid cell), and that its Header carries D15
// elaboration 3's `#`-prefixed self-description.
func TestAdoptProfileUseCase_Plan_SingleChange(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work", "foobarco")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(profileDir, "id_ed25519"), "key bytes")

	uc := AdoptProfileUseCase{}
	plan, err := uc.Plan(AdoptProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work", "foobarco"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	c, ok := plan.Changes[0].(CreateMarker)
	if !ok {
		t.Fatalf("Changes[0] = %T, want CreateMarker", plan.Changes[0])
	}
	if c.Dir != profileDir {
		t.Errorf("Dir = %s, want %s", c.Dir, profileDir)
	}
	if len(c.Header) == 0 {
		t.Fatal("Header is empty, want D15 elaboration 3's # header")
	}
	for _, line := range strings.Split(strings.TrimRight(string(c.Header), "\n"), "\n") {
		if !strings.HasPrefix(line, "#") {
			t.Errorf("Header line %q is not #-prefixed (D15 elaboration 3)", line)
		}
	}
}

// TestAdoptProfileUseCase_Plan_RefusesMissingDirectory guards the grid cell's own precondition:
// adopt profile marks an *existing* directory, it never creates one.
func TestAdoptProfileUseCase_Plan_RefusesMissingDirectory(t *testing.T) {
	dir := t.TempDir()
	uc := AdoptProfileUseCase{}
	_, err := uc.Plan(AdoptProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan against a missing directory: err = %v, want ErrUsage", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "work")); statErr == nil {
		t.Error("Plan created the target directory; adopt profile must never mkdir")
	}
}

func TestAdoptProfileUseCase_Plan_RefusesAlreadyManaged(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(profileDir, markerFileName), "# already a profile\n")

	uc := AdoptProfileUseCase{}
	_, err := uc.Plan(AdoptProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan against an already-managed directory: err = %v, want ErrUsage", err)
	}
	got, readErr := os.ReadFile(filepath.Join(profileDir, markerFileName))
	if readErr != nil {
		t.Fatalf("read marker: %v", readErr)
	}
	if string(got) != "# already a profile\n" {
		t.Errorf("marker content = %q, want unchanged", got)
	}
}

// TestAdoptProfileUseCase_Plan_RejectsPathTraversal mirrors AdoptKeyUseCase's own regression test:
// a malicious profile segment must be rejected before any Change is planned.
func TestAdoptProfileUseCase_Plan_RejectsPathTraversal(t *testing.T) {
	uc := AdoptProfileUseCase{}
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
			_, err := uc.Plan(AdoptProfileRequest{KeyDir: dir, Profile: tc.profile})
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("Plan error = %v, want ErrUsage", err)
			}
			entries, readErr := os.ReadDir(dir)
			if readErr != nil {
				t.Fatalf("ReadDir(%s): %v", dir, readErr)
			}
			if len(entries) != 0 {
				t.Errorf("Plan for profile %+v left unexpected entries in KeyDir: %v", tc.profile, entries)
			}
		})
	}
}

// TestAdoptProfileUseCase_Plan_RequiresBackupFalse confirms CreateMarker.RequiresBackup() stays
// false through this use case's own Plan — a pure creation, nothing pre-existing is overwritten.
func TestAdoptProfileUseCase_Plan_RequiresBackupFalse(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := (AdoptProfileUseCase{}).Plan(AdoptProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, c := range plan.Changes {
		if c.RequiresBackup() {
			t.Errorf("Change %T.RequiresBackup() = true, want false for adopt profile", c)
		}
	}
}

// TestAdoptProfileUseCase_RoundTripsThroughClassifier applies the Plan through the real Applier
// and confirms the directory round-trips through scan.Profiles' own classifier (M1's read path) as
// Managed, not merely "the file exists" — the exit criterion's own stated bar.
func TestAdoptProfileUseCase_RoundTripsThroughClassifier(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(profileDir, "id_ed25519"), "key bytes")

	plan, err := (AdoptProfileUseCase{}).Plan(AdoptProfileRequest{KeyDir: dir, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, c := range plan.Changes {
		if err := c.Apply(fswrite.New()); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}

	candidates, err := scan.Profiles(dir)
	if err != nil {
		t.Fatalf("scan.Profiles: %v", err)
	}
	var found bool
	for _, c := range candidates {
		if c.Dir == profileDir {
			found = true
			if !c.Managed {
				t.Errorf("profile %s: Managed = false after adopt profile, want true", profileDir)
			}
		}
	}
	if !found {
		t.Fatalf("scan.Profiles did not report %s as a candidate at all", profileDir)
	}
}
