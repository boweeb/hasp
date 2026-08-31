package app

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/backup"
	"github.com/boweeb/hasp/internal/adapter/fswrite"
	"github.com/boweeb/hasp/internal/domain"
)

// snapshotDir returns path -> content digest for every entry under root, structural changes
// (added/removed entries, symlink-vs-file) included — the mechanical proof behind roadmap.md §4
// exit criterion 1: "adopt a key, then release it, and the key directory is byte-identical to
// where it started." Mirrors internal/cli/writeguard_test.go's snapshotTree, reimplemented here
// (not imported: internal/cli imports internal/app, so the dependency can't run the other way)
// for internal/app-level testing exactly as this phase's own instructions call for.
func snapshotDir(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}

		info, lstatErr := os.Lstat(path)
		if lstatErr != nil {
			return lstatErr
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			target, linkErr := os.Readlink(path)
			if linkErr != nil {
				return linkErr
			}
			snap[rel] = "symlink:" + target
			return nil
		}
		if d.IsDir() {
			snap[rel] = "dir:" + info.Mode().Perm().String()
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(b)
		snap[rel] = "file:" + info.Mode().Perm().String() + ":" + hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotDir(%s): %v", root, err)
	}
	return snap
}

func assertSnapshotsEqual(t *testing.T, before, after map[string]string) {
	t.Helper()
	for path, sum := range before {
		got, ok := after[path]
		if !ok {
			t.Errorf("round-trip guard: %s was removed", path)
			continue
		}
		if got != sum {
			t.Errorf("round-trip guard: %s changed: before=%s after=%s", path, sum, got)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			t.Errorf("round-trip guard: %s was created", path)
		}
	}
}

// TestAdoptThenRelease_ExitCriterion1_TreeIsByteIdentical is roadmap.md §4 exit criterion 1,
// verified mechanically rather than asserted: adopt a key into a managed profile, release it back
// out, and confirm the entire key directory tree — every file's bytes and mode, every directory,
// every symlink — is byte-for-byte identical to a snapshot taken before adopt ran. The two-way
// door proven, not asserted.
func TestAdoptThenRelease_ExitCriterion1_TreeIsByteIdentical(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "work", "foobarco")
	mkManagedProfileDir(t, profileDir)
	source := filepath.Join(dir, "id_ed25519")
	writeFile(t, source, "irreplaceable secret key material")
	if err := os.Chmod(source, 0o600); err != nil {
		t.Fatal(err)
	}
	// A .pub sidecar, exactly as `new key` always writes one alongside the private key —
	// adopt/release must round-trip this too (see AdoptKeyUseCase.Plan's own doc comment on why).
	pubSource := source + ".pub"
	writeFile(t, pubSource, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... comment\n")
	if err := os.Chmod(pubSource, 0o644); err != nil {
		t.Fatal(err)
	}

	before := snapshotDir(t, dir)

	applier := &Applier{FS: fswrite.New(), Backups: backup.New(dir)}

	adoptPlan, err := (AdoptKeyUseCase{}).Plan(AdoptKeyRequest{
		KeyDir:  dir,
		Source:  source,
		Profile: domain.ProfilePath{"work", "foobarco"},
	})
	if err != nil {
		t.Fatalf("adopt Plan: %v", err)
	}
	if _, err := applier.Apply(adoptPlan); err != nil {
		t.Fatalf("adopt Apply: %v", err)
	}

	// Confirm the intermediate (adopted) state actually looks like adopt's contract: source is now
	// a symlink, the profile directory holds the real, byte-identical file.
	info, err := os.Lstat(source)
	if err != nil {
		t.Fatalf("Lstat(%s): %v", source, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("source is not a symlink after adopt")
	}
	adoptedReal := filepath.Join(profileDir, "id_ed25519")
	adoptedGot, err := os.ReadFile(adoptedReal)
	if err != nil {
		t.Fatalf("read %s: %v", adoptedReal, err)
	}
	if string(adoptedGot) != "irreplaceable secret key material" {
		t.Fatalf("adopted content = %q, want unchanged", adoptedGot)
	}
	pubInfo, err := os.Lstat(pubSource)
	if err != nil {
		t.Fatalf("Lstat(%s): %v", pubSource, err)
	}
	if pubInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatal("pub sidecar is not a symlink after adopt")
	}
	adoptedPub := adoptedReal + ".pub"
	if _, err := os.ReadFile(adoptedPub); err != nil {
		t.Fatalf("read %s: %v", adoptedPub, err)
	}

	releasePlan, err := (ReleaseKeyUseCase{}).Plan(ReleaseKeyRequest{
		KeyDir: dir,
		Alias:  source,
		Source: adoptedReal,
	})
	if err != nil {
		t.Fatalf("release Plan: %v", err)
	}
	if _, err := applier.Apply(releasePlan); err != nil {
		t.Fatalf("release Apply: %v", err)
	}

	// The backup store itself is not part of the "byte-identical" comparison — hasp is expected to
	// have left backups behind (P4); only the original key-material layout must round-trip.
	backupsDir := filepath.Join(dir, ".hasp-backups")
	if err := os.RemoveAll(backupsDir); err != nil {
		t.Fatalf("remove backups dir for comparison: %v", err)
	}

	after := snapshotDir(t, dir)
	assertSnapshotsEqual(t, before, after)
}
