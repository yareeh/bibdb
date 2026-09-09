package internal

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/yareeh/bibdb/internal/vocab"
)

// toTag converts a string to a hashtag format safe for Obsidian.
//
// Obsidian accepts only letters, digits, '-', '_', and '/' (for nested tags).
// Any other punctuation truncates the tag in the rendered note, so we strip
// it. Spaces become '-' first; '/' is preserved for nesting; runs of '-' are
// collapsed.
//
// "personal development"     -> "#personal-development"
// "Hendrix, Jimi"            -> "#hendrix-jimi"
// "children's literature"    -> "#childrens-literature"
// "AI/ML"                    -> "#ai/ml"
func toTag(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")

	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '/' {
			b.WriteRune(r)
		}
	}
	result := b.String()
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	result = strings.Trim(result, "-")
	return "#" + result
}

// KeywordTags renders a comma-separated keywords string into the ordered,
// de-duplicated Obsidian hashtag run. With a vocabulary, entity facets get a
// "<facet>/" prefix (#place/iran) so places, people, orgs and genres become
// nested tags and stay out of the topic tag tree; without one, each keyword is
// tagged verbatim (pre-taxonomy behavior). Distinct keywords that normalize to
// the same tag (e.g. "Mr T" and "Lawrence Tureaud" → #person/mr-t) collapse to
// one, first occurrence kept. Shared by markdown export and `bibdb vocab tags`.
func KeywordTags(v *vocab.Vocabulary, keywords string) []string {
	var tags []string
	seen := make(map[string]bool)
	for _, p := range strings.Split(keywords, ",") {
		if strings.TrimSpace(p) == "" {
			continue
		}
		var tag string
		if v != nil {
			tag = toTag(v.TagPath(p))
		} else {
			tag = toTag(p)
		}
		if tag == "#" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	return tags
}

// authorTag renders one BibTeX author name into a tag. BibTeX stores authors
// as "Family, Given"; we reorder to "Given Family" so the slug reads naturally
// and matches the person-concept style (#person/jimi-hendrix). A curated author
// resolves through the vocabulary to its canonical facet (person or org). A
// comma-form name is unambiguously personal, so it gets the #person/ facet;
// a comma-less author (often an organization byline like "Le Monde") keeps a
// flat tag rather than being mislabeled a person.
func authorTag(v *vocab.Vocabulary, author string) string {
	name := strings.TrimSpace(author)
	if name == "" {
		return ""
	}
	hasComma := strings.Contains(name, ",")
	if i := strings.Index(name, ","); i >= 0 {
		family := strings.TrimSpace(name[:i])
		given := strings.TrimSpace(name[i+1:])
		if family != "" && given != "" {
			name = given + " " + family
		}
	}
	if v != nil {
		if _, ok := v.Lookup(name); ok {
			return toTag(v.TagPath(name)) // curated → canonical facet tag
		}
	}
	slug := toTag(name)
	if slug == "#" {
		return ""
	}
	if hasComma {
		return "#person/" + slug[1:]
	}
	return slug
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
		v := vocab.Active()
		var atags []string
		seen := make(map[string]bool)
		for _, a := range strings.Split(author, " and ") {
			tag := authorTag(v, a)
			if tag == "" || seen[tag] {
				continue
			}
			seen[tag] = true
			atags = append(atags, tag)
		}
		if len(atags) > 0 {
			b.WriteString(strings.Join(atags, " "))
			b.WriteString("\n\n")
		}
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
		if tags := KeywordTags(vocab.Active(), kw); len(tags) > 0 {
			b.WriteString("## Keywords\n\n")
			b.WriteString(strings.Join(tags, " "))
			b.WriteString("\n\n")
		}
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
