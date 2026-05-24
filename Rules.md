# bibdb fix rules

This document is generated from the rule registry in `internal/fixrules/`.
Regenerate with `bibdb fix --list-rules --markdown > Rules.md`. A unit test
(`TestRulesMdInSync`) keeps it in sync with the registered rules.

**Stability**: rule IDs are an API once shipped. Deprecate, never repurpose.

| ID | Since | Severity | Description |
|---|---|---|---|
| `key-format` | 1.4.0 | report | Citation key matches ^[a-z][a-z0-9_]*$ — lowercase, alphanumeric and underscores only, leading letter. |
| `keywords-charset` | 1.4.0 | auto-fix | Strip Obsidian-unsafe punctuation from individual keywords (apostrophes, periods, ampersands, parentheses, …). |
| `newspaper-iso-date` | 1.4.0 | report | Newspaper @article entries carry an ISO date (YYYY-MM-DD) in the number field. |
| `required-fields` | 1.4.0 | report | author, title, year, month, keywords, and abstract must all be present and non-empty. |
| `top-level-keyword` | 1.4.0 | report | Keywords include at least one top-level category: computer science, philosophy, psychology, religion, social sciences, language, pure science, technology, arts, recreation, literature, history, geography. |
| `tracking-params` | 1.4.0 | auto-fix | Strip utm_*, fbclid, gclid tracking parameters from the url field. |
| `utf8-encoding` | 1.4.0 | auto-fix | Text fields use proper UTF-8 — decode HTML entities (&auml;, &#228;, &amp;) and LaTeX accent macros ({\"o}, \^a, \'e) into Unicode characters. |
| `valid-entry-type` | 1.4.0 | report | Entry type must be one of: article, book, inproceedings, misc, online, techreport. |
| `valid-month` | 1.4.0 | auto-fix | Month is a full English name (case-canonicalised; common abbreviations expanded). |

## Discipline

When a new bibdb release changes the criteria for a well-formed entry —
new required field, stricter character set, additional auto-cleanup — add a
corresponding `Rule` in `internal/fixrules/` with `Since = <next release>`.
Re-running `bibdb fix --all` then brings every legacy entry up to date.
