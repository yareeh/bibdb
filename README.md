# bibdb

[![CI](https://github.com/yareeh/bibdb/actions/workflows/ci.yml/badge.svg)](https://github.com/yareeh/bibdb/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/yareeh/bibdb/graph/badge.svg)](https://codecov.io/gh/yareeh/bibdb)
[![Go](https://img.shields.io/github/go-mod/go-version/yareeh/bibdb)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/yareeh/bibdb)](https://github.com/yareeh/bibdb/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/yareeh/bibdb)](https://goreportcard.com/report/github.com/yareeh/bibdb)
[![Built for AI Agents](https://img.shields.io/badge/built%20for-AI%20agents-blueviolet)](https://github.com/yareeh/bibdb)

Git-backed BibTeX database manager for CLI-first workflows. Entries are stored as individual `.bib` files in a sharded directory structure within a git repository, keeping your bibliography portable, version-controlled, and syncable across machines.

## Installation

**macOS/Linux (Homebrew):**
```bash
brew install yareeh/tap/bibdb
```

**From source:**
```bash
go install github.com/yareeh/bibdb@latest
```

## Quick Start

```bash
# Initialize a data repo
bibdb init ~/bibdata

# Import existing .bib file
bibdb import ~/refs/BibTexDB.bib

# Search and browse
bibdb search "machine learning"
bibdb list --type book --year 2024
bibdb get smith2024machinelearning

# Add entries
bibdb add --type article --key doe2025ai \
  --field author="Doe, Jane" \
  --field title="AI Today" \
  --field year=2025

# Or pipe BibTeX directly
echo '@book{key, author={Smith}, title={Test}, year={2025}}' | bibdb add

# Export
bibdb export --format md --output ./notes/    # markdown reference notes
bibdb export --format bib --output refs.bib   # concatenated .bib

# Sync with remote
bibdb sync
```

## Commands

| Command | Description |
|---------|-------------|
| `bibdb init <path>` | Initialize a new data repo and register as backend |
| `bibdb import <file.bib>` | Import monolithic .bib into current backend |
| `bibdb add [--type T] [--key K] [--field k=v ...]` | Add entry from flags or stdin |
| `bibdb get <key>` | Print entry to stdout |
| `bibdb list [--type X] [--year Y]` | List entries in table format |
| `bibdb search <query> [--field F]` | Search entries (case-insensitive substring) |
| `bibdb rename <old> <new>` | Rename cite key |
| `bibdb remove <key> [--force]` | Delete entry |
| `bibdb export [key] [--format md\|bib] [--output path] [--include-meta]` | Export as markdown or .bib (strips `bibdbversion` unless `--include-meta`) |
| `bibdb fix [key] [--all] [--rule ID] [--dry-run] [-n N] [-v]` | Apply registered fix rules — see [Rules.md](Rules.md) |
| `bibdb sync` | Git pull --rebase + push |
| `bibdb backends` | List configured backends |

## Configuration

Config file: `~/.config/bibdb/config.yaml`

```yaml
default: personal
backends:
  personal:
    path: ~/bibdata
    remote: origin
    branch: main
  work:
    path: ~/work-refs
    remote: origin
    branch: main
```

Select backend: `--backend work` flag or `BIBDB_BACKEND=work` env var.

## Connecting to an Existing Remote

To use a bibdb data repo that already exists on a remote (e.g., a colleague's or your own from another machine):

```bash
# Clone the remote repo
git clone git@github.com:user/bibdata.git ~/bibdata

# Register it as a backend
bibdb init ~/bibdata
```

`bibdb init` detects the existing git repo and data, and registers it as a backend without reinitializing. After this, `bibdb sync` will pull and push to the remote.

You can also add the remote manually to a locally initialized repo:

```bash
bibdb init ~/bibdata
cd ~/bibdata
git remote add origin git@github.com:user/bibdata.git
git push -u origin main
```

## Data Repo Structure

Entries are sharded by first two lowercase characters of the cite key:

```
bibdata/
├── .bibdb.yaml
├── entries/
│   ├── ad/
│   │   └── adams2002salmon.bib
│   ├── sm/
│   │   └── smith2019spring.bib
│   └── ...
```

Each `.bib` file contains exactly one BibTeX entry.

## Fix command and entry version metadata

Every entry created or fixed by bibdb carries a `bibdbversion` field
recording the bibdb release that last touched it. `bibdb fix` walks the
registered rules and either auto-fixes the entry in place (strip tracking
params, decode HTML/LaTeX entities, canonicalise month names, …) or surfaces
issues that need an external repair (missing required fields, keyword-list
without a top-level category, …). The version stamp lets `fix --all` skip
entries already covered by their last fix.

```bash
bibdb fix smith2026foo            # one entry
bibdb fix --all                   # every entry
bibdb fix --all --dry-run         # preview only
bibdb fix --all -n 5 -v           # fix the first 5 affected entries verbosely
bibdb fix --list-rules            # show registered rules
```

See [Rules.md](Rules.md) for the rule catalogue. `bibdb export` strips the
`bibdbversion` field by default; pass `--include-meta` to retain it (useful
when downstream tools like skyebot's rendered reference files want the
provenance).

When bibdb gains a feature or fix that changes how a valid entry should look,
add a corresponding rule to `internal/fixrules/` with `Since = <next
release>` and regenerate `Rules.md`. Re-running `bibdb fix --all` then
upgrades every legacy entry.

## Git Sync

Mutations (add/rename/remove/import) automatically:
1. Pull with rebase and autostash
2. Commit changes
3. Push (retry once if remote is ahead)

Read-only operations (get/list/search) don't auto-sync. Use `bibdb sync` to manually sync.
