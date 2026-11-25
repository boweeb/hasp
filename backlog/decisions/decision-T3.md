---
id: decision-T3
title: '`spf13/cobra` for the CLI, commands registered from a table'
date: '2026-08-28'
status: 'Accepted'
---
### Context

The interface layer needs a command framework. [D10](decision-D10.md) fixes the grid at
3 nouns × 8 verbs (24 first-class combinations) with second-class nouns surfacing as flags; the
predecessor's one genuinely good idea, per
`docs/project-assessment-2026-08.md` §2.1, was that this grid was **mechanically** registered —
`cli/verb/<verb>.py` imported the same-named function from each `cli/resource/*.py` — so adding
a noun cost nothing structurally. That property is worth preserving in Go.

### Decision

**`spf13/cobra`**, over `alecthomas/kong`, which was seriously considered.

Commands are registered from a single Go table rather than built up ad hoc, so the grid's
uniformity survives the language change:

```go
type CommandSpec struct {
    Verb, Noun string
    Build      func(deps Deps) *cobra.Command
}

var commands = []CommandSpec{
    {Verb: "list", Noun: "key", Build: buildListKey},
    {Verb: "show", Noun: "key", Build: buildShowKey},
    // ... 24 entries, one per cell in tdd.md §9
}
```

### Rationale

`kong` is arguably the better technical fit for a rigid, statically-known grid — struct-tag
declaration is terser than cobra's imperative `*cobra.Command` construction, and hasp's grid is
about as static as a CLI's surface gets. It loses on two points that matter here: cobra
generates shell completions and man pages as first-class subcommands
(`cobra.Command.GenManTree`, the built-in `completion` command), both of which
[T9](decision-T9.md)'s GoReleaser packaging consumes directly (nfpms/completions archive entries); and
cobra is the ecosystem default, which matters for a project whose entire prior failure mode was
unfinished infrastructure choices (`docs/project-assessment-2026-08.md` §5, "Verb×resource
matrix or `factory/`? ... you never picked one"). This call is routine enough not to need the
user's sign-off, and consequential enough to name so it can be overridden.

### Consequence

- Adding a noun means adding entries to `commands` and to `internal/app`'s use-case set; the CLI
  tree itself never needs hand-editing beyond the table.
- If the table ever fights cobra's construction model badly enough to erase this benefit, `kong`
  remains the documented alternative — this entry is the record of why it was not chosen first,
  not a claim that it never could be.
- Generated completions and man pages are a packaging concern owned entirely by
  [T9](decision-T9.md)/[`tdd.md` §13](../../docs/tdd.md#13-build--distribution--goreleaser-as-a-constraint-not-an-afterthought);
  this entry only establishes that cobra makes them available.
