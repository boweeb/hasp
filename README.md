# hasp

hasp manages **identity**, not secrets: an accurate, queryable picture of the SSH keys, hosts,
and personas on one laptop, and safe ways to rearrange them.

This project is documentation-first — see [`docs/`](docs/) for the full design, starting with
[`docs/README.md`](docs/README.md). The Go implementation follows
[`docs/roadmap.md`](docs/roadmap.md); this is that code.

## Building

```sh
go build ./...
```

Requires the Go toolchain pinned in [`mise.toml`](mise.toml) (`mise install`).

## Testing

```sh
go test ./...
```

## Status

M1 — Inventory. hasp reads an existing `~/.ssh` and reports on it: `list`/`show`/`find`/`check`
across `key`, `host`, and `profile`. Entirely read-only — hasp never writes to `~/.ssh` at this
milestone (see [`docs/decision-log.md`](docs/decision-log.md#d14)). Writes (`new`, `edit`,
`adopt`, `release`) arrive in M2.
