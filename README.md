# hasp

hasp manages **identity**, not secrets: an accurate, queryable picture of the SSH keys, hosts,
and personas on one laptop, and safe ways to rearrange them.

This project is documentation-first — see [`docs/`](docs/) for the full design, starting with
[`docs/README.md`](docs/README.md). The Go implementation begins at
[`docs/roadmap.md`](docs/roadmap.md) §2 (M0 — Bootstrap); this is that code.

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

Bootstrap only (M0). No user-facing commands exist yet beyond `hasp version`.
