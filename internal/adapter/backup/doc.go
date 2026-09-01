// Package backup is the concrete implementation of internal/app's BackupStore port: snapshotting
// a file's current bytes into <key-dir>/.hasp-backups/ before a destructive write, as timestamped
// copies never pruned by hasp (docs/tdd.md §3, §11, T8). The key directory is always injected,
// never defaulted (docs/tdd.md §12) — Store.KeyDir must be set explicitly by whoever constructs
// one. This package never imports internal/app; its exported type satisfies app.BackupStore
// structurally, matching docs/tdd.md §3's ports-at-point-of-consumption convention.
package backup
