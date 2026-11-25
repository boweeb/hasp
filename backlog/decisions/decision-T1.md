---
id: decision-T1
title: 'Pure-Go SSH key derivation via `x/crypto/ssh`; no `ssh-keygen`, no `libmagic`'
date: '2026-08-28'
status: 'Accepted'
---
### Context

[`design.md` §5.1](../../docs/design.md#51-key) defines the derivation gap: a three-row truth table for
whether a fingerprint is derivable, closed with the requirement that hasp report the undecidable
case as `unknown` rather than prompt for a passphrase (P3) or persist a guess (D12). [D12](decision-D12.md)
requires every fact be recomputed on demand; P8 requires it be cheap enough to do so on every
invocation. The read path must determine fingerprint, algorithm, format, encryption state, and
comment for the full key matrix — ed25519, RSA, ECDSA, in both OpenSSH and legacy PEM framing,
encrypted or not, with or without a `.pub` file — from raw bytes, without ever decrypting
anything.

The predecessor project needed two external dependencies to do this: an `ssh-keygen`
subprocess and `libmagic` (via `python-magic`) for format sniffing — both named in
[`docs/project-assessment-2026-08.md` §4](../../docs/project-assessment-2026-08.md) as toolchain weight,
and the former as a source of `Cannot load private key: incorrect passphrase`-shaped failures
that D14 later used as evidence against ever touching an encrypted private key file at all.

### Decision

The entire read path is pure Go, using `golang.org/x/crypto/ssh`. No subprocess is exec'd, no
cgo dependency is introduced. Confirmed against the installed module (`v0.55.0`) this session:

```go
// x/crypto/ssh/keys.go
type PassphraseMissingError struct {
    // PublicKey will be set if the private key format includes an unencrypted
    // public key along with the encrypted private key.
    PublicKey PublicKey
}

func ParseRawPrivateKey(pemBytes []byte) (interface{}, error)
```

`ParseRawPrivateKey` switches on the PEM block's `Type` field — `"OPENSSH PRIVATE KEY"` versus
`"RSA PRIVATE KEY"` / `"EC PRIVATE KEY"` / `"DSA PRIVATE KEY"` / bare `"PRIVATE KEY"` (PKCS#8) —
which gives `format` for free, unencrypted, no decryption attempted. Whether the key is
encrypted is likewise readable without decrypting it: `Proc-Type: 4,ENCRYPTED` /
`DEK-Info` headers for legacy PEM, the cipher name field inside the OpenSSH wire format for the
modern one.

For the undecidable case, the function returns `*PassphraseMissingError` and the struct's own
`PublicKey` field is the typed discriminator design.md's three-row table needs:

| Case | What `ParseRawPrivateKey` does |
| --- | --- |
| Public half present | Never reaches this path — hasp parses the `.pub` via `ParseAuthorizedKey`, then `FingerprintSHA256` |
| OpenSSH format, encrypted, no `.pub` | Returns `*PassphraseMissingError` with **`.PublicKey` populated** — `parseOpenSSHPrivateKey` reads it from the embedded, unencrypted `w.PubKey` field before it ever touches the ciphertext |
| Legacy PEM, encrypted, no `.pub` | Returns `&PassphraseMissingError{}` with **`.PublicKey == nil`** — the `encryptedBlock(block)` check fires before any parsing that could populate it |

Reading `err.PublicKey != nil` after a type assertion on `*ssh.PassphraseMissingError` *is* the
derivation-gap check. No stderr scraping, no exit-code interpretation, no format guess.

### Rationale

This directly satisfies P3 on the read path — encryption state and format are readable without
ever asking for, or attempting, decryption — and it satisfies P8: no process fork, no cgo build
step, per key read. It removes both of the predecessor's external dependencies at once, which
also feeds [T9](decision-T9.md): a statically linked, `CGO_ENABLED=0` binary is only genuinely
`FROM scratch`-capable if nothing in its call graph shells out to a tool that may not exist in
the container.

Also available on the same package, used elsewhere in the read/write path: `ParseAuthorizedKey`
(parses a `.pub` line), `FingerprintSHA256` (the `SHA256:`-prefixed, unpadded-base64 form
OpenSSH ≥6.8 uses — this is the fingerprint format hasp reports, never the legacy MD5 colon-hex
form, though `FingerprintLegacyMD5` exists if a future `find key` clue needs to match one),
`NewPublicKey`, `MarshalAuthorizedKey`, `MarshalPrivateKey`, and
`MarshalPrivateKeyWithPassphrase` — the last two are how `new key` ([T6](decision-T6.md)) authors a file,
the one write path D14 permits.

### Consequence

- Zero runtime dependency on `PATH` containing `ssh-keygen`, and zero cgo — `CGO_ENABLED=0`
  build in [T9](decision-T9.md) is unconditionally achievable, not merely likely.
- The derivation-gap table stops being documentation asserted by hand and becomes a fact a test
  suite checks directly: construct a fixture in each of the three rows, assert the concrete
  return type and the `PublicKey` field's nilness. This is the seed of the fixture matrix in
  [`tdd.md` §12](../../docs/tdd.md#12-testing-strategy--a-function-of-a-directory).
- No code path in `internal/adapter/keyfile` may call anything that supplies a passphrase to
  *decrypt* an existing key — there is no `ParsePrivateKeyWithPassphrase` call for reading, and
  a future contributor adding one to "helpfully" resolve an unknown fingerprint is adding a P3
  violation, not a feature. This absence is the enforcement mechanism, not a comment asking
  someone not to.
