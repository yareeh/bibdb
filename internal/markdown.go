package internal

import (
	"fmt"
	"regexp"
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

// authorOrgRe matches outlet / corporate signal words in a byline, so an
// organization-as-author (news outlets, wire services, institutions) is faceted
// #org/ rather than mislabeled a person. Deliberately excludes role words
// (editor, staff, reporter) and surnames that collide with outlet words (e.g.
// the Finnish surname "Lehti") — those are handled as persons.
var authorOrgRe = regexp.MustCompile(`(?i)\b(times|news|post|press|media|agency|university|institute|sanomat|uutiset|monde|bloomberg|reuters|guardian|bbc|cnn|nbc|vox|axios|politico|magazine|journal|newsroom|alphaville|substack|wikipedia|encyclopedia|wire|ministry|department|council|foundation|association|committee|company|corp|inc|ltd|llc|studios?|productions?|records)\b|wall\s+street|big\s+think|prof\s+g|\bft\b`)

// authorJunkRe matches non-byline noise ("(No specific author listed…)",
// strings with digits or parentheses) that should not become a tag at all.
var authorJunkRe = regexp.MustCompile(`(?i)[(\d]|^no |listed|provided|unknown|specific author`)

// authorTag renders one BibTeX author name into a facet tag. BibTeX stores
// authors as "Family, Given"; we reorder to "Given Family" so the slug reads
// naturally and matches the person-concept style (#person/jimi-hendrix). A
// curated author resolves through the vocabulary to its canonical facet.
// Otherwise: an outlet/corporate byline → #org/; a comma-form name or a
// multi-word "Given Family" → #person/; a bare single word we can't classify
// stays a flat tag; junk yields no tag.
func authorTag(v *vocab.Vocabulary, author string) string {
	name := strings.TrimSpace(author)
	if name == "" || authorJunkRe.MatchString(name) {
		return ""
	}
	hasComma := strings.Contains(name, ",")
	display := name
	if i := strings.Index(name, ","); i >= 0 {
		family := strings.TrimSpace(name[:i])
		given := strings.TrimSpace(name[i+1:])
		if family != "" && given != "" {
			display = given + " " + family
		}
	}
	if v != nil {
		if _, ok := v.Lookup(display); ok {
			if t := toTag(v.TagPath(display)); t != "#" {
				return t // curated → canonical facet tag
			}
		}
	}
	slug := toTag(display)
	if slug == "#" {
		return ""
	}
	body := slug[1:]
	if authorOrgRe.MatchString(name) {
		return "#org/" + body
	}
	if hasComma || len(strings.Fields(display)) >= 2 {
		return "#person/" + body
	}
	return slug // single-word, uncurated, no signal → flat
}

// orgFacetTag renders a publication/publisher name as an #org/ facet tag,
// resolving through the vocabulary to a curated concept's canonical tag when
// known. Returns "" for an empty name.
func orgFacetTag(v *vocab.Vocabulary, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if v != nil {
		if _, ok := v.Lookup(name); ok {
			if t := toTag(v.TagPath(name)); t != "#" {
				return t
			}
		}
	}
	s := toTag(name)
	if s == "#" {
		return ""
	}
	return "#org/" + s[1:]
}

// entityLinkName returns the display name for an entity's [[wiki-link]]: the
// curated prefLabel when known, else the raw name. Author names ("Family,
// Given") are reordered to "Given Family" when reorderComma is set.
func entityLinkName(v *vocab.Vocabulary, raw string, reorderComma bool) string {
	name := strings.TrimSpace(stripBraces(raw))
	if reorderComma {
		if i := strings.Index(name, ","); i >= 0 {
			family := strings.TrimSpace(name[:i])
			given := strings.TrimSpace(name[i+1:])
			if family != "" && given != "" {
				name = given + " " + family
			}
		}
	}
	if v != nil {
		if canon, ok := v.Canonical(name); ok {
			name = canon
		}
	}
	return name
}

// wikiLinkReplacer strips the characters Obsidian would treat as link syntax.
var wikiLinkReplacer = strings.NewReplacer("[", "", "]", "", "#", "", "^", "", "|", "", "\n", " ", "\r", " ")

// wikiLink wraps a display name as an Obsidian [[wiki-link]]. Returns "" empty.
func wikiLink(name string) string {
	name = strings.TrimSpace(wikiLinkReplacer.Replace(name))
	if name == "" {
		return ""
	}
	return "[[" + name + "]]"
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

	// Bibliographic entities — authors, publication, publisher — are emitted
	// both as facet tags (line below the title) and as [[wiki-links]] in a
	// ## Related section. The tag converges roles (an entity that is an author
	// here and a subject keyword elsewhere shares one #person/ or #org/ tag);
	// the link gives Obsidian backlinks + a graph node for "all items by/from
	// this entity". Collected here, tags rendered now, links after the table.
	v := vocab.Active()
	var entTags, entLinks []string
	seenTag := map[string]bool{}
	seenLink := map[string]bool{}
	addEntity := func(tag, link string) {
		if tag != "" && !seenTag[tag] {
			seenTag[tag] = true
			entTags = append(entTags, tag)
		}
		if link != "" && !seenLink[link] {
			seenLink[link] = true
			entLinks = append(entLinks, link)
		}
	}
	if author != "" {
		for _, a := range strings.Split(author, " and ") {
			if strings.TrimSpace(a) == "" {
				continue
			}
			if tag := authorTag(v, a); tag != "" {
				addEntity(tag, wikiLink(entityLinkName(v, a, true)))
			}
		}
	}
	for _, fn := range []string{"journal", "booktitle"} { // publications
		if val := stripBraces(e.Get(fn)); val != "" {
			addEntity(orgFacetTag(v, val), wikiLink(entityLinkName(v, val, false)))
		}
	}
	for _, fn := range []string{"publisher", "institution", "organization", "school"} {
		if val := stripBraces(e.Get(fn)); val != "" {
			addEntity(orgFacetTag(v, val), wikiLink(entityLinkName(v, val, false)))
		}
	}
	if len(entTags) > 0 {
		b.WriteString(strings.Join(entTags, " "))
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

	// Wiki-links to the bibliographic entities → Obsidian backlinks give
	// "all items by/from this author/publication/publisher".
	if len(entLinks) > 0 {
		b.WriteString("## Related\n\n")
		b.WriteString(strings.Join(entLinks, " · "))
		b.WriteString("\n\n")
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
