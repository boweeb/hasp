// Package app holds one use case per verb×noun cell (docs/tdd.md §9) and owns the Plan type
// (docs/tdd.md §4, M2+). For M1's read path, there is no swappable port to inject: D12 makes
// hasp a pure function of a directory, so Derive calls the concrete adapters directly and tests
// exercise it against real fixture trees, exactly as tdd.md §12 describes — no mock filesystem
// is more faithful than a real one for a tool with no other backend to swap in.
package app
