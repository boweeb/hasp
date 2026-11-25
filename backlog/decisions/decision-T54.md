---
id: decision-T54
title: 'Unknown verbs and nouns are usage errors (exit 2): Runnable-plus-`NoArgs` validation, a help-only annotation that keeps help and generated docs unchanged, and restored suggestions'
date: '2026-11-24'
status: 'Accepted'
---
### Context

Before this entry, `hasp list bogus` printed `list`'s help and exited `0`, and `hasp check keyz` ran the aggregate check, because cobra's default `legacyArgs` validator only rejects unknown arguments on the *root* command; a verb parent without a runnable of its own silently swallows them. `hasp lsit` did reach `legacyArgs`, but exited `3` instead of the usage code `2`. [`docs/tdd.md`](../../docs/tdd.md) §16 freezes the exit-code contract at `v1.0.0`, so this had to be settled pre-v1: a typo is a usage mistake, and `2` is the code scripts already branch on for that.

### Decision

**1. Unknown verbs and nouns exit `2` via `app.ErrUsage`.** `rejectUnexpectedArgs` (`internal/cli/root.go`) wraps `cobra.NoArgs` in `app.ErrUsage` and is set as `Args` on root and every verb parent. `check` keeps its real `RunE` (bare `hasp check` still runs the aggregate); registered noun aliases are real subcommands and never reach the validator.

**2. Root and help-only parents are given a `RunE` purely so cobra validates them.** Cobra only runs `ValidateArgs` on a Runnable command, so each parent that had no `RunE` gets `showHelp`. `markHelpOnly` sets `Args`, `RunE`, and a `hasp.help-only` annotation together, so the three cannot drift.

**3. The annotation keeps help and generated docs byte-identical to the pre-change output.** A Runnable parent would otherwise print `hasp list [flags]` above `hasp list [command]` in help, and `cobra/doc` would add a `hasp list [flags]` synopsis to `docs/cli/*.md`. The root usage template is cobra's default with the leading `Usage:{{if .Runnable}}` narrowed to also require the annotation to be absent (`hideRunnableUsageLine`, which panics if cobra's template ever stops containing that text). `tools/gendocs` calls the exported `cli.StripHelpOnlyRunners` on its private tree before generating, so `Runnable()` is false there and `docs/cli/`, `manpages/`, and `completions/` need no regeneration.

**4. "Did you mean" suggestions are restored.** `cobra.NoArgs` omits them, so `rejectUnexpectedArgs` appends the same text cobra's private `findSuggestions` produces, via the public `SuggestionsFor`, honoring `DisableSuggestions` and replicating cobra's bump of a non-positive `SuggestionsMinimumDistance` to `2`. Parents gain suggestions too (`hasp list kye` suggests `key`).

### Rationale

Exit `2` for a mistyped command is the §10 contract's own definition of a usage error, and freezing it wrong at `v1.0.0` would mean either a breaking change later or a permanent inconsistency. Making the parents Runnable is the smallest mechanism cobra offers for argument validation on non-root commands; the cost (Runnable leaking into help and docs) is contained in two narrow, tested places rather than hand-writing help and doc templates. The mechanism is entirely CLI-boundary: nothing in `internal/app` or `internal/domain` changes.

### Consequence

- `internal/cli/root.go`: `rejectUnexpectedArgs`, `markHelpOnly`, `StripHelpOnlyRunners`, `hideRunnableUsageLine`, `suggestionsText`.
- `tools/gendocs/main.go` strips help-only runners before generating; its package doc says why.
- Guards: `internal/cli/helponly_test.go` and `internal/cli/testdata/script/reject-unknown-nouns.txtar`; `go run ./tools/gendocs -check` fails if the strip is removed.
- [`docs/tdd.md`](../../docs/tdd.md) §10 states the rule and links here.
- Because root and the parents are now Runnable, `PersistentPreRunE` (`readGlobalFlags` plus `configureLogging`) also runs for bare root and bare parents; this is harmless, since `requireKeyDir` is not called there. Cobra's built-in help command is wrapped (`preserveUnknownHelpTopic`) so `hasp help bogus` keeps its "Unknown help topic" output on stderr, which root's non-nil `Args` would otherwise bypass.
- A cobra upgrade that changes the default usage template or `findSuggestions` text fails tests loudly rather than drifting silently.
