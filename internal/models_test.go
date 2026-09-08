package internal

import "testing"

func TestStampVersionAppendsWhenMissing(t *testing.T) {
	e := &Entry{Key: "x", Type: "misc", Fields: []Field{{Name: "title", Value: "T"}}}
	StampVersion(e, "1.4.0")
	if e.Get("bibdbversion") != "1.4.0" {
		t.Errorf("expected bibdbversion=1.4.0, got %q", e.Get("bibdbversion"))
	}
	if len(e.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(e.Fields))
	}
	if e.Fields[1].Name != "bibdbversion" {
		t.Errorf("expected bibdbversion appended last, got order %+v", e.Fields)
	}
}

func TestStampVersionUpdatesInPlace(t *testing.T) {
	e := &Entry{Key: "x", Type: "misc", Fields: []Field{
		{Name: "title", Value: "T"},
		{Name: "bibdbversion", Value: "1.3.1"},
		{Name: "year", Value: "2026"},
	}}
	StampVersion(e, "1.4.0")
	if got := e.Get("bibdbversion"); got != "1.4.0" {
		t.Errorf("expected 1.4.0, got %q", got)
	}
	// Position must not change — bibdbversion was at index 1 originally
	if len(e.Fields) != 3 || e.Fields[1].Name != "bibdbversion" {
		t.Errorf("expected bibdbversion to stay at index 1, got %+v", e.Fields)
	}
}

func TestStampVersionCaseInsensitive(t *testing.T) {
	// Set is case-insensitive; existing BibDBVersion or BIBDBVERSION must update in place.
	e := &Entry{Key: "x", Type: "misc", Fields: []Field{{Name: "BibDBVersion", Value: "1.3.0"}}}
	StampVersion(e, "1.4.0")
	if got := e.Get("bibdbversion"); got != "1.4.0" {
		t.Errorf("expected 1.4.0, got %q", got)
	}
	if len(e.Fields) != 1 {
		t.Errorf("expected exactly 1 field (case-insensitive update), got %d", len(e.Fields))
	}
}

func TestEntryVersionDefaultsToZero(t *testing.T) {
	e := &Entry{Key: "x", Type: "misc"}
	if got := EntryVersion(e); got != "0.0.0" {
		t.Errorf("expected 0.0.0 for missing field, got %q", got)
	}
}

func TestEntryVersionReturnsStored(t *testing.T) {
	e := &Entry{Key: "x", Type: "misc", Fields: []Field{{Name: "bibdbversion", Value: "1.4.0"}}}
	if got := EntryVersion(e); got != "1.4.0" {
		t.Errorf("expected 1.4.0, got %q", got)
	}
}

func TestValidateKeyFormat(t *testing.T) {
	valid := []string{"smith2024", "a", "jager2026foo", "x_y_2"}
	for _, k := range valid {
		if err := ValidateKeyFormat(k); err != nil {
			t.Errorf("ValidateKeyFormat(%q) = %v, want nil", k, err)
		}
	}
	invalid := []string{"", "Smith2024", "jäger2026", "2024smith", "a-b", "töyra"}
	for _, k := range invalid {
		if err := ValidateKeyFormat(k); err == nil {
			t.Errorf("ValidateKeyFormat(%q) = nil, want error", k)
		}
	}
}

func TestShardIsRuneSafe(t *testing.T) {
	cases := map[string]string{
		"smith2024": "sm",
		"McCarthy":  "mc", // lowercased
		"a":         "a_", // padded
		"":          "__",
		"jäger2026": "jä", // rune-based: valid UTF-8, not a split byte
		"töyra":     "tö",
	}
	for in, want := range cases {
		if got := Shard(in); got != want {
			t.Errorf("Shard(%q) = %q, want %q", in, got, want)
		}
	}
}
