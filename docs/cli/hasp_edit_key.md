## hasp edit key

Rename, alias, replace material, or move a key between profiles

```
hasp edit key <name-or-clue> [flags]
```

### Options

```
      --add-alias string          create an alias for the key at <profile>/<name> (bare <name> for a top-level alias) — D2's write path (T19)
  -h, --help                      help for key
      --name string               rename the key's real file in place (new basename)
      --profile string            move the key's real file into a different, already-existing profile directory (dotted path)
      --remove-alias string       remove an existing alias at <profile>/<name> (bare <name> for a top-level alias); never removes the real key file
      --replace-material string   replace the key's private key bytes with the contents of <path> (T22)
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

* [hasp edit](hasp_edit.md)	 - Change an existing resource in place: rename, alias, replace material, or relocate

