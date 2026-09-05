## hasp release

Move a managed resource back out of managed territory (inverse of adopt)

### Options

```
  -h, --help   help for release
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

* [hasp](hasp.md)	 - hasp manages SSH identity: keys, hosts, and profiles on one laptop.
* [hasp release host](hasp_release_host.md)	 - Re-insert a managed Host stanza as plain text after hasp's marked region (D14, D18)
* [hasp release key](hasp_release_key.md)	 - Move a managed key back out of its profile directory (inverse of adopt, D14)
* [hasp release profile](hasp_release_profile.md)	 - Remove a profile directory's .hasp marker, showing its contents first (D15, D18)

