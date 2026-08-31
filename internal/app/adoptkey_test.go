package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/backup"
	"github.com/boweeb/hasp/internal/adapter/fswrite"
	"github.com/boweeb/hasp/internal/domain"
)

// mkManagedProfileDir creates dir (including parents) and writes a .hasp marker inside it — the
// fixture shape AdoptKeyUseCase.Plan requires of a target profile (D13, D14).
func mkManagedProfileDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, markerFileName), []byte("# profile marker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAdoptKeyUseCase_Plan_RejectsEmptyRequest(t *testing.T) {
	uc := AdoptKeyUseCase{}
	cases := []AdoptKeyRequest{
		{Source: "/tmp/id_ed25519", Profile: domain.ProfilePath{"work"}}, // no KeyDir
		{KeyDir: "/tmp", Profile: domain.ProfilePath{"work"}},            // no Source
		{KeyDir: "/tmp", Source: "/tmp/id_ed25519"},                      // no Profile
	}
	for i, req := range cases {
		if _, err := uc.Plan(req); !errors.Is(err, ErrUsage) {
			t.Errorf("case %d: err = %v, want ErrUsage", i, err)
		}
	}
}

// TestAdoptKeyUseCase_Plan_SingleChange confirms §4/§11's collapse to one Change holds: adopt's
// Plan is exactly one ReplaceWithSymlink, not the naive two-Change [MoveFile, CreateSymlink] shape
// tdd.md §4 walks through and rejects.
func TestAdoptKeyUseCase_Plan_SingleChange(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work", "foobarco")
	mkManagedProfileDir(t, profileDir)
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")

	uc := AdoptKeyUseCase{}
	plan, err := uc.Plan(AdoptKeyRequest{KeyDir: dir, Source: source, Profile: domain.ProfilePath{"work", "foobarco"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	c, ok := plan.Changes[0].(ReplaceWithSymlink)
	if !ok {
		t.Fatalf("Changes[0] = %T, want ReplaceWithSymlink", plan.Changes[0])
	}
	if c.From != source {
		t.Errorf("From = %s, want %s", c.From, source)
	}
	want := filepath.Join(profileDir, "id_ed25519")
	if c.To != want {
		t.Errorf("To = %s, want %s", c.To, want)
	}
}

// TestAdoptKeyUseCase_Plan_IncludesPubSidecarWhenPresent guards the regression found while
// proving this phase's own CLI round trip: a `new key`-generated key always has a .pub sidecar,
// and keyfile.Inspect derives HasPublicHalf/Comment from a .pub sitting next to the key's
// *resolved* (real) location — so adopt must relocate the sidecar too, or those facts silently go
// wrong the moment the key is adopted.
func TestAdoptKeyUseCase_Plan_IncludesPubSidecarWhenPresent(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	mkManagedProfileDir(t, profileDir)
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")
	writeFile(t, source+".pub", "ssh-ed25519 AAAA... comment\n")

	uc := AdoptKeyUseCase{}
	plan, err := uc.Plan(AdoptKeyRequest{KeyDir: dir, Source: source, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2 (private key + .pub sidecar)", len(plan.Changes))
	}
	pub, ok := plan.Changes[1].(ReplaceWithSymlink)
	if !ok {
		t.Fatalf("Changes[1] = %T, want ReplaceWithSymlink", plan.Changes[1])
	}
	if pub.From != source+".pub" {
		t.Errorf("Changes[1].From = %s, want %s", pub.From, source+".pub")
	}
	wantTo := filepath.Join(profileDir, "id_ed25519.pub")
	if pub.To != wantTo {
		t.Errorf("Changes[1].To = %s, want %s", pub.To, wantTo)
	}
}

func TestAdoptKeyUseCase_Plan_RequiresManagedTargetProfile(t *testing.T) {
	dir := t.TempDir()
	unmanagedDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(unmanagedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "key bytes")

	uc := AdoptKeyUseCase{}
	_, err := uc.Plan(AdoptKeyRequest{KeyDir: dir, Source: source, Profile: domain.ProfilePath{"work"}})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan into an unmanaged directory: err = %v, want ErrUsage", err)
	}
}

// TestAdoptKeyUseCase_Plan_RejectsPathTraversal mirrors newkey.go's own regression test: a
// malicious --profile value must be rejected before any Change is planned, independent of
// whatever internal/cli does or doesn't validate.
func TestAdoptKeyUseCase_Plan_RejectsPathTraversal(t *testing.T) {
	uc := AdoptKeyUseCase{}

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
			source := filepath.Join(dir, "id_ed25519")
			writeFile(t, source, "key bytes")

			_, err := uc.Plan(AdoptKeyRequest{KeyDir: dir, Source: source, Profile: tc.profile})
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("Plan error = %v, want ErrUsage", err)
			}

			// Nothing beyond the fixture's own source file may exist under dir.
			entries, readErr := os.ReadDir(dir)
			if readErr != nil {
				t.Fatalf("ReadDir(%s): %v", dir, readErr)
			}
			if len(entries) != 1 || entries[0].Name() != "id_ed25519" {
				t.Errorf("Plan for profile %+v left unexpected entries in KeyDir: %v", tc.profile, entries)
			}
		})
	}
}

// TestAdoptKeyUseCase_Plan_RejectsSourceOutsideKeyDir is the source-side mirror of the
// traversal guard: a Source path outside req.KeyDir must be rejected before anything is planned,
// regardless of how it got there.
func TestAdoptKeyUseCase_Plan_RejectsSourceOutsideKeyDir(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	source := filepath.Join(outside, "id_ed25519")
	writeFile(t, source, "key bytes")

	uc := AdoptKeyUseCase{}
	_, err := uc.Plan(AdoptKeyRequest{KeyDir: dir, Source: source, Profile: domain.ProfilePath{"work"}})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// TestAdoptKeyUseCase_ExitCriterion5_InjectedFailureLeavesSourceReadable is roadmap.md §4 exit
// criterion 5, run through the real Applier and BackupStore rather than the raw fswrite primitive
// directly: interrupt adopt (an unwritable destination profile directory) and assert a complete,
// readable key remains at its original path.
func TestAdoptKeyUseCase_ExitCriterion5_InjectedFailureLeavesSourceReadable(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "id_ed25519")
	original := "irreplaceable secret"
	writeFile(t, source, original)

	profileDir := filepath.Join(dir, "work")
	mkManagedProfileDir(t, profileDir)
	if err := os.Chmod(profileDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(profileDir, 0o700) })

	uc := AdoptKeyUseCase{}
	plan, err := uc.Plan(AdoptKeyRequest{KeyDir: dir, Source: source, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
	if _, err := applier.Apply(plan); err == nil {
		t.Fatal("Apply succeeded against an unwritable profile directory, want an error")
	}

	got, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("source %s no longer readable after a failed adopt: %v", source, err)
	}
	if string(got) != original {
		t.Errorf("source content = %q, want unchanged %q", got, original)
	}
	if info, statErr := os.Lstat(source); statErr != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Error("source was turned into a symlink despite the failed adopt")
	}
}

// TestAdoptKeyUseCase_ExitCriterion5_FailurePastVerification interrupts adopt precisely between
// the copy-and-verify step succeeding and the terminal replace running — the specific window §11
// point 4's own text calls out — by making Source's own directory unwritable (blocking the
// scratch symlink the replace step needs to create there) while leaving KeyDir's root, and the
// destination profile directory, writable — so BackupStore.Snapshot and the copy step both
// succeed and only the terminal replace fails. Asserts source still holds the complete, unchanged
// original file afterward.
func TestAdoptKeyUseCase_ExitCriterion5_FailurePastVerification(t *testing.T) {
	root := t.TempDir() // KeyDir: writable, holds .hasp-backups and the profile dir
	sourceDir := filepath.Join(root, "unmanaged")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDir, "id_ed25519")
	original := "irreplaceable secret"
	writeFile(t, source, original)

	profileDir := filepath.Join(root, "work")
	mkManagedProfileDir(t, profileDir)

	if err := os.Chmod(sourceDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sourceDir, 0o700) })

	plan, err := (AdoptKeyUseCase{}).Plan(AdoptKeyRequest{KeyDir: root, Source: source, Profile: domain.ProfilePath{"work"}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(root)}
	if _, err := applier.Apply(plan); err == nil {
		t.Fatal("Apply succeeded despite an unwritable source directory, want an error")
	}

	got, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("source %s no longer readable after a failed adopt: %v", source, err)
	}
	if string(got) != original {
		t.Errorf("source content = %q, want unchanged %q", got, original)
	}
	if info, statErr := os.Lstat(source); statErr != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Error("source was turned into a symlink despite the failed replace step")
	}

	// The destination directory holds the verified copy from the successful copy step — this is
	// §11 point 4's own stated legitimate state: not yet deduplicated into a symlink, which check
	// reports as duplicate-key-confirmed. Recovering from here is manual (remove the redundant
	// destination copy, or finish the swap by hand) — retrying adopt key unmodified fails closed,
	// because newTarget already exists (see ReplaceWithSymlink's own doc comment in plan.go).
	destCopy := filepath.Join(profileDir, "id_ed25519")
	destGot, err := os.ReadFile(destCopy)
	if err != nil {
		t.Fatalf("verified destination copy %s is missing: %v", destCopy, err)
	}
	if string(destGot) != original {
		t.Errorf("destination copy content = %q, want %q", destGot, original)
	}
}
