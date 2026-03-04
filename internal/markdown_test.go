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
		"# smith2019spring",
		"## Smith, Ali: Spring",
		"**Type:** book",
		"**Year:** 2019",
		"**Month:** March",
		"**Keywords:** fiction, British",
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
