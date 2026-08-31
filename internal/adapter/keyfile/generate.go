package keyfile

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// GenerateOptions controls new key generation (tdd.md §9's `new` grid cell, T1). Only ed25519 is
// supported — the grid's stated default — so there is no Algorithm field yet; a later algorithm
// choice is additive to this struct, not a redesign of it.
type GenerateOptions struct {
	Comment string

	// Passphrase, if non-empty, encrypts the generated private key (D17, T6: accepted only at
	// generation time). Generate zeros this slice in place before returning, regardless of
	// outcome — the caller's own copy of the secret is zeroed as a side effect, since a Go slice
	// shares its backing array with the caller (T6: "defers zeroing the slice").
	Passphrase []byte
}

// GeneratedKey is a freshly authored ed25519 keypair, ready to write to disk. Both fields are
// plain bytes with no further processing needed before a WriteKeyFile change carries them.
type GeneratedKey struct {
	PrivateKeyPEM []byte // OpenSSH-format private key, PEM-encoded (ssh.MarshalPrivateKey[WithPassphrase])
	PublicKeyLine []byte // authorized_keys-format public key line, newline-terminated
}

// Generate creates a new ed25519 keypair and marshals it to the OpenSSH private key format plus
// its .pub sidecar line, using only crypto/ed25519 and golang.org/x/crypto/ssh — no subprocess,
// no cgo (T1). It never reads or writes anything on disk; the caller (internal/app's
// NewKeyUseCase) is responsible for turning the result into WriteKeyFile changes.
func Generate(opts GenerateOptions) (GeneratedKey, error) {
	defer zeroBytes(opts.Passphrase)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return GeneratedKey{}, fmt.Errorf("generate ed25519 key: %w", err)
	}

	var block *pem.Block
	if len(opts.Passphrase) > 0 {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, opts.Comment, opts.Passphrase)
	} else {
		block, err = ssh.MarshalPrivateKey(priv, opts.Comment)
	}
	if err != nil {
		return GeneratedKey{}, fmt.Errorf("marshal private key: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return GeneratedKey{}, fmt.Errorf("derive public key: %w", err)
	}
	line := ssh.MarshalAuthorizedKey(sshPub)
	if opts.Comment != "" {
		line = append(line[:len(line)-1], []byte(" "+opts.Comment+"\n")...)
	}

	return GeneratedKey{
		PrivateKeyPEM: pem.EncodeToMemory(block),
		PublicKeyLine: line,
	}, nil
}

// zeroBytes overwrites b's contents in place — explicit, rather than left for garbage collection
// (D17, T6's "buffer zeroed immediately after use").
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
