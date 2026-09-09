package internal

import (
	"strings"
	"testing"

	"github.com/yareeh/bibdb/internal/vocab"
)

func TestFormatMarkdown(t *testing.T) {
	e := &Entry{
		Type: "book",
		Key:  "smith2019spring",
		Fields: []Field{
			{Name: "author", Value: "Smith, Ali"},
			{Name: "title", Value: "Spring"},
			{Name: "year", Value: "2019"},
			{Name: "month", Value: "March"},
			{Name: "publisher", Value: "Penguin"},
			{Name: "keywords", Value: "fiction, British"},
			{Name: "abstract", Value: "A novel about spring."},
			{Name: "url", Value: "https://example.com"},
		},
	}

	md := FormatMarkdown(e)

	checks := []string{
		"# Smith, Ali: Spring",
		"**Key:** smith2019spring",
		"**Type:** book",
		"**Year:** 2019",
		"**Month:** March",
		"#person/ali-smith",
		"## Keywords",
		"#fiction #british",
		"| publisher | Penguin |",
		"## Abstract",
		"A novel about spring.",
		"## Links",
		"[URL](https://example.com)",
		"```bibtex",
	}

	for _, want := range checks {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in markdown output:\n%s", want, md)
		}
	}
}

func TestFormatMarkdownStripsBraces(t *testing.T) {
	e := &Entry{
		Type: "article",
		Key:  "willingham2015scientific",
		Fields: []Field{
			{Name: "author", Value: "{Wikipedia contributors}"},
			{Name: "title", Value: "The {Scientific} {Status} of {Learning} {Styles}"},
			{Name: "year", Value: "2015"},
			{Name: "howpublished", Value: "Wikipedia, {The} Free Encyclopedia"},
			{Name: "abstract", Value: "Theories of {learning} styles."},
		},
	}

	md := FormatMarkdown(e)

	// Rendered text should have no braces
	mustContain := []string{
		"# Wikipedia contributors: The Scientific Status of Learning Styles",
		"#org/wikipedia-contributors",
		"| howpublished | Wikipedia, The Free Encyclopedia |",
		"Theories of learning styles.",
	}
	for _, want := range mustContain {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in markdown output:\n%s", want, md)
		}
	}

	// But the BibTeX block should still have braces
	if !strings.Contains(md, "{Wikipedia contributors}") {
		t.Errorf("BibTeX block should preserve braces:\n%s", md)
	}
}

func TestFormatMarkdownMultipleAuthors(t *testing.T) {
	e := &Entry{
		Type: "book",
		Key:  "test2024",
		Fields: []Field{
			{Name: "author", Value: "Blas, Javier and Farchy, Jack"},
			{Name: "title", Value: "The World for Sale"},
			{Name: "year", Value: "2024"},
		},
	}

	md := FormatMarkdown(e)

	checks := []string{
		"#person/javier-blas #person/jack-farchy",
	}
	for _, want := range checks {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in markdown output:\n%s", want, md)
		}
	}
}

func TestToTagStripsObsidianUnsafeChars(t *testing.T) {
	// Obsidian tags accept letters, digits, '-', '_', '/'. Any other punctuation
	// truncates the tag at the bad char when rendered, so toTag must strip them.
	cases := []struct{ in, want string }{
		{"children's literature", "#childrens-literature"},
		{"hope and mortality", "#hope-and-mortality"},
		{"social sciences", "#social-sciences"},
		{"Hendrix, Jimi", "#hendrix-jimi"},
		{"hanna mahlamäki", "#hanna-mahlamäki"},
		{"C++ programming", "#c-programming"},
		{"R&D notes", "#rd-notes"},
		{"AI/ML", "#ai/ml"},
		{"some.thing.dotted", "#somethingdotted"},
		{"  leading and trailing  ", "#leading-and-trailing"},
		{"double--dash", "#double-dash"},
	}
	for _, c := range cases {
		got := toTag(c.in)
		if got != c.want {
			t.Errorf("toTag(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatMarkdownDedupsTags(t *testing.T) {
	// Distinct keywords that normalize to the same tag must not repeat it.
	// Mr T, his real name and his A-Team character all map to #person/mr-t.
	yaml := `
scheme:
  facets: [top, topic, person]
  defaultFacet: topic
concepts:
  - {id: mr-t, prefLabel: Mr T, facet: person, altLabel: [Lawrence Tureaud, BA Baracus]}
`
	v, err := vocab.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parse vocab: %v", err)
	}
	vocab.SetActive(v)
	defer vocab.SetActive(nil)

	e := &Entry{
		Type: "article",
		Key:  "hattenstone2026idont",
		Fields: []Field{
			{Name: "author", Value: "Hattenstone, Simon"},
			{Name: "title", Value: "The hidden life of Mr T"},
			{Name: "keywords", Value: "Mr T, Lawrence Tureaud, BA Baracus, personal narrative"},
		},
	}

	md := FormatMarkdown(e)
	if n := strings.Count(md, "#person/mr-t"); n != 1 {
		t.Errorf("expected #person/mr-t exactly once, got %d\n%s", n, md)
	}
	if !strings.Contains(md, "#personal-narrative") {
		t.Errorf("distinct keyword should still be tagged:\n%s", md)
	}
}

func TestKeywordTags(t *testing.T) {
	yaml := `
scheme:
  facets: [top, topic, place, person]
  defaultFacet: topic
concepts:
  - {id: social-sciences, prefLabel: social sciences, facet: top}
  - {id: iran, prefLabel: Iran, facet: place}
  - {id: mr-t, prefLabel: Mr T, facet: person, altLabel: [Lawrence Tureaud, BA Baracus]}
`
	v, err := vocab.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parse vocab: %v", err)
	}

	got := KeywordTags(v, "social sciences, Mr T, Lawrence Tureaud, Iran, some unknown thing")
	want := []string{"#social-sciences", "#person/mr-t", "#place/iran", "#some-unknown-thing"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("KeywordTags = %v, want %v", got, want)
	}

	// Without a vocabulary, keywords are tagged verbatim (still deduped).
	got = KeywordTags(nil, "Fiction, fiction, British")
	if strings.Join(got, " ") != "#fiction #british" {
		t.Errorf("KeywordTags(nil) = %v, want [#fiction #british]", got)
	}

	// Empty and whitespace-only inputs yield no tags.
	if tags := KeywordTags(v, " , ,"); len(tags) != 0 {
		t.Errorf("expected no tags, got %v", tags)
	}
}

func TestAuthorTag(t *testing.T) {
	v, err := vocab.Parse([]byte(`
scheme:
  facets: [top, person, org]
  defaultFacet: topic
concepts:
  - {id: donald-trump, prefLabel: Donald Trump, facet: person, altLabel: [Trump]}
  - {id: anthropic, prefLabel: Anthropic, facet: org}
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cases := []struct {
		name   string
		author string
		vocab  *vocab.Vocabulary
		want   string
	}{
		// --- persons: comma form, reordered "Family, Given" -> "Given Family" ---
		{"comma person", "Hendrix, Jimi", nil, "#person/jimi-hendrix"},
		{"comma person 2", "Blas, Javier", nil, "#person/javier-blas"},
		// --- persons: no-comma "Given Family" (the 866-author majority) ---
		{"nocomma person", "Simon Hattenstone", nil, "#person/simon-hattenstone"},
		{"nocomma person 2", "Eshe Nelson", nil, "#person/eshe-nelson"},
		{"three-part name", "Juha-Pekka Raeste", nil, "#person/juha-pekka-raeste"},
		// --- FAILURE FIXED: "Lehti" is a Finnish surname, not the outlet word ---
		{"surname Lehti not org", "Anu-Elina Lehti", nil, "#person/anu-elina-lehti"},
		// --- orgs: outlet/corporate bylines -> #org/ (not #person/) ---
		{"org WSJ", "Wall Street Journal", nil, "#org/wall-street-journal"},
		{"org Guardian", "The Guardian", nil, "#org/the-guardian"},
		{"org Le Monde", "Le Monde", nil, "#org/le-monde"},
		{"org Bloomberg News", "Bloomberg News", nil, "#org/bloomberg-news"},
		{"org Big Think", "Big Think", nil, "#org/big-think"},
		{"org HS", "Helsingin Sanomat", nil, "#org/helsingin-sanomat"},
		{"org Wikipedia", "Wikipedia contributors", nil, "#org/wikipedia-contributors"},
		// --- junk bylines -> no tag ---
		{"junk parenthetical", "(No specific author listed in the provided text)", nil, ""},
		{"empty", "", nil, ""},
		// --- single word, uncurated, no signal -> flat (can't tell person vs org) ---
		{"single unknown", "Pyhimys", nil, "#pyhimys"},
		// --- curated resolves to canonical facet regardless of form ---
		{"curated person comma", "Trump, Donald", v, "#person/donald-trump"},
		{"curated org", "Anthropic", v, "#org/anthropic"},
	}
	for _, c := range cases {
		if got := authorTag(c.vocab, c.author); got != c.want {
			t.Errorf("%s: authorTag(%q) = %q, want %q", c.name, c.author, got, c.want)
		}
	}
}

func TestWikiLink(t *testing.T) {
	cases := []struct{ in, want string }{
		{"The Guardian", "[[The Guardian]]"},
		{"Le Monde", "[[Le Monde]]"},
		{"", ""},
		{"  ", ""},
		{"Weird [Name] #x |y", "[[Weird Name x y]]"}, // link-breaking chars stripped
	}
	for _, c := range cases {
		if got := wikiLink(c.in); got != c.want {
			t.Errorf("wikiLink(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatMarkdownRelatedSection(t *testing.T) {
	v, err := vocab.Parse([]byte(`
scheme:
  facets: [top, person, org]
  defaultFacet: topic
concepts:
  - {id: donald-trump, prefLabel: Donald Trump, facet: person, altLabel: [Trump]}
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	vocab.SetActive(v)
	defer vocab.SetActive(nil)

	e := &Entry{
		Type: "article",
		Key:  "hattenstone2026idont",
		Fields: []Field{
			{Name: "author", Value: "Hattenstone, Simon"},
			{Name: "title", Value: "The hidden life of Mr T"},
			{Name: "journal", Value: "The Guardian"},
			{Name: "keywords", Value: "social sciences"},
		},
	}
	md := FormatMarkdown(e)

	// Entity facet tags (line below title): author person + publication org.
	if !strings.Contains(md, "#person/simon-hattenstone") {
		t.Errorf("missing author facet tag:\n%s", md)
	}
	if !strings.Contains(md, "#org/the-guardian") {
		t.Errorf("missing publication facet tag:\n%s", md)
	}
	// ## Related wiki-links for backlink navigation.
	if !strings.Contains(md, "## Related") {
		t.Errorf("missing Related section:\n%s", md)
	}
	if !strings.Contains(md, "[[Simon Hattenstone]]") {
		t.Errorf("missing author wiki-link (reordered):\n%s", md)
	}
	if !strings.Contains(md, "[[The Guardian]]") {
		t.Errorf("missing publication wiki-link:\n%s", md)
	}
}

func TestFormatMarkdownCuratedEntityLinkAndDedup(t *testing.T) {
	v, err := vocab.Parse([]byte(`
scheme:
  facets: [top, person, org]
  defaultFacet: topic
concepts:
  - {id: donald-trump, prefLabel: Donald Trump, facet: person, altLabel: [Trump]}
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	vocab.SetActive(v)
	defer vocab.SetActive(nil)

	// Curated author "Trump, Donald" → canonical [[Donald Trump]] link + tag.
	// Publisher equals a repeated org → deduped in both tags and links.
	e := &Entry{
		Type: "book",
		Key:  "trump2024",
		Fields: []Field{
			{Name: "author", Value: "Trump, Donald"},
			{Name: "title", Value: "A Book"},
			{Name: "publisher", Value: "Penguin"},
			{Name: "institution", Value: "Penguin"},
		},
	}
	md := FormatMarkdown(e)
	if !strings.Contains(md, "[[Donald Trump]]") || !strings.Contains(md, "#person/donald-trump") {
		t.Errorf("curated author should link/tag canonically:\n%s", md)
	}
	if n := strings.Count(md, "[[Penguin]]"); n != 1 {
		t.Errorf("repeated org should dedup to one link, got %d:\n%s", n, md)
	}
	if n := strings.Count(md, "#org/penguin"); n != 1 {
		t.Errorf("repeated org should dedup to one tag, got %d:\n%s", n, md)
	}
}

func TestEntityLinkNames(t *testing.T) {
	v, err := vocab.Parse([]byte(`
scheme:
  facets: [top, person, org]
  defaultFacet: topic
concepts:
  - {id: donald-trump, prefLabel: Donald Trump, facet: person, altLabel: [Trump]}
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	e := &Entry{
		Type: "article",
		Key:  "x",
		Fields: []Field{
			{Name: "author", Value: "Trump, Donald and (No specific author listed)"},
			{Name: "journal", Value: "The Guardian"},
			{Name: "publisher", Value: "The Guardian"}, // dup with journal → collapsed
		},
	}
	got := EntityLinkNames(v, e)
	want := []string{"Donald Trump", "The Guardian"} // junk byline skipped, dup collapsed
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("EntityLinkNames = %v, want %v", got, want)
	}
	// Names must match the [[link]] text used in the note (so stubs resolve).
	md := FormatMarkdown(func() *Entry { vocab.SetActive(v); return e }())
	vocab.SetActive(nil)
	for _, n := range got {
		if !strings.Contains(md, "[["+n+"]]") {
			t.Errorf("EntityLinkNames %q not present as [[link]] in note:\n%s", n, md)
		}
	}
}
