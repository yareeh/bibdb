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

- `cmd/` — Cobra commands (init, import, add, get, list, search, rename, remove, export, sync, backends, fix)
- `internal/` — Core logic (parser, writer, store, repo, markdown, config, models)
- `internal/version/` — `Constant`, `Current()`, `GTE()` — bumped on every release.
- `internal/fixrules/` — Rule registry + runner used by `bibdb fix`. Each rule is its own file; rules register from `init()`.
- Data repos are separate git repos configured as named backends in `~/.config/bibdb/config.yaml`
- Entries sharded by first 2 lowercase chars of cite key into `entries/<shard>/<key>.bib`
- Every entry carries a `bibdbversion` field stamped on creation (in `add`/`import`) and updated after `fix`. It tells the fix runner which rules an entry has already been certified against.

## Fix-rule discipline

**Every time a feature or bug fix in bibdb changes how a valid entry should look, add a corresponding rule to `internal/fixrules/`** with `Since = <next release>`. Then:

1. Regenerate the docs: `go run . fix --list-rules --markdown > Rules.md`
2. Bump `internal/version/version.go` `Constant` on release.
3. `TestRulesMdInSync` will trip if you forget step 1.

This way `bibdb fix --all` brings every legacy entry up to date when users upgrade.

## Releasing

```bash
git tag v0.1.0
git push origin v0.1.0
gh release create v0.1.0 --generate-notes
```

GoReleaser handles cross-compilation and Homebrew tap publishing.

## Development workflow (TDD)

For every new feature or bug fix:
1. **Write tests first**
2. **Implement** the feature
3. **Lint and format**
4. **Run tests**
5. Fix any issues and repeat until clean
