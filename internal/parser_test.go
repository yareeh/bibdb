package internal

import (
	"os"
	"testing"
)

func TestParseSimpleBook(t *testing.T) {
	input := `@book{smith2019spring,
  author = {Smith, Ali},
  title = {Spring},
  year = {2019},
  month = {March}
}`
	entries, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Type != "book" {
		t.Errorf("type = %q, want book", e.Type)
	}
	if e.Key != "smith2019spring" {
		t.Errorf("key = %q, want smith2019spring", e.Key)
	}
	if e.Get("author") != "Smith, Ali" {
		t.Errorf("author = %q", e.Get("author"))
	}
	if e.Get("year") != "2019" {
		t.Errorf("year = %q", e.Get("year"))
	}
	if len(e.Fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(e.Fields))
	}
}

func TestParseQuotedValues(t *testing.T) {
	input := `@article{test2020,
  author = "Doe, John",
  title = "A {Test} Title"
}`
	entries, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	e := entries[0]
	if e.Get("author") != "Doe, John" {
		t.Errorf("author = %q", e.Get("author"))
	}
	if e.Get("title") != "A {Test} Title" {
		t.Errorf("title = %q", e.Get("title"))
	}
}

func TestParseNestedBraces(t *testing.T) {
	input := `@book{test,
  title = {The {Data-Centric} {Revolution}}
}`
	entries, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Get("title") != "The {Data-Centric} {Revolution}" {
		t.Errorf("title = %q", entries[0].Get("title"))
	}
}

func TestParseMultipleEntries(t *testing.T) {
	input := `@book{a, title = {First}}
@article{b, title = {Second}}`
	entries, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Key != "a" {
		t.Errorf("first key = %q", entries[0].Key)
	}
	if entries[1].Key != "b" {
		t.Errorf("second key = %q", entries[1].Key)
	}
}

func TestParseComments(t *testing.T) {
	input := `%% A comment
% Another comment
@book{test, title = {Hello}}`
	entries, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestParseBareNumber(t *testing.T) {
	input := `@book{test, year = 2019, pages = 320}`
	entries, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Get("year") != "2019" {
		t.Errorf("year = %q", entries[0].Get("year"))
	}
}

func TestParseMultilineAbstract(t *testing.T) {
	input := `@book{test,
  abstract = {First line.
Second line.
Third line.}
}`
	entries, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	abs := entries[0].Get("abstract")
	if abs != "First line.\nSecond line.\nThird line." {
		t.Errorf("abstract = %q", abs)
	}
}

func TestParseFieldOrder(t *testing.T) {
	input := `@book{test,
  author = {Z},
  title = {A},
  year = {2020}
}`
	entries, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	fields := entries[0].Fields
	if fields[0].Name != "author" || fields[1].Name != "title" || fields[2].Name != "year" {
		t.Errorf("field order not preserved: %v", fields)
	}
}

func TestParseBackslashInValue(t *testing.T) {
	input := `@book{test,
  publisher = {Faber \& Faber}
}`
	entries, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Get("publisher") != `Faber \& Faber` {
		t.Errorf("publisher = %q", entries[0].Get("publisher"))
	}
}

func TestParseSampleBibFile(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.bib")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := Parse(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 7 {
		t.Errorf("expected 7 entries, got %d", len(entries))
	}

	found := map[string]bool{}
	for _, e := range entries {
		found[e.Key] = true
	}
	for _, key := range []string{"smith2019spring", "knuth1974structured", "adams2002salmon", "blas2021worldforsale"} {
		if !found[key] {
			t.Errorf("missing entry %q", key)
		}
	}
}

func TestValidateKey(t *testing.T) {
	valid := []string{"smith2019spring", "McComb2019", "a", "test-key_2025"}
	for _, k := range valid {
		if err := ValidateKey(k); err != nil {
			t.Errorf("ValidateKey(%q) = %v, want nil", k, err)
		}
	}

	invalid := []string{"", "a:b", "a/b", "a\\b", "a<b", "a>b", "a\"b", "a|b", "a?b", "a*b", "a b", ".", ".."}
	for _, k := range invalid {
		if err := ValidateKey(k); err == nil {
			t.Errorf("ValidateKey(%q) = nil, want error", k)
		}
	}
}
