# BibTeX Database Manager

Manage BibTeX entries using the bibdb CLI tool.

## When to Use
- User wants to add a citation to their bibliography
- User wants to search/find/list bibliography entries
- User wants to export bibliography as markdown

## Available Commands
- `bibdb add --type <type> --key <key> --field k=v ...` or pipe BibTeX via stdin
- `bibdb get <key>` — retrieve entry
- `bibdb search <query>` — find entries
- `bibdb list` — list all entries
- `bibdb remove <key> --force` — delete entry
- `bibdb export --format md --output <dir>` — export as markdown
- `bibdb sync` — sync with remote

## Entry ID Format
{authorsurname}{year}{keywordsfromtitle} — e.g., smith2024machinelearning

## Adding via stdin
echo '@book{key, author={...}, ...}' | bibdb add

## Required Fields (warnings only)
- author, title, year
- month — Full month name in English
- keywords — Always in English, comma-separated
- abstract — In the original language of the content
