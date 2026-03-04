# bibdb

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
| `bibdb export [key] [--format md\|bib] [--output path]` | Export as markdown or .bib |
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

## Git Sync

Mutations (add/rename/remove/import) automatically:
1. Pull with rebase and autostash
2. Commit changes
3. Push (retry once if remote is ahead)

Read-only operations (get/list/search) don't auto-sync. Use `bibdb sync` to manually sync.
