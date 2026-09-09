package cmd

import (
	"testing"

	"github.com/yareeh/bibdb/internal"
	"github.com/yareeh/bibdb/internal/vocab"
)

func TestScanCoverage(t *testing.T) {
	v, err := vocab.Parse([]byte(`
scheme:
  facets: [top, topic, place]
  defaultFacet: topic
concepts:
  - {id: social-sciences, prefLabel: social sciences, facet: top}
  - {id: iran, prefLabel: Iran, facet: place, altLabel: [Venäjä]}
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	entries := []*internal.Entry{
		{Key: "a", Fields: []internal.Field{{Name: "keywords", Value: "social sciences, welfare policy, Iran"}}},
		{Key: "b", Fields: []internal.Field{{Name: "keywords", Value: "welfare policy, welfare policy, Christopher Nolan"}}},
		{Key: "c", Fields: []internal.Field{{Name: "keywords", Value: "welfare policy"}}},
	}

	stats := scanCoverage(v, entries, 2)
	// "social sciences" and "Iran" are curated → excluded. "welfare policy" is
	// uncurated, in 3 entries (counted once per entry despite the dup in b).
	// "Christopher Nolan" is uncurated but only in 1 entry → below min-count 2.
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d: %+v", len(stats), stats)
	}
	if stats[0].Term != "welfare policy" || stats[0].Count != 3 {
		t.Errorf("got %+v, want welfare policy count 3", stats[0])
	}

	// min-count 1 surfaces the one-off too, ranked below the frequent term.
	all := scanCoverage(v, entries, 1)
	if len(all) != 2 || all[0].Term != "welfare policy" || all[1].Term != "Christopher Nolan" {
		t.Errorf("unexpected ranking: %+v", all)
	}
}
