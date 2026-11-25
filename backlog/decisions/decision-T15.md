---
id: decision-T15
title: 'Safety mechanics: atomic write, symlink-through, mode preservation, D4''s move rule'
date: '2026-08-28'
status: 'Accepted — refined by T20, T22'
---
### Context

P4 (every write reversible) and [D4](decision-D4.md) (*"hasp has no operation that can
destroy an irreplaceable secret... this is canon"*) both need a concrete write-time mechanism, not
just a policy. Four distinct sharp edges live here, and each has a specific way to get it wrong.

### Decision

**1. Atomic write.** Write to a temporary file in the **same directory** as the target (so the
final replace is same-filesystem and therefore eligible to be atomic on POSIX), `fsync` the temp
file, then `os.Rename` (`func Rename(oldpath, newpath string) error` — Go documents only that
*"if newpath already exists and is not a directory, Rename replaces it"*; the atomicity comes
from POSIX `rename(2)` on the platforms hasp targets, not from Go, which notes rename is *"not an
atomic operation"* on non-Unix platforms) the temp file onto the target, then `fsync` the
containing directory. The directory `fsync` is easy to skip and looks correct until a crash —
without it, the rename itself is not guaranteed durable on Linux.

**2. A symlinked `~/.ssh/config` is resolved and written through — the symlink itself is never
replaced.** Detected via `os.Lstat` distinguishing a symlink from a regular file, with
`filepath.EvalSymlinks` resolving the real target; the atomic-write dance in point 1 runs against
the **resolved** target's directory, not `~/.ssh`. Named explicitly because dotfiles repositories
make this layout common (`~/.ssh/config -> ~/dotfiles/ssh/config`), and a tool that transparently
removes and recreates the symlink as a plain file silently converts the user's dotfiles-managed
config into an orphaned copy — destructive by omission, not by design, which is exactly the class
of bug D4's spirit exists to prevent even though D4's letter is about key material specifically.

**3. Mode and ownership are preserved across a write.** `os.Stat` the original file before
writing, then apply the same mode (and ownership, where the process has permission to) to the
temp file before the rename in point 1 — so a `0600` config file does not silently become
umask-determined (commonly `0644`) after hasp's first touch.

**4. D4's move rule for key material: copy → verify → unlink source. Never unlink first.**
Verification is fingerprint re-derivation and comparison when the key is fingerprint-derivable
([T1](decision-T1.md)/[T12](decision-T12.md)); a byte-for-byte comparison for the undecidable case. This is the
mechanism behind `edit key --profile`, a plain relocation with no alias required — no failure
path between the copy and the verified unlink leaves zero readable copies of the secret.

**Refined for `adopt`, which must also leave an alias behind — see [T20](decision-T20.md).** `new key` and
`edit key --replace-material` are covered by the same atomic-write discipline (point 1) for a
different reason — they write to a path that may already hold a key — and are refined
in [T22](decision-T22.md).

### Rationale

Each point traces to a named principle: 1 and 3 to P4 (a reversible write is only meaningful if
the write itself cannot corrupt the file mid-flight, or silently loosen its permissions); 2 to P6
(nothing hasp does should require the user to have anticipated hasp's own implementation detail —
a dotfiles-managed symlink is the user's own arrangement, and hasp is a guest in it); 4 is D4
applied literally, at the level of the exact sequence of syscalls, not merely as a policy
statement.

### Consequence

- The guard test for point 4 injects a write failure partway through `adopt` (for example, a
  read-only destination directory) and asserts the **original** file still exists afterward — a
  test that fails loudly if a future refactor ever reorders copy/verify/unlink.
- The guard test for point 2 constructs a symlinked `~/.ssh/config` fixture and asserts, after a
  write, that the symlink's **target inode** is unchanged — not merely that the file's content is
  correct. Content-correct-but-inode-replaced is exactly the silent-breakage failure mode this
  entry exists to catch, and a content-only assertion would not catch it.
- These four mechanics are the concrete implementation behind every `RequiresBackup() == true`
  `Change.Apply` in [T4](decision-T4.md)'s `Applier` — this entry is where "how" lives; T4 is where "when."
