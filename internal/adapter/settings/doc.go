// Package settings reads (and never writes) ~/.config/hasp/settings.toml (docs/tdd.md §8, T7).
// The v1 keyset is exactly [new_key] default_passphrase_mode — the admission rule (D16) permits
// nothing else, and a test enforces the exact field set of Settings by name so a PR adding a
// field changes that test visibly.
package settings
