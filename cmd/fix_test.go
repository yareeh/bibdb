package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yareeh/bibdb/internal"
)

// captureStdout swaps os.Stdout for the duration of fn and returns what was
// written. Used to assert verbose output from the fix command.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	w.Close()
	os.Stdout = old
	return <-done
}

func resetFixFlags() {
	fixAll = false
	fixRules = nil
	fixListRules = false
	fixMarkdown = false
	fixDryRun = false
	fixReportOnly = false
	fixMinVersion = ""
	fixLimit = 0
	fixVerbose = false
	fixQuiet = false
}

// dirtyEntry returns an entry with a tracking-param URL and an apostrophe in
// its keywords — two AutoFix rules will fire — but missing abstract/year
// (Report rules will surface those).
func dirtyEntry(key string) *internal.Entry {
	return &internal.Entry{
		Type: "article", Key: key,
		Fields: []internal.Field{
			{Name: "author", Value: "Smith, A."},
			{Name: "title", Value: "T"},
			{Name: "month", Value: "may"}, // valid-month auto-fix → May
			{Name: "keywords", Value: "literature, children's literature"},
			{Name: "url", Value: "https://example.com/x?utm_source=newsletter&id=42"},
		},
	}
}

func TestFixSingleEntryWritesAndStamps(t *testing.T) {
	resetFixFlags()
	store, _ := setupTestBackend(t, []*internal.Entry{dirtyEntry("smith2026foo")})

	if err := runCmd("fix", "smith2026foo"); err != nil {
		t.Fatalf("fix: %v", err)
	}

	got, err := store.Read("smith2026foo")
	if err != nil {
		t.Fatal(err)
	}
	if v := internal.EntryVersion(got); v == "0.0.0" {
		t.Errorf("expected bibdbversion stamped, got %q", v)
	}
	if strings.Contains(got.Get("keywords"), "'") {
		t.Errorf("apostrophe still present in keywords: %q", got.Get("keywords"))
	}
	if strings.Contains(got.Get("url"), "utm_source") {
		t.Errorf("tracking still present in url: %q", got.Get("url"))
	}
	if got.Get("month") != "May" {
		t.Errorf("month not canonicalised: %q", got.Get("month"))
	}
}

func TestFixDryRunDoesNotWrite(t *testing.T) {
	resetFixFlags()
	store, _ := setupTestBackend(t, []*internal.Entry{dirtyEntry("a")})

	if err := runCmd("fix", "a", "--dry-run"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := store.Read("a")
	if !strings.Contains(got.Get("keywords"), "'") {
		t.Errorf("dry-run should not have modified entry — apostrophe gone: %q", got.Get("keywords"))
	}
	if got.Get("month") == "May" {
		t.Errorf("dry-run should not have canonicalised month, still got %q", got.Get("month"))
	}
}

func TestFixAllIteratesEveryEntry(t *testing.T) {
	resetFixFlags()
	store, _ := setupTestBackend(t, []*internal.Entry{
		dirtyEntry("a"), dirtyEntry("b"), dirtyEntry("c"),
	})

	if err := runCmd("fix", "--all"); err != nil {
		t.Fatalf("fix --all: %v", err)
	}
	for _, k := range []string{"a", "b", "c"} {
		got, _ := store.Read(k)
		if strings.Contains(got.Get("keywords"), "'") {
			t.Errorf("%s: apostrophe still present", k)
		}
	}
}

func TestFixLimitStopsAfterN(t *testing.T) {
	resetFixFlags()
	store, _ := setupTestBackend(t, []*internal.Entry{
		dirtyEntry("a"), dirtyEntry("b"), dirtyEntry("c"),
	})

	if err := runCmd("fix", "--all", "--limit", "1"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	// Count entries that were fixed (no apostrophe in keywords).
	fixed := 0
	for _, k := range []string{"a", "b", "c"} {
		got, _ := store.Read(k)
		if !strings.Contains(got.Get("keywords"), "'") {
			fixed++
		}
	}
	if fixed != 1 {
		t.Errorf("expected exactly 1 fixed entry under --limit 1, got %d", fixed)
	}
}

func TestFixRuleFilterRestrictsRules(t *testing.T) {
	resetFixFlags()
	store, _ := setupTestBackend(t, []*internal.Entry{dirtyEntry("a")})

	if err := runCmd("fix", "a", "--rule", "tracking-params"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := store.Read("a")
	if strings.Contains(got.Get("url"), "utm_source") {
		t.Errorf("tracking-params should have fired, url=%q", got.Get("url"))
	}
	// Other AutoFix rules must NOT have fired.
	if !strings.Contains(got.Get("keywords"), "'") {
		t.Errorf("keywords-charset should NOT have fired under --rule filter, keywords=%q", got.Get("keywords"))
	}
	if got.Get("month") != "may" {
		t.Errorf("valid-month should NOT have fired, month=%q", got.Get("month"))
	}
}

func TestFixReportOnlyDoesNotMutate(t *testing.T) {
	resetFixFlags()
	store, _ := setupTestBackend(t, []*internal.Entry{dirtyEntry("a")})

	if err := runCmd("fix", "a", "--report-only"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := store.Read("a")
	if !strings.Contains(got.Get("keywords"), "'") {
		t.Errorf("--report-only must not mutate, but keywords changed")
	}
}

func TestFixListRulesTabular(t *testing.T) {
	resetFixFlags()
	setupTestBackend(t, nil)
	out := captureStdout(t, func() {
		if err := runCmd("fix", "--list-rules"); err != nil {
			t.Fatalf("fix --list-rules: %v", err)
		}
	})
	for _, want := range []string{"ID", "SINCE", "SEVERITY", "tracking-params", "utf8-encoding", "1.4.0"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestFixListRulesMarkdown(t *testing.T) {
	resetFixFlags()
	setupTestBackend(t, nil)
	out := captureStdout(t, func() {
		if err := runCmd("fix", "--list-rules", "--markdown"); err != nil {
			t.Fatalf("fix: %v", err)
		}
	})
	for _, want := range []string{"# bibdb fix rules", "| ID | Since | Severity", "`tracking-params`", "Discipline"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected markdown output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestFixVerboseLogsEveryEntry(t *testing.T) {
	resetFixFlags()
	// One dirty entry, one already-clean entry.
	clean := &internal.Entry{
		Type: "misc", Key: "clean",
		Fields: []internal.Field{
			{Name: "author", Value: "X"},
			{Name: "title", Value: "T"},
			{Name: "year", Value: "2026"},
			{Name: "month", Value: "May"},
			{Name: "keywords", Value: "literature, foo"},
			{Name: "abstract", Value: "A."},
			{Name: "bibdbversion", Value: "1.4.0"},
		},
	}
	setupTestBackend(t, []*internal.Entry{dirtyEntry("dirty"), clean})

	out := captureStdout(t, func() {
		if err := runCmd("fix", "--all", "--verbose"); err != nil {
			t.Fatalf("fix: %v", err)
		}
	})
	if !strings.Contains(out, "fix dirty") {
		t.Errorf("expected verbose output to include 'fix dirty', got:\n%s", out)
	}
	// `clean` was stamped to the current version in the fixture, so under
	// full-sweep verbose mode it produces a "skip clean" line. Older v1.4.x
	// also accepted "scan clean (no changes)" / "stamp clean".
	if !strings.Contains(out, "skip clean") && !strings.Contains(out, "scan clean") && !strings.Contains(out, "stamp clean") {
		t.Errorf("expected verbose output to mention the clean entry, got:\n%s", out)
	}
}

func TestFixRejectsKeyWithAll(t *testing.T) {
	resetFixFlags()
	setupTestBackend(t, []*internal.Entry{dirtyEntry("a")})
	if err := runCmd("fix", "a", "--all"); err == nil {
		t.Error("expected error when combining <key> and --all")
	}
}

func TestFixRequiresKeyOrAll(t *testing.T) {
	resetFixFlags()
	setupTestBackend(t, nil)
	if err := runCmd("fix"); err == nil {
		t.Error("expected error when neither <key> nor --all provided")
	}
}

func TestFixSecondRunIsNoop(t *testing.T) {
	resetFixFlags()
	store, _ := setupTestBackend(t, []*internal.Entry{dirtyEntry("a")})
	if err := runCmd("fix", "a"); err != nil {
		t.Fatal(err)
	}
	stampedV := internal.EntryVersion(must(store.Read("a")))
	// Second run should be a no-op — version stamp blocks rules.
	if err := runCmd("fix", "a"); err != nil {
		t.Fatal(err)
	}
	after := must(store.Read("a"))
	if v := internal.EntryVersion(after); v != stampedV {
		t.Errorf("second run changed bibdbversion from %q to %q (expected no change)", stampedV, v)
	}
}

// reportOnlyEntry has issues only Report rules can flag (no apostrophes,
// no tracking params, no LaTeX/HTML to decode) — but missing abstract and
// no top-level keyword category.
func reportOnlyEntry(key string) *internal.Entry {
	return &internal.Entry{
		Type: "misc", Key: key,
		Fields: []internal.Field{
			{Name: "author", Value: "Doe, J."},
			{Name: "title", Value: "T"},
			{Name: "year", Value: "2026"},
			{Name: "month", Value: "May"},
			{Name: "keywords", Value: "foo, bar"}, // no top-level category
			// abstract missing
		},
	}
}

func TestFixStampsReportOnlyEntriesInFullSweep(t *testing.T) {
	// Bug observed in v1.4.0: entries that only triggered Report rules were
	// never stamped, so every `fix --all` rerun spammed the same reports.
	resetFixFlags()
	store, _ := setupTestBackend(t, []*internal.Entry{reportOnlyEntry("only_reports")})

	if err := runCmd("fix", "--all"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := store.Read("only_reports")
	if v := internal.EntryVersion(got); v == "0.0.0" {
		t.Fatalf("expected report-only entry to be stamped after a full sweep, got %q", v)
	}
}

func TestFixSecondFullSweepDoesNotRerunReports(t *testing.T) {
	resetFixFlags()
	setupTestBackend(t, []*internal.Entry{reportOnlyEntry("a")})

	// First sweep — stamps and surfaces reports in the summary.
	out1 := captureStdout(t, func() {
		if err := runCmd("fix", "--all"); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out1, "Reported issues:") {
		t.Fatalf("first sweep should surface a 'Reported issues:' line, got:\n%s", out1)
	}

	// Second sweep — entry already certified, no report should fire.
	resetFixFlags()
	out2 := captureStdout(t, func() {
		if err := runCmd("fix", "--all"); err != nil {
			t.Fatal(err)
		}
	})
	if strings.Contains(out2, "Reported issues:") {
		t.Errorf("second sweep should not re-emit reports for stamped entries, got:\n%s", out2)
	}
}

func TestFixLimitCapsStampingNotJustAutoFixes(t *testing.T) {
	// Bug observed in v1.4.1: `--limit 1` only broke after the 1st AutoFix
	// mutation. When no AutoFix could fire (e.g. the data was already
	// auto-fixed) but stamping still happened on every iterated entry, the
	// limit never triggered and the full database silently got stamped.
	// v1.4.2 counts every touch (mutation or stamp) toward the limit.
	resetFixFlags()
	// Three entries that only Report-rule against — no AutoFix can fire.
	store, _ := setupTestBackend(t, []*internal.Entry{
		reportOnlyEntry("a"), reportOnlyEntry("b"), reportOnlyEntry("c"),
	})

	if err := runCmd("fix", "--all", "--limit", "1"); err != nil {
		t.Fatal(err)
	}

	stamped := 0
	for _, k := range []string{"a", "b", "c"} {
		got, _ := store.Read(k)
		if internal.EntryVersion(got) != "0.0.0" {
			stamped++
		}
	}
	if stamped != 1 {
		t.Errorf("--limit 1 should have stamped exactly 1 entry, got %d", stamped)
	}
}

func TestFixFilteredRunDoesNotFullStampUnchangedEntry(t *testing.T) {
	// --rule restricts the run; an unchanged entry must NOT be marked covered
	// against rules that weren't executed.
	resetFixFlags()
	store, _ := setupTestBackend(t, []*internal.Entry{reportOnlyEntry("a")})

	if err := runCmd("fix", "a", "--rule", "tracking-params"); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Read("a")
	if v := internal.EntryVersion(got); v != "0.0.0" {
		t.Errorf("filtered run should not certify entry up to current version, got bibdbversion=%q", v)
	}
}

// captureStderr is the stderr twin of captureStdout — runFix emits the
// "Issues that need external repair:" summary on stderr.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	w.Close()
	os.Stderr = old
	return <-done
}

func must(e *internal.Entry, err error) *internal.Entry {
	if err != nil {
		panic(err)
	}
	return e
}
