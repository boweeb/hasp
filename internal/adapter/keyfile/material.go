package keyfile

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// Material is the parsed key material a fingerprint scheme (internal/adapter/fpscheme, T35)
// needs: either the public half alone, or the decrypted private key — T35's declared-material
// distinction ("the load-bearing field") made concrete as a Go type. It deliberately carries
// less structure than Info: Info is Inspect's plain-read answer (T1), computed without ever
// asking for a passphrase (P3, D19's custody rule, unchanged); Material is a strictly separate,
// opt-in accessor for the smaller set of callers that have D19's consent for one operation.
type Material struct {
	Public  ssh.PublicKey // nil when underivable (§5.1's derivation gap)
	Private any           // nil unless the caller supplied a correct passphrase, or the key is unencrypted
	Comment string        // "" if unavailable; populated from the .pub sidecar when one exists (T1's fast path)
}

// OpenMaterial reads the private key file at path and returns whatever Material can be derived
// from it, given passphrase.
//
// passphrase == nil means "public half only, never attempt to decrypt" — the only caller as of
// M3.6.1 (chunk M3.6.2 wires a real passphrase in, sourced from ssh-agent or an
// x/term.ReadPassword prompt, T38/T39). An unencrypted private key is readable regardless of
// passphrase — Material.Private is populated whenever the on-disk key is not itself encrypted,
// matching Inspect's own Encrypted field (T1): "the caller supplied no passphrase" and "the key
// is encrypted" are different facts, and OpenMaterial must not conflate them by leaving Private
// nil for a plain key just because passphrase was nil.
//
// A wrong or rejected passphrase is reported as Material.Private == nil, never as an error — the
// same §5.1 stance an undecidable fingerprint already takes: an honest "still unknown" beats
// aborting the whole investigation over one key. OpenMaterial's error return is reserved for
// path itself being unreadable or not a recognizable private key at all, the same failure shape
// Inspect already uses.
//
// D19/T39's shape, made legible on sight rather than merely asserted in this comment (tdd.md
// §11's own instruction for exactly this function): OpenMaterial is the one call site in this
// tree that ever supplies a passphrase to an *existing* key's decryption, so a reviewer scanning
// a diff for a P3/D19 violation has exactly one place to check, the same review discipline T6's
// structural audit already established for `new key`'s generation path. It is:
//   - user-initiated: reachable only from a code path an explicit flag chose for this
//     invocation (--investigate, D21/D19) — never from a plain read's derivation pipeline.
//   - scoped to one operation: no cache, no memoization; a second call re-reads and
//     re-decrypts from scratch. Nothing here retains passphrase or key material past the call.
//   - zeroed after use, by the caller: OpenMaterial does not copy passphrase anywhere that
//     outlives this call, and never itself zeroes the slice, because the caller owns the
//     backing array (typically x/term.ReadPassword's return, T6) and is the only party who
//     knows when every derivation that needed it has finished running. The caller is
//     responsible for zeroing it once done — this function's contract is "does not retain,"
//     not "zeroes on the caller's behalf."
func OpenMaterial(path string, passphrase []byte) (Material, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Material{}, fmt.Errorf("read %s: %w", path, err)
	}

	var mat Material

	key, err := ssh.ParseRawPrivateKey(raw)
	switch err {
	case nil:
		// Not encrypted: readable regardless of passphrase (T1's plain-key path).
		mat.Private = key
		if signer, sErr := ssh.NewSignerFromKey(key); sErr == nil {
			mat.Public = signer.PublicKey()
		}
	default:
		var missing *ssh.PassphraseMissingError
		if !errors.As(err, &missing) {
			return Material{}, fmt.Errorf("%s: %w", path, err)
		}
		if missing.PublicKey != nil {
			// OpenSSH wire format embeds the public half unencrypted (T1's second
			// derivation-gap row) — derivable without any passphrase at all.
			mat.Public = missing.PublicKey
		}
		if passphrase != nil {
			if decrypted, dErr := ssh.ParseRawPrivateKeyWithPassphrase(raw, passphrase); dErr == nil {
				mat.Private = decrypted
				if signer, sErr := ssh.NewSignerFromKey(decrypted); sErr == nil {
					mat.Public = signer.PublicKey()
				}
			}
			// A wrong passphrase falls through with mat.Private left nil — an honest
			// unknown (§5.1), not an error; see the doc comment above.
		}
	}

	if pubBytes, pErr := os.ReadFile(path + ".pub"); pErr == nil {
		if pub, comment, _, _, aErr := ssh.ParseAuthorizedKey(pubBytes); aErr == nil {
			mat.Comment = comment
			// The .pub sidecar is authoritative for the public half whenever it parses,
			// the same fast path Inspect already takes (T1) — it never requires a
			// passphrase and never requires the private file to have parsed at all.
			mat.Public = pub
		}
	}

	return mat, nil
}
