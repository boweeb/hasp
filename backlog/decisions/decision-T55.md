---
id: decision-T55
title: 'Flag parse errors and extra positionals on zero-argument leaves are usage errors (exit 2), each fixed once at the command-tree root'
date: '2026-11-25'
status: 'Accepted · Extends: T54'
---
### Context

The review of [T54](decision-T54.md) found two more ways malformed input slipped past [`docs/tdd.md`](../../docs/tdd.md) §10's usage-error contract, both present on `main` before T54. First, every cobra flag parse error (`hasp --bogus`, `hasp list key --bogus`, `hasp list key --json=maybe`, `hasp list key --key-dir` with no value) printed cobra's message and exited `3`, because nothing set a `FlagErrorFunc`, so the error never carried `app.ErrUsage`. §10's table already assigns "bad flags/arguments" to code `2`, so the code disagreed with the TDD. `hasp --version`, a pre-v1 acceptance-testing finding, is one instance: hasp registers no `--version` flag, so it is an unknown flag. Second, the leaves that take no positionals (`version`, and `key`, `host`, `profile` under both `list` and `check`) declared no `Args`, and cobra skips its `legacyArgs` check entirely for a command with no subcommands, so `hasp version bogus` and `hasp check key extra` exited `0` and ignored the token. §16 freezes exit codes at `v1.0.0`, so a fix that only tightens behavior is cheap now and a compatibility break afterwards.

### Decision

**1. One `FlagErrorFunc` on root makes every flag parse error a usage error.** `flagUsageError` (`internal/cli/root.go`) wraps cobra's error as `app.ErrUsage: <cobra's text>`, so stderr reads `hasp: usage error: unknown flag: --bogus` and the process exits `2` with nothing on stdout. Cobra consults the root's func when a child sets none, so every current and future subcommand inherits it.

**2. Every Runnable command that declares no `Args` gets `rejectUnexpectedArgs`.** `rejectArgsWhereUndeclared` walks the finished tree once in `NewRootCmd`, before cobra's `help` command is added (its positionals are a topic path). Commands that already declare a validator (`exactArgs`, `minimumArgs`, help-only parents) keep theirs.

**3. On a leaf, `rejectUnexpectedArgs` names the token as an unexpected argument.** On a command with subcommands it keeps [T54](decision-T54.md)'s `unknown command "x" for "hasp list"` text and suggestions. On a leaf, that wording would mislead, since the token was never meant to be a command, so it reports `unexpected argument "x" for "hasp version"` instead.

### Rationale

Both gaps are the §10 contract being enforced unevenly, not a new rule, and each is a single defect at the tree's shared root, so each is fixed once there rather than per command file. A per-file `Args` on each leaf would fix today's leaves and leave the next one to repeat the omission. An exit code of `0` on a dropped token is the worse of the two. A user who typed `hasp check key extra` gets a clean result that looks like it covers what they asked, which states more confidence than the run earned, the opposite of [P10](../../docs/design.md#4-principles).

### Consequence

- `internal/cli/root.go`: `flagUsageError`, `rejectArgsWhereUndeclared`, and a leaf branch in `rejectUnexpectedArgs`.
- Guards: `TestEveryRunnableCommandValidatesArgs` in `internal/cli/helponly_test.go` fails for any Runnable command with nil `Args` (cobra's `help`, `completion`, and `__complete` are skipped: hasp does not own their argument contracts), and `internal/cli/testdata/script/usage-error-gaps.txtar` pins exit code, empty stdout, and the stderr line for each case against a real temp `--key-dir`. Each was confirmed to fail with its guard removed.
- Help, `hasp help <topic>`, bare-parent help, `__complete` output, `docs/cli/`, `manpages/`, and `completions/` are byte-identical to `main`: neither change touches any rendering path.
- Out of scope, left as is: cobra's generated `completion <shell>` commands keep cobra's own `NoArgs`, so `hasp completion bash extra` still exits `3`. Those commands are cobra's, added during `Execute`, and outside the tree hasp builds.
- [`docs/tdd.md`](../../docs/tdd.md) §10 states the rule and links here.
