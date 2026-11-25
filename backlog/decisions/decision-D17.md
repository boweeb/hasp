---
id: decision-D17
title: '`new key` may ask for a passphrase, at generation time only'
date: '2026-08-29'
status: 'Accepted · Narrows: §3.2, P3'
---
**Opened by:** [`docs/tdd.md`](../../docs/tdd.md) §9, [`T6`](decision-T6.md)

### Context

`docs/design.md` §3.2 lists nine hard boundaries under the strongest sentence in that document:
*"These are hard boundaries. Each one, if crossed, turns hasp into a different and worse
project."* One of the nine is *"**Not a passphrase manager.** hasp never asks for, stores, or
transmits a passphrase."*

`new key` generates a keypair. A generated key either carries a passphrase or it does not, and
hasp is the process authoring the file, so something must supply the answer. The honest starting
point — stated in [T6](decision-T6.md), which corrected an earlier draft of itself for
getting exactly this wrong — is that **nothing in the ratified design licenses hasp asking.**

[D14](decision-D14.md)'s carve-out is the sentence that looks like it might: *"hasp never writes to a private
key file. The sole exception is generating a new key, where hasp authors the file outright."* It
licenses hasp **authoring** the file. It says nothing about **asking** for a passphrase while
doing so. Those are different acts, and reading the second out of the first is precisely the kind
of citation error this log exists to catch.

So: an amendment, not a reinterpretation.

### Options

**(i) Never ask; flags only.** `--no-passphrase` and `--passphrase-stdin`, no prompt ever. §3.2
survives untouched, since accepting piped bytes is not hasp *asking*. *Cost:* the interactive
path — overwhelmingly the common one — becomes typing a secret into a pipe with no masking, no
confirmation, and no signal that the terminal is waiting. The strictest option produces the worst
interactive experience, and the predictable response is that people stop setting passphrases.

**(ii) Ask, with the boundary narrowed.** A real prompt, no local echo, at generation time only.

**(iii) Ask, and treat the existing carve-out as already permitting it.** Rejected on sight: that
is the citation error above, and adopting it would make this document claim to say something it
does not.

### Decision

**Option (ii).** §3.2's passphrase boundary is **narrowed**, by exactly this case and no other.

hasp may accept a passphrase **only** at generation time, and the narrowing carries four
constraints that are its entire scope:

1. **Generation only.** Never to decrypt, unlock, or re-encrypt an existing key.
2. **Never persisted.** Not in settings, not in metadata, not in a log, not in a backup.
3. **Never transmitted.** §3.2's *"transmits"* is not narrowed at all.
4. **Zeroed after use.** The buffer does not outlive the write.

### Rationale

The boundary was written to keep hasp out of *custody* of secrets — to stop it becoming a vault,
an agent, or a thing you must trust with the keys to everything. A passphrase that exists for one
function call and is then zeroed puts hasp in custody of nothing. It puts hasp exactly where
`ssh-keygen` already stands, for the same few milliseconds, doing the one job hasp is already
ratified to do.

Refusing to narrow would not have protected anything. It would have pushed users toward keys with
no passphrase at all — a worse security outcome produced by the stricter rule, which is the
failure mode where a principle held literally defeats the purpose it was written for.

**The enforcement is structural, not documentary.** No function in hasp's domain or application
layer accepts a passphrase alongside an *existing* key value. That absence is the mechanism: a
diff adding one is a visible violation on sight — [P6](decision-D12.md)'s legibility applied to a safety
property rather than to data.

### Consequence

- **§3.2 and P3 both change**, and `docs/design.md` states the narrowed form in both places. The
  bullet remains a hard boundary with one named exception; it does not become a preference.
- **Every existing citation of P3-as-passphrase-prohibition is unaffected, and this is the
  precision that matters.** `docs/design.md` §5.1's derivation gap still refuses to prompt in
  order to fingerprint an *existing* key — that is a read of something already on disk, and D17
  does not reach it. [D14](decision-D14.md)'s rejection of comment-marking still stands on exactly the
  ground it was decided on: marking an encrypted key would require decrypting it. **D17 narrows
  the boundary for a file hasp is authoring, and loosens nothing about keys that already exist.**
- Non-interactive callers use `--no-passphrase` or `--passphrase-stdin`, and a machine-wide
  default is expressible in settings ([D16](decision-D16.md)). With no mode resolvable and no TTY, `new key`
  **fails closed** rather than silently generating an unprotected key.
- **hasp is still not a passphrase manager.** It cannot store one, retrieve one, or use one to
  open anything, and it has no operation that takes a passphrase together with an existing key.
  The one thing it can now do is hold a secret it never keeps, for a file it is writing itself.
