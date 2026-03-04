package internal

import (
	"strings"
	"testing"
)

func TestFormatEntry(t *testing.T) {
	e := &Entry{
		Type: "book",
		Key:  "smith2019spring",
		Fields: []Field{
			{Name: "author", Value: "Smith, Ali"},
			{Name: "title", Value: "Spring"},
			{Name: "year", Value: "2019"},
		},
	}

	got := FormatEntry(e)
	want := `@book{smith2019spring,
  author = {Smith, Ali},
  title = {Spring},
  year = {2019}
}
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	input := `@article{test2020,
  author = {Doe, John},
  title = {A {Test} Title},
  year = {2020}
}
`
	entries, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	output := FormatEntry(&entries[0])

	// Parse again
	entries2, err := Parse(output)
	if err != nil {
		t.Fatal(err)
	}
	if entries2[0].Key != entries[0].Key {
		t.Errorf("key mismatch: %q vs %q", entries2[0].Key, entries[0].Key)
	}
	if entries2[0].Get("title") != entries[0].Get("title") {
		t.Errorf("title mismatch: %q vs %q", entries2[0].Get("title"), entries[0].Get("title"))
	}
}

func TestFormatEntryPreservesOrder(t *testing.T) {
	e := &Entry{
		Type: "book",
		Key:  "test",
		Fields: []Field{
			{Name: "zebra", Value: "last"},
			{Name: "alpha", Value: "first"},
		},
	}
	got := FormatEntry(e)
	zebraIdx := strings.Index(got, "zebra")
	alphaIdx := strings.Index(got, "alpha")
	if zebraIdx > alphaIdx {
		t.Error("field order not preserved")
	}
}
