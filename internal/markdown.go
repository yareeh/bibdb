package internal

import (
	"fmt"
	"strings"
)

// toTag converts a string to a hashtag format.
// "personal development" -> "#personal-development"
// "Hendrix, Jimi" -> "#hendrix-jimi"
func toTag(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "--", "-")
	s = strings.TrimRight(s, "-")
	return "#" + s
}

// stripBraces removes BibTeX protective braces from display text.
// "{Scientific}" -> "Scientific", "{{Wikipedia contributors}}" -> "Wikipedia contributors"
func stripBraces(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '{' || s[i] == '}' {
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// FormatMarkdown formats an entry as a markdown reference note.
func FormatMarkdown(e *Entry) string {
	var b strings.Builder

	author := stripBraces(e.Get("author"))
	title := stripBraces(e.Get("title"))

	fmt.Fprintf(&b, "# %s: %s\n\n", author, title)
	if author != "" {
		authors := strings.Split(author, " and ")
		for i, a := range authors {
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(toTag(a))
		}
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "**Key:** %s\n", e.Key)
	fmt.Fprintf(&b, "**Type:** %s\n", e.Type)
	if y := e.Get("year"); y != "" {
		fmt.Fprintf(&b, "**Year:** %s\n", y)
	}
	if m := e.Get("month"); m != "" {
		fmt.Fprintf(&b, "**Month:** %s\n", m)
	}
	b.WriteString("\n")

	// Table of other fields
	skipFields := map[string]bool{
		"author": true, "title": true, "year": true, "month": true,
		"keywords": true, "abstract": true, "url": true, "doi": true,
	}

	var tableFields []Field
	for _, f := range e.Fields {
		if !skipFields[strings.ToLower(f.Name)] {
			tableFields = append(tableFields, f)
		}
	}

	if len(tableFields) > 0 {
		b.WriteString("| Field | Value |\n")
		b.WriteString("|-------|-------|\n")
		for _, f := range tableFields {
			fmt.Fprintf(&b, "| %s | %s |\n", f.Name, stripBraces(f.Value))
		}
		b.WriteString("\n")
	}

	if kw := e.Get("keywords"); kw != "" {
		b.WriteString("## Keywords\n\n")
		parts := strings.Split(kw, ",")
		for i, p := range parts {
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(toTag(p))
		}
		b.WriteString("\n\n")
	}

	if abs := e.Get("abstract"); abs != "" {
		fmt.Fprintf(&b, "## Abstract\n\n%s\n\n", stripBraces(abs))
	}

	url := e.Get("url")
	doi := e.Get("doi")
	if url != "" || doi != "" {
		b.WriteString("## Links\n\n")
		if url != "" {
			fmt.Fprintf(&b, "- [URL](%s)\n", url)
		}
		if doi != "" {
			fmt.Fprintf(&b, "- DOI: %s\n", doi)
		}
		b.WriteString("\n")
	}

	b.WriteString("## BibTeX\n\n```bibtex\n")
	b.WriteString(FormatEntry(e))
	b.WriteString("```\n")

	return b.String()
}
