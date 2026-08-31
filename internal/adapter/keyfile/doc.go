// Package keyfile wraps golang.org/x/crypto/ssh to parse, classify, fingerprint, generate, and
// marshal key files, entirely in pure Go — no subprocess, no cgo (T1). fixtures_test.go's direct-
// library self-check from M0 remains alongside as the independent proof of x/crypto/ssh's
// behavior this package's own tests are cross-checked against.
package keyfile
