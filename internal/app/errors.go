package app

import "errors"

// ErrFindings signals that check surfaced issues — not a hasp failure (T14). check is the only
// use case ever permitted to return it; cmd/hasp's main() maps it to exit code 1.
var ErrFindings = errors.New("check found issues")

// ErrUsage signals bad flags/arguments, or (from M2 onward) an unresolvable non-interactive
// write precondition (T6). cmd/hasp's main() maps it to exit code 2.
var ErrUsage = errors.New("usage error")
