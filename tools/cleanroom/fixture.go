package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boweeb/hasp/internal/adapter/keyfile"
)

// markerFileName and markerFileContents mirror internal/app/change_marker.go's own unexported
// constant and internal/app/adoptkey_test.go's mkManagedProfileDir helper exactly (".hasp",
// "# profile marker\n") — verified against that source, not guessed, since a fixture whose marker
// doesn't match what AdoptKeyUseCase.Plan checks for would make `adopt key` fail for a reason
// unrelated to what this tool is actually testing.
const (
	markerFileName     = ".hasp"
	markerFileContents = "# profile marker\n"
)

// backupDirName mirrors internal/adapter/backup's own unexported T8 constant (".hasp-backups").
// Duplicated here as a literal rather than imported, the same way
// internal/app/adopt_release_roundtrip_test.go's own round-trip guard hardcodes it: hasp's backup
// store is expected to gain content during adopt/release (P4) and is deliberately excluded from
// the byte-identical comparison, not part of what this tool proves stayed untouched.
const backupDirName = ".hasp-backups"

// fixture is the on-disk layout buildFixture produces, and the handles main.go needs to drive
// hasp against it: a small, realistic ~/.ssh-shaped directory holding one key already adopted
// into a managed profile (a top-level alias symlink plus the real file inside the profile
// directory, exactly the shape AdoptKeyUseCase.Plan produces) and one still-unmanaged key sitting
// at the top level with no profile membership — so `adopt` has real work to do and `release` has
// something to reverse. Modeled closely on internal/app/adoptkey_test.go's mkManagedProfileDir
// and internal/app/pipeline_test.go's writeFile, per this tool's own design.
type fixture struct {
	root string

	adoptedName   string // basename shared by the pre-adopted key's alias and its real location
	unmanagedName string // basename of the key adopt/release round-trips during this run
	destProfile   string // dotted --profile path adopt key moves unmanagedName into
}

// buildFixture creates a fresh fixture rooted at root, returning the handles the round trip
// needs. root must already exist or be creatable by os.MkdirAll.
func buildFixture(root string) (fixture, error) {
	fx := fixture{
		root:          root,
		adoptedName:   "id_ed25519_adopted",
		unmanagedName: "id_ed25519_unmanaged",
		destProfile:   "personal",
	}

	if err := os.MkdirAll(root, 0o700); err != nil {
		return fixture{}, fmt.Errorf("mkdir %s: %w", root, err)
	}

	if err := writeSSHConfig(root); err != nil {
		return fixture{}, err
	}

	// One key already adopted: real file inside a managed profile directory, plus a top-level
	// alias symlink pointing at it — both the private key and its .pub sidecar, matching what
	// AdoptKeyUseCase.Plan itself produces (internal/app/adoptkey_test.go's
	// TestAdoptKeyUseCase_Plan_IncludesPubSidecarWhenPresent).
	adoptedProfileDir := filepath.Join(root, "work", "adopted")
	if err := mkManagedProfileDir(adoptedProfileDir); err != nil {
		return fixture{}, err
	}
	adoptedReal := filepath.Join(adoptedProfileDir, fx.adoptedName)
	if err := writeGeneratedKey(adoptedReal, "adopted@hasp-cleanroom"); err != nil {
		return fixture{}, err
	}
	adoptedAlias := filepath.Join(root, fx.adoptedName)
	if err := os.Symlink(adoptedReal, adoptedAlias); err != nil {
		return fixture{}, fmt.Errorf("symlink %s -> %s: %w", adoptedAlias, adoptedReal, err)
	}
	if err := os.Symlink(adoptedReal+".pub", adoptedAlias+".pub"); err != nil {
		return fixture{}, fmt.Errorf("symlink %s -> %s: %w", adoptedAlias+".pub", adoptedReal+".pub", err)
	}

	// An empty managed profile directory as adopt's destination for the unmanaged key below —
	// AdoptKeyUseCase.Plan requires the target profile directory to already carry a marker
	// (internal/app/adoptkey_test.go's TestAdoptKeyUseCase_Plan_RequiresManagedTargetProfile).
	if err := mkManagedProfileDir(filepath.Join(root, fx.destProfile)); err != nil {
		return fixture{}, err
	}

	// One unmanaged key, still sitting at the top level with no profile membership — the one
	// adopt/release actually exercises.
	if err := writeGeneratedKey(filepath.Join(root, fx.unmanagedName), "unmanaged@hasp-cleanroom"); err != nil {
		return fixture{}, err
	}

	return fx, nil
}

// mkManagedProfileDir creates dir (including parents) and writes the ".hasp" marker inside it —
// the fixture shape AdoptKeyUseCase.Plan requires of a target profile (D13, D14), mirroring
// internal/app/adoptkey_test.go's helper of the same purpose.
func mkManagedProfileDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	marker := filepath.Join(dir, markerFileName)
	if err := os.WriteFile(marker, []byte(markerFileContents), 0o644); err != nil {
		return fmt.Errorf("write marker %s: %w", marker, err)
	}
	return nil
}

// writeGeneratedKey writes a freshly generated ed25519 keypair (internal/adapter/keyfile.Generate
// — the same pure-Go, no-subprocess mechanism `new key` itself uses, T1) to path and path+".pub",
// so keyfile.Inspect can actually classify it as a key once hasp derives the fixture — arbitrary
// bytes would silently fail keyfile.Inspect and be skipped entirely (internal/app/pipeline.go's
// fail-open "not actually a key" branch), which would make `list key` report the wrong count.
func writeGeneratedKey(path, comment string) error {
	generated, err := keyfile.Generate(keyfile.GenerateOptions{Comment: comment})
	if err != nil {
		return fmt.Errorf("generate key for %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, generated.PrivateKeyPEM, 0o600); err != nil {
		return fmt.Errorf("write private key %s: %w", path, err)
	}
	if err := os.WriteFile(path+".pub", generated.PublicKeyLine, 0o644); err != nil {
		return fmt.Errorf("write public key %s: %w", path+".pub", err)
	}
	return nil
}

// writeSSHConfig writes a minimal ssh_config at root/config (hasp's fixed host-group root
// filename — internal/app/pipeline.go's DeriveOptions.ConfigRoot default, filepath.Join(KeyDir,
// "config")) with one Host block, satisfying the fixture's own requirement of at least one.
func writeSSHConfig(root string) error {
	const body = `# hasp cleanroom fixture (tools/cleanroom) — a minimal, synthetic ssh_config.
Host cleanroom-example
    HostName example.com
    User git
`
	path := filepath.Join(root, "config")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
