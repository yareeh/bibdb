package internal

import (
	"strings"
	"testing"
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
		"#smith-ali",
		"**Key:** smith2019spring",
		"**Type:** book",
		"**Year:** 2019",
		"**Month:** March",
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
		"#blas-javier #farchy-jack",
	}
	for _, want := range checks {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in markdown output:\n%s", want, md)
		}
	}
}
