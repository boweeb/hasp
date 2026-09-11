// Package fpscheme is T35's fingerprint scheme registry: every way an external system (AWS's own
// console chief among them) computes a fingerprint for an SSH key, expressed as one Go function
// per scheme rather than one shell pipeline per scheme — T35's own argument against the
// empty-digest hazard a broken pipe silently produces (`d41d8c…`/`da39a3…`, the MD5/SHA-1 of the
// empty string, both shaped exactly like a real fingerprint).
//
// It also holds T36's clue normalization and shape routing (Normalize, CandidateSchemes), so a
// later scheme addition and a later `find`-command change (chunk M3.6.3) stay independent of one
// another: `find` consumes this package's routing rather than reimplementing it.
//
// This package imports crypto/x509, crypto/sha1, crypto/md5, and golang.org/x/crypto/ssh —
// exactly the boundary tdd.md §3's layering rules push adapter code like this into
// internal/adapter/*, out of internal/domain, which may import nothing beyond the standard
// library (T13). No scheme here ever opens a socket or execs a subprocess — layering_test.go's
// TestNoNetworkGuard checks this package's own direct imports mechanically rather than leaving
// it to review discipline (T35, T47).
package fpscheme
