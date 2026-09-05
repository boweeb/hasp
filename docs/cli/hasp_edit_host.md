## hasp edit host

Change directives, move between host groups, or rebind a managed Host stanza

```
hasp edit host <pattern> [flags]
```

### Options

```
      --group string      move the stanza to this host group (an explicitly passed empty string moves it to the default group, ~/.ssh/config itself)
  -h, --help              help for host
      --hostname string   set the HostName directive's value
      --key string        rebind: point IdentityFile at this key (name or clue)
      --port string       set the Port directive's value
      --remove-hostname   remove the HostName directive if present
      --remove-port       remove the Port directive if present
      --remove-user       remove the User directive if present
      --unbind            rebind: remove the IdentityFile directive if present
      --user string       set the User directive's value
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

