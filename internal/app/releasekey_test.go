package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/backup"
	"github.com/boweeb/hasp/internal/adapter/fswrite"
)

func TestReleaseKeyUseCase_Plan_RejectsEmptyRequest(t *testing.T) {
	uc := ReleaseKeyUseCase{}
	cases := []ReleaseKeyRequest{
		{Alias: "/a", Source: "/b"},    // no KeyDir
		{KeyDir: "/tmp", Source: "/b"}, // no Alias
		{KeyDir: "/tmp", Alias: "/a"},  // no Source
	}
	for i, req := range cases {
		if _, err := uc.Plan(req); !errors.Is(err, ErrUsage) {
			t.Errorf("case %d: err = %v, want ErrUsage", i, err)
		}
	}
}

// TestReleaseKeyUseCase_Plan_TwoChangesInOrder confirms the ordering §4 requires: the
// alias-replacing restore first (least destructive to interrupt), the redundant-copy Remove last
// (T20's "least recoverable step last" rule).
func TestReleaseKeyUseCase_Plan_TwoChangesInOrder(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	mkManagedProfileDir(t, profileDir)
	source := filepath.Join(profileDir, "id_ed25519")
	writeFile(t, source, "key bytes")
	alias := filepath.Join(dir, "id_ed25519")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}

	uc := ReleaseKeyUseCase{}
	plan, err := uc.Plan(ReleaseKeyRequest{KeyDir: dir, Alias: alias, Source: source})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2", len(plan.Changes))
	}
	restore, ok := plan.Changes[0].(ReplaceSymlinkWithFile)
	if !ok {
		t.Fatalf("Changes[0] = %T, want ReplaceSymlinkWithFile", plan.Changes[0])
	}
	if restore.Alias != alias || restore.Source != source {
		t.Errorf("Changes[0] = %+v, want Alias=%s Source=%s", restore, alias, source)
	}
	remove, ok := plan.Changes[1].(Remove)
	if !ok {
		t.Fatalf("Changes[1] = %T, want Remove", plan.Changes[1])
	}
	if remove.Path != source {
		t.Errorf("Changes[1].Path = %s, want %s", remove.Path, source)
	}
}

// TestReleaseKeyUseCase_Plan_IncludesPubSidecarWhenPresent is the release-side mirror of
// AdoptKeyUseCase's own .pub-sidecar test: releasing an adopted key with a .pub sidecar must also
// restore the sidecar, or a re-adopted key would come back without one.
func TestReleaseKeyUseCase_Plan_IncludesPubSidecarWhenPresent(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	mkManagedProfileDir(t, profileDir)
	source := filepath.Join(profileDir, "id_ed25519")
	writeFile(t, source, "key bytes")
	writeFile(t, source+".pub", "ssh-ed25519 AAAA... comment\n")
	alias := filepath.Join(dir, "id_ed25519")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source+".pub", alias+".pub"); err != nil {
		t.Fatal(err)
	}

	uc := ReleaseKeyUseCase{}
	plan, err := uc.Plan(ReleaseKeyRequest{KeyDir: dir, Alias: alias, Source: source})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 4 {
		t.Fatalf("len(Changes) = %d, want 4 (private restore, pub restore, private remove, pub remove)", len(plan.Changes))
	}
	pubRestore, ok := plan.Changes[1].(ReplaceSymlinkWithFile)
	if !ok {
		t.Fatalf("Changes[1] = %T, want ReplaceSymlinkWithFile", plan.Changes[1])
	}
	if pubRestore.Alias != alias+".pub" || pubRestore.Source != source+".pub" {
		t.Errorf("Changes[1] = %+v, want Alias=%s Source=%s", pubRestore, alias+".pub", source+".pub")
	}
	pubRemove, ok := plan.Changes[3].(Remove)
	if !ok {
		t.Fatalf("Changes[3] = %T, want Remove", plan.Changes[3])
	}
	if pubRemove.Path != source+".pub" {
		t.Errorf("Changes[3].Path = %s, want %s", pubRemove.Path, source+".pub")
	}
}

func TestReleaseKeyUseCase_Plan_RequiresManagedSourceProfile(t *testing.T) {
	dir := t.TempDir()
	unmanagedDir := filepath.Join(dir, "work")
	if err := os.MkdirAll(unmanagedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(unmanagedDir, "id_ed25519")
	writeFile(t, source, "key bytes")
	alias := filepath.Join(dir, "id_ed25519")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}

	uc := ReleaseKeyUseCase{}
	_, err := uc.Plan(ReleaseKeyRequest{KeyDir: dir, Alias: alias, Source: source})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan from an unmanaged directory: err = %v, want ErrUsage", err)
	}
}

func TestReleaseKeyUseCase_Plan_RejectsPathsOutsideKeyDir(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	profileDir := filepath.Join(dir, "work")
	mkManagedProfileDir(t, profileDir)
	source := filepath.Join(profileDir, "id_ed25519")
	writeFile(t, source, "key bytes")
	outsideAlias := filepath.Join(outside, "id_ed25519")

	uc := ReleaseKeyUseCase{}
	_, err := uc.Plan(ReleaseKeyRequest{KeyDir: dir, Alias: outsideAlias, Source: source})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// TestReleaseKeyUseCase_ExitCriterion5Mirror_FailureDuringRestoreLeavesAdoptedStateWorking
// interrupts release at its first step (an unwritable alias directory) and asserts the key
// remains fully discoverable through its original, adopted layout — release's own mirror of exit
// criterion 5.
func TestReleaseKeyUseCase_ExitCriterion5Mirror_FailureDuringRestoreLeavesAdoptedStateWorking(t *testing.T) {
	root := t.TempDir()
	profileDir := filepath.Join(root, "work")
	mkManagedProfileDir(t, profileDir)
	source := filepath.Join(profileDir, "id_ed25519")
	original := "irreplaceable secret"
	writeFile(t, source, original)
	alias := filepath.Join(root, "id_ed25519")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}

	plan, err := (ReleaseKeyUseCase{}).Plan(ReleaseKeyRequest{KeyDir: root, Alias: alias, Source: source})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// Make root itself unwritable so nothing further can be written there — neither the backup
	// snapshot nor ReplaceSymlinkWithFile's own scratch temp file (created beside alias, i.e. in
	// root) — so Apply fails before touching alias at all.
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(root)}
	if _, err := applier.Apply(plan); err == nil {
		t.Fatal("Apply succeeded despite an unwritable alias directory, want an error")
	}

	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}

	// alias is untouched: still a symlink to source, still resolving to the original bytes — the
	// pre-release (adopted) state, fully working.
	info, statErr := os.Lstat(alias)
	if statErr != nil {
		t.Fatalf("alias %s missing after a failed release: %v", alias, statErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("alias is no longer a symlink after a failed release")
	}
	got, err := os.ReadFile(alias)
	if err != nil {
		t.Fatalf("read through alias %s: %v", alias, err)
	}
	if string(got) != original {
		t.Errorf("content through alias = %q, want unchanged %q", got, original)
	}
	sourceGot, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("source %s no longer readable: %v", source, err)
	}
	if string(sourceGot) != original {
		t.Errorf("source content = %q, want unchanged %q", sourceGot, original)
	}
}

// TestReleaseKeyUseCase_FailureDuringRemoveLeavesAliasDiscoverable interrupts release at its
// second, least-recoverable step (Remove of the redundant profile-directory copy) and asserts the
// key remains fully discoverable at the released, top-level path — the Remove failing leaves a
// harmless duplicate, never a broken or undiscoverable state.
func TestReleaseKeyUseCase_FailureDuringRemoveLeavesAliasDiscoverable(t *testing.T) {
	root := t.TempDir()
	profileDir := filepath.Join(root, "work")
	mkManagedProfileDir(t, profileDir)
	source := filepath.Join(profileDir, "id_ed25519")
	original := "irreplaceable secret"
	writeFile(t, source, original)
	alias := filepath.Join(root, "id_ed25519")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}

	plan, err := (ReleaseKeyUseCase{}).Plan(ReleaseKeyRequest{KeyDir: root, Alias: alias, Source: source})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// Make profileDir unwritable only after the restore step has a writable root to work with —
	// Remove needs write permission on profileDir itself to unlink source from it.
	if err := os.Chmod(profileDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(profileDir, 0o700) })

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(root)}
	if _, err := applier.Apply(plan); err == nil {
		t.Fatal("Apply succeeded despite an unwritable profile directory, want an error at the Remove step")
	}

	// alias now holds a real, complete, independent copy — fully discoverable — even though
	// Remove(source) failed.
	info, statErr := os.Lstat(alias)
	if statErr != nil {
		t.Fatalf("alias %s missing after a partial release: %v", alias, statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("alias is still a symlink; the restore step should have already succeeded")
	}
	got, err := os.ReadFile(alias)
	if err != nil {
		t.Fatalf("read alias %s: %v", alias, err)
	}
	if string(got) != original {
		t.Errorf("alias content = %q, want %q", got, original)
	}

	if err := os.Chmod(profileDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// source is the redundant, not-yet-removed duplicate — still present, still valid.
	sourceGot, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("source %s missing: %v", source, err)
	}
	if string(sourceGot) != original {
		t.Errorf("source content = %q, want %q", sourceGot, original)
	}
}
