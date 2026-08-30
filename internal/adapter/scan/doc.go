// Package scan walks the filesystem into raw facts, the first step of the derivation pipeline
// (docs/tdd.md §5): candidate key files, profile directories, and the host-group/Include tree.
// Classification — is a candidate file actually a key, is a directory actually a managed
// profile — belongs to keyfile and the app derivation pipeline, not here.
package scan
