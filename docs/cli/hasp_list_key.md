## hasp list key

List every key found, managed and unmanaged

```
hasp list key [flags]
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

* [hasp list](hasp_list.md)	 - List every resource of one kind

