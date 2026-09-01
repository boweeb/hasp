// Package scan walks the filesystem into raw facts, the first step of the derivation pipeline
// (docs/tdd.md §5): candidate key files, profile directories, and the host-group/Include tree.
// Classification — is a candidate file actually a key, is a directory actually a managed
// profile — belongs to keyfile and the app derivation pipeline, not here.
package scan

// backupDirName is T8's own name for hasp's backup store (~/.ssh/.hasp-backups/), duplicated here
// (rather than imported from internal/adapter/backup, which this package must not depend on —
// scan sits below app in the dependency order, and backup is consumed at app's own level) so both
// Keys and Profiles can exclude it from their walks. Without this, a snapshot the Applier writes
// mid-Plan — a byte-for-byte copy of whatever key material it just backed up — would itself be
// discovered as a new key candidate (and its directory as a phantom unmanaged profile candidate)
// on the very next scan, silently duplicating an entry that was never meant to be user-visible.
// This was a latent gap since M1: nothing before M2's adopt/release actually populated
// .hasp-backups/ during a scan-adjacent test, so it never surfaced until this phase's Change
// kinds started snapshotting real key material inside a real key directory.
const backupDirName = ".hasp-backups"
