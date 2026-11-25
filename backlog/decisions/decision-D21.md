---
id: decision-D21
title: 'Investigation is a first-class purpose; J10 is ratified'
date: '2026-09-03'
status: 'Accepted · Adds: J10 · Amends: §1, §3.1'
---
### Context

hasp's documented framing is management, with comprehension (P7) as the read-side goal. But the
project was born from an investigation: a Bash loop running `ssh-keygen -l -f` across `~/.ssh`,
grepping for a fingerprint held in hand from an AWS console. §2 already asks the question that
founding story motivates; what the design has never said is that *answering* it is a distinct kind
of work from managing an arrangement.

### Decision

Name **visibility** as the root purpose, serving two ends: **investigation** (answering a question
whose shape you do not yet know) and **operational management** (arranging what you already have).
Ratify **J10 — Investigate**. ID order is not chronological here any more than it is in the
technical log, where [T27](decision-T27.md) was decided first and numbered last: J10 is
numbered last and is v1 scope, while J9 remains post-v1.

### Rationale

An investigation needs *more* data than comprehension does — including signals hasp cannot itself
interpret, because the human is looking for a thread hasp was never taught to see. That is the
opposite of P7's instinct, and it needs its own name so the two do not fight. P7 does not need
amending — it already scopes itself to *"The default output of a read"* — but the inversion is
worth stating rather than leaving implicit: under `--investigate` the human correlating output
*is* the intended workflow, exactly the case P7 elsewhere calls a failure.

### Consequence

`--investigate` has a journey to cite, and future commands may grow the same mode. §2's founding
story is recorded rather than left as folklore.
