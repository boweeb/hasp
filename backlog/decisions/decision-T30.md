---
id: decision-T30
title: 'The preview/apply race is detected by a witness, and fails closed'
date: '2026-08-29'
status: 'Accepted · Amends: T4, T15'
---
**Closes:** a gap named, but not closed, by [T26](decision-T26.md)

### Context

[T26](decision-T26.md) ended by naming a problem it did not solve:

> Computing the diff means `Plan()` reads the target file's current bytes. Reads are always safe
> (P5), so this costs nothing in guarantees — but it does mean a `Plan` is computed against a
> snapshot, and a file changed between preview and confirm is a race the `Applier` must detect
> rather than assume away.

Nothing acted on it. `tdd.md` §11 presents itself as *"One table of every guard in the system and
which way it fails"* and had no row for this one — so an implementer reading that table would
have concluded, correctly per the document and incorrectly in fact, that the case was handled.

The window is real and not narrow. [D6](decision-D6.md)'s cycle is **preview → confirm → back
up → write**, and the confirm step is a human reading output: seconds to minutes. In that window
an editor can save `~/.ssh/config`, another shell can run `ssh-keygen`, or a dotfiles manager can
re-link the file.

**P8 does not excuse it.** *"A single human operating interactively"* rules out concurrent hasp
processes contending for a lock. It does not rule out that same human having the file open in
another window — which is the overwhelmingly likely form of this race, and arguably at its most
likely precisely when they are being asked to confirm a change to that file.

### Decision

**Every `Change` that read a "before" state records a witness of it, and `Applier` re-verifies
all witnesses before applying anything.**

```go
type Witness struct {
    Path    string
    Size    int64
    ModTime time.Time
    Sum     [32]byte // SHA-256 of the bytes Plan() actually read
}
```

- **Captured at `Plan()` time**, from the same read that produced `Preview().Diff` — no extra I/O.
- **Re-verified for the whole `Plan` before the first `Change` is applied**, not per-change as it
  goes. A `Plan` is the unit the user consented to, so it is the unit that gets validated.
- **Any mismatch fails closed.** Nothing is written, nothing is backed up, exit `3`
  ([T14](decision-T14.md)), with a message naming the file that changed and saying to re-run.

**The hash decides.** Size and mtime are carried because they make the failure cheap to explain,
but a file rewritten to byte-identical content is **not** a race — the preview is still accurate —
and hasp does not refuse on a touched mtime alone.

### Rationale

D6's safety is bought entirely by the user *reading* the preview. A plan whose preview no longer
describes the file it is about to write spends that safety without the user's knowledge: they
consented to a change against a state that no longer exists. Applying it anyway is worse than
refusing, because the refusal costs one re-run and the alternative costs a silent, unreviewed
write to a file P2 promises to protect.

Fail-closed is the default `tdd.md` §11 already declares for any guard without a stated exception,
and nothing here argues for an exception.

The competing option — re-read, recompute, and re-preview automatically — was rejected. It turns
one confirmed decision into a loop nobody asked for, and a user who has just read a diff and
pressed `y` should not be shown a *different* diff and asked again in the same breath. Re-running
is explicit, and costs one keystroke.

**Backups do not make this safe on their own.** P4 would let the user recover the clobbered file,
which is exactly the *apology rather than a safety property* shape
[D18](decision-D18.md) rejects for `release host`. The same reasoning applies here.

### Consequence

- **`tdd.md` §11's table gains its missing row**, which makes its claim to list every guard true.
- **A `Plan` is time-limited**, and that is worth stating explicitly: it is valid only against the
  machine state it was computed from. Nothing persists a `Plan` across processes today, and this
  entry is the reason not to start.
- A guard test mutates a target file between `Plan()` and `Apply()` and asserts three things: the
  write is refused, the target's bytes are unchanged, and **no backup was written** — a backup
  must not fire for a plan that never applies, or `~/.ssh/.hasp-backups/` fills with snapshots of
  writes that never happened.
- `WriteKeyFile` with `AllowOverwrite == false` ([T22](decision-T22.md)) is unaffected and keeps its own
  separate check: it fails closed on an existing target regardless of any witness, because it
  never read a "before" state to witness.
