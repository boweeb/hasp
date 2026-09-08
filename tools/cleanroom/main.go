// Command cleanroom is the mechanical proof behind roadmap.md §5.5 exit criterion 6: install hasp
// exactly the way README.md's install steps say ("go install
// github.com/boweeb/hasp/cmd/hasp@latest", or the tag-pinned equivalent a release build uses),
// then run it — `list key`, `adopt key`, `release key` — against a synthetic ~/.ssh fixture this
// tool builds, and confirm the fixture is byte-identical (see internal/snapshot) to where it
// started once `adopt` and `release` have both run. It is a standalone dev-time tool, run via `go
// run ./tools/cleanroom` (or the CleanRoom Mage target), matching this repo's tools/docscheck,
// tools/gendocs, and tools/genfixtures convention: package main under tools/, never imported by
// cmd/hasp, exec'ing the hasp binary under test as a real subprocess rather than calling into
// internal/app or internal/cli directly — this is deliberately testing the installed binary a
// stranger would get, not hasp's own internal API.
//
// This is a hard pass/fail script, not a judgement call, per roadmap.md §5.5 exit criterion 6's
// own stated intent: any subprocess exiting non-zero, or a non-empty post-round-trip snapshot
// diff, is reported to stderr and the tool exits 1; a clean run prints a short confirmation to
// stdout and exits 0.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/boweeb/hasp/internal/cli/render"
	"github.com/boweeb/hasp/internal/snapshot"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "cleanroom:", err)
		os.Exit(1)
	}
	fmt.Println("cleanroom: OK — adopt/release round-tripped a synthetic ~/.ssh fixture byte-identical (roadmap.md §5.5 exit criterion 6)")
}

func run() error {
	haspBinFlag := flag.String("hasp-bin", "", "path to the hasp binary under test (default: resolve \"hasp\" via PATH)")
	homeFlag := flag.String("home", "", "fixture directory to build and test against (default: a fresh temp directory)")
	keepFlag := flag.Bool("keep", false, "don't remove a self-created fixture temp directory on exit (for post-mortem inspection); has no effect with -home, which is always the caller's to clean up")
	flag.Parse()

	bin, err := resolveHaspBin(*haspBinFlag)
	if err != nil {
		return err
	}

	root, err := resolveHome(*homeFlag)
	if err != nil {
		return err
	}
	// A self-created fixture directory holds generated private key material (buildFixture,
	// below) and must not be left behind indefinitely across repeated local `go run mage.go
	// cleanroom` invocations — -home is the one case where root belongs to the caller, not us, so
	// it's left untouched regardless of -keep.
	if *homeFlag == "" && !*keepFlag {
		defer func() {
			if rmErr := os.RemoveAll(root); rmErr != nil {
				fmt.Fprintf(os.Stderr, "cleanroom: remove fixture temp dir %s: %v\n", root, rmErr)
			}
		}()
	}

	fx, err := buildFixture(root)
	if err != nil {
		return fmt.Errorf("build fixture: %w", err)
	}

	before, err := snapshot.Snapshot(root)
	if err != nil {
		return fmt.Errorf("snapshot fixture before adopt/release: %w", err)
	}

	if err := verifyInventory(bin, fx); err != nil {
		return err
	}

	if err := runHaspOK(bin, "adopt", "key", fx.unmanagedName,
		"--profile", fx.destProfile, "--key-dir", root, "--json", "--yes"); err != nil {
		return err
	}

	if err := runHaspOK(bin, "release", "key", fx.unmanagedName,
		"--key-dir", root, "--json", "--yes"); err != nil {
		return err
	}

	// hasp's own backup store (T8, ~/.ssh/.hasp-backups/) is expected to gain content across
	// adopt/release — P4's "back up before every destructive write" — and is deliberately
	// excluded from the byte-identical comparison, mirroring
	// internal/app/adopt_release_roundtrip_test.go's own exclusion of it exactly (remove before
	// the final snapshot, rather than teaching internal/snapshot an exclusion parameter neither
	// existing caller needs).
	if err := os.RemoveAll(filepath.Join(root, backupDirName)); err != nil {
		return fmt.Errorf("remove %s before final snapshot: %w", backupDirName, err)
	}

	after, err := snapshot.Snapshot(root)
	if err != nil {
		return fmt.Errorf("snapshot fixture after adopt/release: %w", err)
	}

	mismatches := snapshot.Diff(before, after)
	if len(mismatches) > 0 {
		fmt.Fprintln(os.Stderr, "cleanroom: fixture is not byte-identical after adopt/release:")
		for _, m := range mismatches {
			fmt.Fprintln(os.Stderr, "  -", m)
		}
		return fmt.Errorf("%d mismatch(es) — roadmap.md §5.5 exit criterion 6 fails", len(mismatches))
	}

	return nil
}

// resolveHaspBin returns explicit if non-empty, else the "hasp" found on PATH — the same
// resolution CleanRoom (magefiles/magefile.go) leaves to this tool rather than duplicating.
func resolveHaspBin(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	found, err := exec.LookPath("hasp")
	if err != nil {
		return "", fmt.Errorf("resolve hasp binary: -hasp-bin was not given and %w", err)
	}
	return found, nil
}

func resolveHome(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	dir, err := os.MkdirTemp("", "hasp-cleanroom-*")
	if err != nil {
		return "", fmt.Errorf("create fixture temp dir: %w", err)
	}
	return dir, nil
}

// keyListItem is the subset of domain.Key's JSON shape (internal/domain/key.go) this tool needs
// to assert against: just the "name" field. domain.Key itself isn't reused directly for
// unmarshaling here — its Identity field is domain.KeyIdentity, an interface hasp only ever
// marshals (json.Marshaler), never unmarshals, so json.Unmarshal into a bare []domain.Key fails
// on that field every time; render.Envelope (the versioned outer envelope) is still reused as
// designed, only its Data payload gets this narrower, unmarshal-safe view.
type keyListItem struct {
	Name string `json:"name"`
}

// verifyInventory runs `list key --json` against the fixture and asserts the envelope's data
// names both keys buildFixture created — roadmap.md §5.5 exit criterion 6's "hasp list key
// returns an accurate inventory of a synthetic ~/.ssh fixture" half, independent of the
// adopt/release round trip that follows it.
func verifyInventory(bin string, fx fixture) error {
	stdout, err := runHasp(bin, "list", "key", "--key-dir", fx.root, "--json")
	if err != nil {
		return err
	}

	var env render.Envelope
	if err := json.Unmarshal(stdout, &env); err != nil {
		return fmt.Errorf("list key --json: unmarshal envelope: %w\noutput:\n%s", err, stdout)
	}
	var keys []keyListItem
	if err := json.Unmarshal(env.Data, &keys); err != nil {
		return fmt.Errorf("list key --json: unmarshal envelope data: %w\ndata:\n%s", err, env.Data)
	}

	names := map[string]bool{}
	for _, k := range keys {
		names[k.Name] = true
	}
	if !names[fx.adoptedName] {
		return fmt.Errorf("list key --json: fixture's already-adopted key %q not found in %d reported keys", fx.adoptedName, len(keys))
	}
	if !names[fx.unmanagedName] {
		return fmt.Errorf("list key --json: fixture's unmanaged key %q not found in %d reported keys", fx.unmanagedName, len(keys))
	}
	return nil
}

// runHaspOK runs the hasp subprocess and fails with the command, its stderr, and its exit error
// if it exits non-zero — a non-zero exit from adopt/release is itself a failure this tool must
// report clearly, not something only the final snapshot diff would eventually surface.
func runHaspOK(bin string, args ...string) error {
	_, err := runHasp(bin, args...)
	return err
}

// runHasp execs bin with args, returning stdout on success. On a non-zero exit it returns an
// error naming the exact command and both stdout and stderr the subprocess produced.
func runHasp(bin string, args ...string) ([]byte, error) {
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %v failed: %w\nstdout:\n%s\nstderr:\n%s",
			bin, args, err, stdout.String(), stderr.String())
	}
	return stdout.Bytes(), nil
}
