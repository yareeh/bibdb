# bibdb

Git-backed BibTeX database CLI tool.

## Build

```
go build -o bibdb .
```

## Test

```
go test ./...
```

## Architecture

- `cmd/` — Cobra commands (init, import, add, get, list, search, rename, remove, export, sync, backends)
- `internal/` — Core logic (parser, writer, store, repo, markdown, config, models)
- Data repos are separate git repos configured as named backends in `~/.config/bibdb/config.yaml`
- Entries sharded by first 2 lowercase chars of cite key into `entries/<shard>/<key>.bib`

## Releasing

```bash
git tag v0.1.0
git push origin v0.1.0
gh release create v0.1.0 --generate-notes
```

GoReleaser handles cross-compilation and Homebrew tap publishing.
