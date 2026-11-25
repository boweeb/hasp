---
id: decision-T35
title: 'The fingerprint scheme registry: open, pure-Go, no subprocess, no network'
date: '2026-09-03'
status: 'Accepted'
---
### Context

[D20](decision-D20.md) widens [J2](../../docs/design.md#7-journeys) from matching spellings to matching
**schemes** — the same key produces a different fingerprint depending on how it was computed, and
AWS alone uses three: SHA-1 over the PKCS#8 DER of the *private* key for an AWS-created RSA key,
MD5 over the PKIX/SPKI DER of the *public* key for an imported RSA key, and SHA-256 over the SSH
wire-format public key for ED25519 (created or imported — identical either way). This session
verified all three against `https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/verify-keys.html`
and confirmed them on the repository's own fixtures:

| Scheme | Hashed input | Hash | Needs | Verified value (`testdata/keys/rsa-pem-plain-pub`) |
| --- | --- | --- | --- | --- |
| AWS created-RSA | PKCS#8 DER of the **private** key | SHA-1 | private key, decrypted | `97:47:11:3c:af:56:47:b3:f9:a9:89:36:6d:ca:be:0b:33:a0:05:f7` |
| AWS imported-RSA | PKIX/SPKI DER of the **public** key | MD5 | public half only | `a8:e7:45:95:5f:a3:f0:b1:79:6c:c2:f1:d2:80:57:ea` |
| AWS ED25519 (created *or* imported) | SSH wire-format public key | SHA-256 | public half only | `SHA256:lqTGTP6KJSQptQJQEZj7scuX7jLWFfb0qoAHTT1IpPA` (`testdata/keys/ed25519-openssh-plain-pub`) |
| Legacy SSH MD5 | SSH wire-format public key | MD5 | public half only | `34:29:f4:da:3c:db:49:4b:35:ba:c1:c2:cd:2e:75:a8` |

**The fourth row is the one a reader should not skip.** Legacy SSH MD5 — what `ssh-keygen -E md5`
prints, what OpenSSH printed by default before 6.8, and what a great deal of surviving
documentation shows — has **exactly the same shape** as AWS imported-RSA (16 bytes, colon-hex, 47
characters) and a **completely different value**, because one hashes the SSH wire-format blob and
the other hashes the PKIX/SPKI DER encoding. The two rows above prove it on the same key:
`34:29:f4:…` versus `a8:e7:45:…`. [T1](decision-T1.md) anticipated exactly this, noting that
`FingerprintLegacyMD5` *"exists if a future `find key` clue needs to match one"*. Registering it
here, rather than leaving it to be discovered by a user whose correct clue silently failed to
match, is the difference between an open registry and an AWS-shaped one.
[T36](decision-T36.md) carries the consequence: a 47-character clue is ambiguous and both schemes are
computed.

**Six shell "strategies" collapse to three values.** Proven on the repository's own fixtures:
`openssl rsa -pubout -outform DER | openssl md5` and
`ssh-keygen -e -m PEM | openssl rsa -RSAPublicKey_in -outform DER | openssl md5` both produce the
imported-RSA value above, on the same input. The second pipeline exists only because `openssl rsa
-in` cannot read an OpenSSH-format private key — verified this session: exit 1, `"Could not find
private key"`. hasp parses both formats natively ([T1](decision-T1.md)), so it needs **one code path per
scheme**, never one per shell incantation that happens to produce that scheme.

**A shell pipeline that loses its input silently emits a hash of nothing**, and that hash is
shaped exactly like a real fingerprint: `d41d8c…` is MD5 of the empty string, `da39a3…` is SHA-1
of the empty string. Either one slots into a fingerprint-shaped field without complaint. A native
Go implementation that reads the key bytes it already parsed cannot make that mistake — there is
no pipe stage for the input to fall out of.

### Decision

A **registry** of scheme definitions, each declaring: a stable `id`, the hash function, the
encoding it renders as, and — the load-bearing field — **which key material it needs**: public
half only, or the decrypted private key. A scheme is `func(Key) (Fingerprint, error)`, computed
entirely from bytes hasp already has in memory; nothing in the registry ever opens a socket. This
is the boundary the user ratified explicitly: **a scheme is an encoding of facts about the key,
computed locally — never a network call** — which keeps §3.2's *"Not a key distribution
mechanism."* intact even though this feature is, on its face, about matching a value from an
external console.

Every scheme is stdlib or already-required: `x509.MarshalPKCS8PrivateKey` +
`crypto/sha1` for created-RSA; `x509.MarshalPKIXPublicKey` + `crypto/md5` for imported-RSA; the
existing `x/crypto/ssh` wire-format marshaling + `FingerprintSHA256` ([T1](decision-T1.md)) for the
SSH-native/ED25519 scheme. **Zero new modules.**

**The registry is open from the start, not closed at three.** GitHub and GitLab both display
SHA-256 fingerprints — already covered by the SSH-native scheme — and other consoles will surface
their own schemes over time. Each scheme's declared input requirement is what lets a caller (§9's
`find`, [T36](decision-T36.md)) know in advance whether a given key can even be evaluated against it without
asking the user for anything.

### Rationale

The three-scheme table is not incidental complexity — it is the literal shape of the problem
[J10](../../docs/design.md#7-journeys) exists to answer, and D20's own rationale is that a false negative here
reads as *"you don't have this key,"* which is worse than an honest miss. The empty-digest hazard
is the strongest available argument for computing every scheme natively rather than shelling out
to whichever `openssl`/`ssh-keygen` pipeline happens to exist on the caller's `PATH`: a broken
pipe silently produces a value indistinguishable from a real one, and [T1](decision-T1.md)'s "no subprocess"
rule already forecloses that failure class for the read path generally — this is one more scheme
falling under a rule already in force, not a new exception to it.

### Consequence

- [T36](decision-T36.md)'s `find` and any future `--investigate` surface consume the registry as a single
  dependency; adding a fourth scheme is one new entry, not a new call site scattered through the
  command surface.
- Each scheme's declared input requirement (public half vs. private key) is exactly what
  [D19](decision-D19.md)'s consent gate keys off of: a caller can enumerate every scheme
  computable from what has already been read, and knows in advance which remaining ones would
  require asking the user for something.
- `tdd.md` §5 gains scheme computation as a projection step in the derivation pipeline, and §12
  gains a fixture-backed test asserting the AWS vectors above by name, plus a guard that no scheme
  performs I/O beyond reading the key file already in memory.
