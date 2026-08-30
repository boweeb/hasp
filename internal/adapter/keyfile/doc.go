// Package keyfile wraps golang.org/x/crypto/ssh to parse, classify, and fingerprint key files,
// entirely in pure Go — no subprocess, no cgo (T1). Generation and marshaling (new key) are M2
// scope. fixtures_test.go's direct-library self-check from M0 remains alongside as the
// independent proof of x/crypto/ssh's behavior this package's own tests are cross-checked
// against.
package keyfile
