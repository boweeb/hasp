## hasp show key

Show full detail for one key: identity, locations, profiles, and bound hosts

```
hasp show key <name> [flags]
```

### Options

```
  -h, --help          help for key
      --investigate   opt-in investigation mode (tdd.md §18): every registered fingerprint scheme, a confidence-graded origin, and ssh-agent-sourced facts, at the cost of speed and possibly a passphrase prompt
```

### Options inherited from parent commands

```
      --json             machine-readable output
      --key-dir string   key directory (default ~/.ssh) (default "~/.ssh")
      --no-color         disable ANSI color in human output
      --verbose          enable diagnostic logging to stderr
      --yes              consent to apply a write non-interactively
```

### SEE ALSO

* [hasp show](hasp_show.md)	 - Show full detail for one resource

