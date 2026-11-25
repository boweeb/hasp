---
id: decision-D20
title: 'J2 spans fingerprint schemes, not merely spellings'
date: '2026-09-03'
status: 'Accepted · Amends: J2 (§7)'
---
### Context

J2 currently reads *"Punctuation and case shouldn't matter,"* which frames normalization as
cosmetic — as though there were one fingerprint per key and only its spelling varied. That is
false: AWS alone computes an RSA fingerprint one of two different ways depending on a key's
provenance (SHA-1 over the private key if AWS generated it, MD5 over the public key if it was
imported), and a third way again for ED25519, so the same key can carry more than one valid
fingerprint and the console shows exactly one of them. A user pasting a console fingerprint into a
tool that only knows the scheme its own key format natively produces gets silence back — and
silence reads as "you don't have this key," which is a confidently wrong answer, not a missing
one.

### Decision

J2's requirement is matching across every **scheme** hasp knows, not just every spelling.
Normalization spans hash algorithm and encoding, not only punctuation and case — including
stripping base64 `=` padding, which AWS emits and `ssh-keygen` omits.

### Rationale

The failure mode is silent and produces a confident wrong answer, which is worse than an explicit
error. This is exactly the question §2 opens with: a fingerprint from an AWS console, and which of
the user's keys it is.

### Consequence

`find` acquires a scheme registry. A false negative here is now a defect against a ratified
journey, not a missing convenience.
