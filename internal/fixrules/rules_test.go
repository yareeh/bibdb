package fixrules

import (
	"strings"
	"testing"

	"github.com/yareeh/bibdb/internal"
)

func entry(typ, key string, kv ...string) *internal.Entry {
	e := &internal.Entry{Type: typ, Key: key}
	for i := 0; i+1 < len(kv); i += 2 {
		e.Fields = append(e.Fields, internal.Field{Name: kv[i], Value: kv[i+1]})
	}
	return e
}

func mustRule(t *testing.T, id string) Rule {
	t.Helper()
	r, ok := ByID(id)
	if !ok {
		t.Fatalf("rule %q not registered", id)
	}
	return r
}

// --- required-fields ---

func TestRequiredFields(t *testing.T) {
	r := mustRule(t, "required-fields")

	full := entry("article", "x",
		"author", "Smith, A.", "title", "T", "year", "2026", "month", "May",
		"keywords", "literature, foo", "abstract", "An abstract.")
	if res := r.Apply(full); len(res.NeedsExternal) != 0 {
		t.Errorf("full entry should pass, got %v", res.NeedsExternal)
	}

	missing := entry("article", "x", "title", "T", "year", "2026")
	res := r.Apply(missing)
	if len(res.NeedsExternal) != 1 {
		t.Fatalf("expected one report message, got %v", res.NeedsExternal)
	}
	for _, want := range []string{"author", "month", "keywords", "abstract"} {
		if !strings.Contains(res.NeedsExternal[0], want) {
			t.Errorf("expected message to mention %q, got %q", want, res.NeedsExternal[0])
		}
	}
}

// --- valid-entry-type ---

func TestValidEntryType(t *testing.T) {
	r := mustRule(t, "valid-entry-type")
	for _, ok := range []string{"article", "book", "inproceedings", "misc", "online", "techreport"} {
		if res := r.Apply(entry(ok, "x")); len(res.NeedsExternal) != 0 {
			t.Errorf("%q should pass, got %v", ok, res.NeedsExternal)
		}
	}
	res := r.Apply(entry("phdthesis", "x"))
	if len(res.NeedsExternal) != 1 || !strings.Contains(res.NeedsExternal[0], "phdthesis") {
		t.Errorf("expected report mentioning phdthesis, got %v", res.NeedsExternal)
	}
}

// --- valid-month ---

func TestValidMonthCanonical(t *testing.T) {
	r := mustRule(t, "valid-month")
	e := entry("article", "x", "month", "May")
	if res := r.Apply(e); res.Changed {
		t.Errorf("canonical month should not change, got %+v", res)
	}
}

func TestValidMonthCaseFix(t *testing.T) {
	r := mustRule(t, "valid-month")
	e := entry("article", "x", "month", "may")
	res := r.Apply(e)
	if !res.Changed || e.Get("month") != "May" {
		t.Errorf("expected month canonicalised to May, got %q (changed=%v)", e.Get("month"), res.Changed)
	}
}

func TestValidMonthAlias(t *testing.T) {
	r := mustRule(t, "valid-month")
	cases := map[string]string{"Jan": "January", "jan.": "January", "sept": "September", "Dec": "December"}
	for in, want := range cases {
		e := entry("article", "x", "month", in)
		r.Apply(e)
		if got := e.Get("month"); got != want {
			t.Errorf("alias %q → %q, want %q", in, got, want)
		}
	}
}

func TestValidMonthUnknownReports(t *testing.T) {
	r := mustRule(t, "valid-month")
	res := r.Apply(entry("article", "x", "month", "Heinäkuu"))
	if len(res.NeedsExternal) != 1 {
		t.Errorf("expected one report for unknown month, got %v", res.NeedsExternal)
	}
}

// --- key-format ---

func TestKeyFormatValid(t *testing.T) {
	r := mustRule(t, "key-format")
	for _, k := range []string{"smith2026foo", "abc_def", "a", "z9"} {
		if res := r.Apply(entry("misc", k)); len(res.NeedsExternal) != 0 {
			t.Errorf("key %q should pass, got %v", k, res.NeedsExternal)
		}
	}
}

func TestKeyFormatInvalid(t *testing.T) {
	r := mustRule(t, "key-format")
	for _, k := range []string{"Smith2026", "2026smith", "smith-foo", "smith.foo", ""} {
		if res := r.Apply(entry("misc", k)); len(res.NeedsExternal) == 0 {
			t.Errorf("key %q should be reported", k)
		}
	}
}

// --- top-level-keyword ---

func TestTopLevelKeywordPresent(t *testing.T) {
	r := mustRule(t, "top-level-keyword")
	for _, kw := range []string{
		"literature, fiction",
		"Computer Science, AI",
		"social sciences, politics",
	} {
		if res := r.Apply(entry("misc", "x", "keywords", kw)); len(res.NeedsExternal) != 0 {
			t.Errorf("keywords %q should pass, got %v", kw, res.NeedsExternal)
		}
	}
}

func TestTopLevelKeywordMissing(t *testing.T) {
	r := mustRule(t, "top-level-keyword")
	res := r.Apply(entry("misc", "x", "keywords", "foo, bar, baz"))
	if len(res.NeedsExternal) != 1 {
		t.Errorf("expected one report for missing category, got %v", res.NeedsExternal)
	}
}

// --- tracking-params ---

func TestTrackingParamsStripped(t *testing.T) {
	r := mustRule(t, "tracking-params")
	e := entry("misc", "x", "url", "https://example.com/a?utm_source=newsletter&utm_campaign=spring&id=42&fbclid=abc")
	res := r.Apply(e)
	if !res.Changed {
		t.Fatalf("expected change, got %+v", res)
	}
	got := e.Get("url")
	if strings.Contains(got, "utm_") || strings.Contains(got, "fbclid") {
		t.Errorf("tracking still present: %q", got)
	}
	if !strings.Contains(got, "id=42") {
		t.Errorf("non-tracking param dropped: %q", got)
	}
}

func TestTrackingParamsNoop(t *testing.T) {
	r := mustRule(t, "tracking-params")
	e := entry("misc", "x", "url", "https://example.com/a?id=42")
	if res := r.Apply(e); res.Changed {
		t.Errorf("no tracking params present, should not change; got %+v", res)
	}
}

// --- keywords-charset ---

func TestKeywordsCharsetStripsApostrophe(t *testing.T) {
	r := mustRule(t, "keywords-charset")
	e := entry("misc", "x", "keywords", "literature, children's literature, fantasy")
	res := r.Apply(e)
	if !res.Changed {
		t.Fatalf("expected change, got %+v", res)
	}
	if strings.Contains(e.Get("keywords"), "'") {
		t.Errorf("apostrophe still present: %q", e.Get("keywords"))
	}
	if !strings.Contains(e.Get("keywords"), "childrens literature") {
		t.Errorf("expected 'childrens literature', got %q", e.Get("keywords"))
	}
}

func TestKeywordsCharsetPreservesUnicode(t *testing.T) {
	r := mustRule(t, "keywords-charset")
	e := entry("misc", "x", "keywords", "hanna mahlamäki, literature")
	if res := r.Apply(e); res.Changed {
		t.Errorf("unicode letters should be preserved, got change: %+v (now %q)", res, e.Get("keywords"))
	}
}

// --- utf8-encoding ---

func TestUTF8DecodesHTMLEntities(t *testing.T) {
	r := mustRule(t, "utf8-encoding")
	e := entry("misc", "x",
		"title", "M&auml;k&auml;l&auml;n kirja",
		"author", "M&#228;kel&#228;, Jari")
	res := r.Apply(e)
	if !res.Changed {
		t.Fatalf("expected change, got %+v", res)
	}
	if e.Get("title") != "Mäkälän kirja" {
		t.Errorf("title decode failed: %q", e.Get("title"))
	}
	if e.Get("author") != "Mäkelä, Jari" {
		t.Errorf("author decode failed: %q", e.Get("author"))
	}
}

func TestUTF8DecodesLatexAccents(t *testing.T) {
	r := mustRule(t, "utf8-encoding")
	e := entry("misc", "x",
		"title", `M{\"a}kel{\"a}n kirja`,
		"author", `M\"akel\"a, Jari`)
	r.Apply(e)
	if !strings.Contains(e.Get("title"), "Mäkelän") {
		t.Errorf("LaTeX accent decode failed: %q", e.Get("title"))
	}
	if !strings.Contains(e.Get("author"), "Mäkelä") {
		t.Errorf("LaTeX accent (no braces) decode failed: %q", e.Get("author"))
	}
}

func TestUTF8DecodesLatexSpecials(t *testing.T) {
	r := mustRule(t, "utf8-encoding")
	e := entry("misc", "x", "title", `Stra{\ss}e — \aa rsbok`)
	r.Apply(e)
	if !strings.Contains(e.Get("title"), "Straße") {
		t.Errorf("\\ss decode failed: %q", e.Get("title"))
	}
	if !strings.Contains(e.Get("title"), "årsbok") {
		t.Errorf("\\aa decode failed: %q", e.Get("title"))
	}
}

func TestUTF8NoopAlreadyClean(t *testing.T) {
	r := mustRule(t, "utf8-encoding")
	e := entry("misc", "x", "title", "Mäkelän kirja", "author", "Mäkelä, Jari")
	if res := r.Apply(e); res.Changed {
		t.Errorf("clean UTF-8 should not change, got %+v", res)
	}
}

// --- newspaper-iso-date ---

func TestNewspaperISODateMissing(t *testing.T) {
	r := mustRule(t, "newspaper-iso-date")
	e := entry("article", "x", "journal", "Helsingin Sanomat", "number", "")
	res := r.Apply(e)
	if len(res.NeedsExternal) != 1 {
		t.Errorf("expected one report for empty number, got %v", res.NeedsExternal)
	}
}

func TestNewspaperISODateNonISO(t *testing.T) {
	r := mustRule(t, "newspaper-iso-date")
	e := entry("article", "x", "journal", "The Guardian", "number", "issue 42")
	res := r.Apply(e)
	if len(res.NeedsExternal) != 1 || !strings.Contains(res.NeedsExternal[0], "issue 42") {
		t.Errorf("expected report mentioning non-ISO number, got %v", res.NeedsExternal)
	}
}

func TestNewspaperISODateValid(t *testing.T) {
	r := mustRule(t, "newspaper-iso-date")
	e := entry("article", "x", "journal", "Helsingin Sanomat", "number", "2026-05-21")
	if res := r.Apply(e); len(res.NeedsExternal) != 0 {
		t.Errorf("valid ISO date should pass, got %v", res.NeedsExternal)
	}
}

func TestNewspaperISODateNonNewspaperIgnored(t *testing.T) {
	r := mustRule(t, "newspaper-iso-date")
	e := entry("article", "x", "journal", "Nature", "number", "")
	if res := r.Apply(e); len(res.NeedsExternal) != 0 {
		t.Errorf("non-newspaper article should be ignored, got %v", res.NeedsExternal)
	}
}
