## hasp adopt key

Move an unmanaged key into a managed profile directory, leaving a top-level alias (D13)

```
hasp adopt key <name-or-clue> [flags]
```

### Options

```
  -h, --help             help for key
      --profile string   the managed profile to adopt the key into (dotted path, e.g. work.foobarco)
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

* [hasp adopt](hasp_adopt.md)	 - Move an unmanaged resource into managed territory

