## hasp new key

Generate a new ed25519 keypair (never overwrites an existing file)

```
hasp new key <name> [flags]
```

### Options

```
  -h, --help               help for key
      --no-passphrase      generate without a passphrase, explicitly
      --passphrase         prompt interactively for a passphrase (no local echo)
      --passphrase-stdin   read the passphrase from one line of stdin
      --profile string     place the key in this profile (dotted path, e.g. work.foobarco)
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

* [hasp new](hasp_new.md)	 - Create a new key, host, or profile

