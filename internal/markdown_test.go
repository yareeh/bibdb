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
		"#wikipedia-contributors",
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
	// Curated: Donald Trump is a person concept; author "Trump, Donald" resolves.
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
	cases := []struct {
		author string
		vocab  *vocab.Vocabulary
		want   string
	}{
		{"Hendrix, Jimi", nil, "#person/jimi-hendrix"},   // comma → person, reordered
		{"Blas, Javier", nil, "#person/javier-blas"},
		{"Le Monde", nil, "#le-monde"},                    // no comma → flat (likely org)
		{"Helsingin Sanomat", nil, "#helsingin-sanomat"},  // org byline stays flat
		{"Trump, Donald", v, "#person/donald-trump"},      // curated → canonical
		{"", nil, ""},
	}
	for _, c := range cases {
		if got := authorTag(c.vocab, c.author); got != c.want {
			t.Errorf("authorTag(%q) = %q, want %q", c.author, got, c.want)
		}
	}
}
