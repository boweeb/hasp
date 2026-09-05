//go:build mage

// Package main holds hasp's Mage targets (docs/tdd.md §17, T32): every build/test/lint action is
// a Mage target here, and a platform workflow file's only job is to invoke one of them. The
// `//go:build mage` tag keeps this package out of `go build ./...` and `go vet ./...` entirely,
// and out of cmd/hasp's own dependency graph — see cmd/hasp/layering_test.go.
package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// golangciLintVersion pins golangci-lint to an exact version (T32: "pinned, not floating"),
// verified against the module proxy as the latest v2 release at the time this was written.
const golangciLintVersion = "v2.13.2"

// Build compiles the real hasp binary.
func Build() error {
	return sh.RunV("go", "build", "-o", "bin/hasp", "./cmd/hasp")
}

// Test runs the full test suite.
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// Vet runs go vet across the module.
func Vet() error {
	return sh.RunV("go", "vet", "./...")
}

// Lint runs golangci-lint pinned to an exact version, per T32, rather than an unpinned `@latest`
// that turns an unrelated upstream release into a red build on a commit that changed nothing.
func Lint() error {
	return sh.RunV("go", "run",
		"github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"+golangciLintVersion,
		"run", "./...")
}

// Cross is a compile-only check that the module still builds for both darwin architectures
// GoReleaser targets (docs/tdd.md §17: "Darwin compiles in CI"). It intentionally never passes
// `-o`, so no artifacts are produced or kept — this only proves the code compiles there.
func Cross() error {
	for _, goarch := range []string{"amd64", "arm64"} {
		env := map[string]string{"GOOS": "darwin", "GOARCH": goarch}
		if err := sh.RunWithV(env, "go", "build", "./..."); err != nil {
			return err
		}
	}
	return nil
}

// Fuzz runs the existing SSH-config round-trip fuzz test as a short smoke fuzz. 10s is a
// CI-smoke duration meant to catch obvious regressions quickly, not exhaustive fuzzing — it is
// deliberately excluded from CI's blocking target set (see CI below).
func Fuzz() error {
	return sh.RunV("go", "test",
		"-run=^$",
		"-fuzz=FuzzSSHConfigRoundTrip",
		"-fuzztime=10s",
		"./internal/adapter/sshconfig")
}

// Fixtures regenerates the key fixtures under testdata/keys/ that gate M1's read path; see
// tools/genfixtures's own package doc for what it produces.
func Fixtures() error {
	return sh.RunV("go", "run", "./tools/genfixtures")
}

// CI is the aggregate target a workflow shim invokes (T32's thin-shim rule). It deliberately
// excludes Fuzz — fuzzing is a smoke/optional target, not blocking on every CI run.
func CI() {
	mg.Deps(Build, Vet, Lint, Test, Cross, Fixtures)
}
