// Package fswrite is the concrete implementation of internal/app's WriteFS port: atomic writes
// (temp file + fsync + os.Rename + fsync the containing directory), symlink-through resolution
// for a target that is itself a symlink, mode preservation across a write, and D4's
// copy-verify-unlink move primitive (docs/tdd.md §11 "Write mechanics" 1-4). This package never
// imports internal/app — its exported type satisfies app.WriteFS structurally, matching
// docs/tdd.md §3's ports-at-point-of-consumption convention.
package fswrite
