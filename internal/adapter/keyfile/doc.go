// Package keyfile wraps golang.org/x/crypto/ssh to parse, classify, fingerprint, generate, and
// marshal key files, entirely in pure Go — no subprocess, no cgo (T1). The keyfile adapter
// itself is M1 scope; this package carries only its fixture self-check (fixtures_test.go) in M0.
package keyfile
