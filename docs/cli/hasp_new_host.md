## hasp new host

Create a new Host stanza

```
hasp new host <pattern...> [flags]
```

### Options

```
      --group string      the host group to create this stanza in (default: ~/.ssh/config itself)
  -h, --help              help for host
      --hostname string   the HostName directive's value
      --key string        bind this host to a key (name or clue); omit for no explicit binding
      --port string       the Port directive's value
      --user string       the User directive's value
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

