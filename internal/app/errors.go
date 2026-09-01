package app

import "errors"

// ErrFindings signals that check surfaced issues — not a hasp failure (T14). check is the only
// use case ever permitted to return it; cmd/hasp's main() maps it to exit code 1.
var ErrFindings = errors.New("check found issues")

// ErrUsage signals bad flags/arguments, or (from M2 onward) an unresolvable non-interactive
// write precondition (T6). cmd/hasp's main() maps it to exit code 2.
var ErrUsage = errors.New("usage error")

// ErrPlanStale signals that a Plan's witness re-verification (T30) found the machine changed
// since the Plan was built — Applier.Apply refuses the whole Plan before its first write, and
// nothing is backed up. Unwrapped, this maps to exit code 3 (tdd.md §10) like any other error.
var ErrPlanStale = errors.New("plan is stale: the machine changed since this plan was built")

// ErrKeyFileExists signals that WriteKeyFile.Apply refused to write because its target path
// already exists and AllowOverwrite is false (T22, D4) — hasp never overwrites key material it
// did not just verify it authored in this same Plan.
var ErrKeyFileExists = errors.New("key file already exists")

// ErrMarkerExists signals that CreateMarker.Apply refused to write because a .hasp marker
// already exists at the target directory — hasp never silently replaces an existing marker,
// which may carry a user-written note (D15).
var ErrMarkerExists = errors.New("profile marker already exists")
