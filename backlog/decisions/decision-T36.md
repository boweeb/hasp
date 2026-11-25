---
id: decision-T36
title: '`find` matches across every registered scheme; the clue''s shape routes the search'
date: '2026-09-03'
status: 'Accepted — amends `tdd.md` §9''s `find` cell'
---
### Context

[D20](decision-D20.md) requires `find key` to match a clue given in *any* scheme hasp knows,
not only the SSH-native one `tdd.md` §9 originally described ("identify a key from a fingerprint
fragment... normalizing punctuation and case"). [T35](decision-T35.md) supplies the schemes; this entry
supplies the matching algorithm — including how a bare clue, with no metadata attached, is routed
to a candidate scheme before any key is even examined.

### Decision

**Normalization**, applied to the clue before routing: strip colons and internal whitespace,
lowercase hex digits, strip base64 `=` padding (AWS emits it, `ssh-keygen` omits it — D20 names
this explicitly), and tolerate an optional `SHA256:`/`MD5:` prefix.

**The clue's raw shape NARROWS the search; it does not uniquely determine it.** That distinction
is the whole of this decision, and getting it wrong would reintroduce the silent miss
[D20](decision-D20.md) exists to eliminate.

| Clue shape | Candidate schemes |
| --- | --- |
| 59-character colon-hex (40 hex digits) | SHA-1 — [T35](decision-T35.md)'s created-RSA scheme. One candidate |
| 47-character colon-hex (32 hex digits) | MD5 — **two** candidates: AWS imported-RSA (MD5 over PKIX/SPKI DER) **and** legacy SSH MD5 (MD5 over the SSH wire-format blob) |
| base64, with or without a `SHA256:` prefix | SHA-256 — the SSH-native/ED25519 scheme. One candidate |

**The MD5 collision is real and was anticipated by this log.** [T1](decision-T1.md) already noted that hasp
reports SHA-256 and *"never the legacy MD5 colon-hex form, though `FingerprintLegacyMD5` exists if
a future `find key` clue needs to match one"* — this entry is that future. Legacy SSH MD5 and AWS
imported-RSA are **different hashes over different byte sequences of the same key**: the former
over the SSH wire-format public blob, the latter over the PKIX/SPKI DER encoding. They produce
different digests and identical shapes. A 47-character clue is therefore ambiguous by
construction, and `find` **computes both and compares both** rather than guessing one. Legacy MD5
clues are not exotic — they are what `ssh-keygen -E md5` prints, what OpenSSH printed by default
before 6.8, and what a great deal of still-circulating documentation shows.

`find` computes only the schemes a clue's shape admits, not every registered scheme for every key,
so the common case (an SSH-native clue) stays exactly as cheap as it is today. What the shape
buys is a smaller candidate set, never a single answer.

**Which scheme matched is the origin evidence — but only for the AWS schemes.** Per
[P10](../../docs/design.md#4-principles): a match under created-RSA or imported-RSA is deterministic proof of
provenance, because AWS computes one or the other depending on exactly how the key came to exist,
so `find` reports origin as `confirmed`. A match under **legacy SSH MD5** proves nothing about AWS
provenance — it is simply another way of naming the same public key — so origin stays `possible`.
The same holds for the SSH-native/ED25519 scheme, because AWS's ED25519 fingerprint is identical
whether the key was created or imported (Background, verified against AWS's documentation). Only
two of the four registered schemes carry provenance, and conflating "the clue matched" with "the
origin is proven" is exactly the error P10's vocabulary exists to prevent.

### Rationale

D20's own rationale carries this entry: a silent miss on a clue given in a scheme hasp did not
think to try reads as "you don't have this key," which is a confidently wrong answer, not a
missing one — precisely the question §2 opens the whole document with. Routing by shape rather
than computing every scheme against every key keeps the common path — an SSH-native fingerprint,
the overwhelming majority of clues in practice — exactly as cheap as it was before this entry.

### Consequence

- `tdd.md` §9's `find` cell for **key** is amended to describe multi-scheme matching; the **host**
  and **profile** `find` cells are unaffected, since neither is fingerprint-shaped.
- A clue that matches under the created-RSA scheme is proof hasp read the *private* key
  ([T35](decision-T35.md)) — which only happens under [D19](decision-D19.md)'s consent gate — so a
  successful created-RSA match is also, incidentally, evidence the gate fired correctly for that
  key during this invocation.
- `check`'s existing duplicate-detection vocabulary ([T12](decision-T12.md)) already distinguishes confirmed
  from unconfirmed; `find`'s confirmed/possible split is the same distinction applied to a
  different question, which is part of the evidence [D22](decision-D22.md) cites for P10's
  taxonomy being right-sized rather than newly invented.
