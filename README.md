# hasp

hasp manages **identity**, not secrets: an accurate, queryable picture of the SSH keys, hosts,
and personas on one laptop, and safe ways to rearrange them. It never takes custody of private
key material, and until you explicitly `adopt` something, hasp only reads — it never writes to
`~/.ssh`.

This project is documentation-first. The design, the reasoning behind it, and the current
milestone all live under [`docs/`](docs/), starting with [`docs/README.md`](docs/README.md). For
where the project actually stands right now, see [`docs/roadmap.md`](docs/roadmap.md) — that
table is the one place milestone status is tracked, so it isn't repeated (and inevitably left
stale) here.

hasp is pre-1.0 and under active development. The command surface is usable today; it is not yet
frozen.

## Install

Once a release is tagged, install the latest with the Go toolchain:

```sh
go install github.com/boweeb/hasp/cmd/hasp@latest
```

Tagged releases also publish prebuilt archives, an SBOM, a cosign signature, and a container
image at `ghcr.io/boweeb/hasp` — see the [releases page](https://github.com/boweeb/hasp/releases)
for the current tag.

### Building from source

```sh
go build ./...
go test ./...
```

Requires the Go toolchain pinned in [`mise.toml`](mise.toml) (`mise install` will fetch it).

## Sixty-second demo

Everything below is read-only — hasp's inventory commands never write anything, on any
`~/.ssh` you point them at. Try it against your own key directory:

```sh
hasp --key-dir ~/.ssh list key
hasp --key-dir ~/.ssh show key <name>
hasp --key-dir ~/.ssh check
```

`list key` inventories every key it finds, managed and unmanaged alike. `show key <name>` gives
full detail on one of them — identity, locations, profiles, and bound hosts. `check` reports
untidiness (duplicates, missing public halves, orphans, dangling references) without fixing
anything; fixing is a separate, explicit step (`adopt`, then `edit`) once you decide you want it.

For the full command surface — every subcommand and flag, kept byte-identical to the binary by
CI — see the generated reference starting at [`docs/cli/hasp.md`](docs/cli/hasp.md), or run
`hasp help`. For a guide to the rest of what hasp can do, organized around real workflows, see
[`docs/user-guide.md`](docs/user-guide.md). For what hasp leaves behind on disk, see
[`docs/on-disk.md`](docs/on-disk.md).

## License

MIT — see [`LICENSE`](LICENSE).
