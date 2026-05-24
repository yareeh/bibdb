package fixrules

import (
	"testing"

	"github.com/yareeh/bibdb/internal"
)

// withRegistry temporarily replaces the registry for a test. The real rules
// live in their own files and register via init(), so this lets us test the
// runner in isolation without depending on the production rule set.
func withRegistry(t *testing.T, rules []Rule) {
	t.Helper()
	saved := registry
	registry = nil
	for _, r := range rules {
		Register(r)
	}
	t.Cleanup(func() { registry = saved })
}

func TestRunSkipsRulesAlreadyCovered(t *testing.T) {
	withRegistry(t, []Rule{
		{ID: "a", Since: "1.0.0", Severity: AutoFix, Apply: func(e *internal.Entry) Result {
			return Result{Changed: true, Messages: []string{"a fired"}}
		}},
		{ID: "b", Since: "1.4.0", Severity: AutoFix, Apply: func(e *internal.Entry) Result {
			return Result{Changed: true, Messages: []string{"b fired"}}
		}},
	})
	e := &internal.Entry{Key: "x", Fields: []internal.Field{{Name: "bibdbversion", Value: "1.2.0"}}}
	rep := Run(e, RunOpts{})
	if _, ok := rep.PerRule["a"]; ok {
		t.Errorf("rule a (Since 1.0.0) should be skipped — entry already at 1.2.0; PerRule=%v", rep.PerRule)
	}
	if _, ok := rep.PerRule["b"]; !ok {
		t.Errorf("rule b (Since 1.4.0) should run — entry at 1.2.0; PerRule=%v", rep.PerRule)
	}
	if rep.StampVersion != "1.4.0" {
		t.Errorf("StampVersion = %q, want 1.4.0 (highest Since among rules that ran)", rep.StampVersion)
	}
}

func TestRunPartialRuleFilterStampsOnlyThatRulesSince(t *testing.T) {
	withRegistry(t, []Rule{
		{ID: "a", Since: "1.3.0", Severity: AutoFix, Apply: func(e *internal.Entry) Result {
			return Result{Changed: true}
		}},
		{ID: "b", Since: "1.4.0", Severity: AutoFix, Apply: func(e *internal.Entry) Result {
			return Result{Changed: true}
		}},
	})
	e := &internal.Entry{Key: "x"}
	rep := Run(e, RunOpts{OnlyIDs: []string{"a"}})
	if _, ok := rep.PerRule["b"]; ok {
		t.Errorf("rule b should not run when --rule a is specified")
	}
	if rep.StampVersion != "1.3.0" {
		t.Errorf("StampVersion = %q, want 1.3.0 — must not certify against rule b which didn't run", rep.StampVersion)
	}
}

func TestRunLegacyEntryFiresEveryRule(t *testing.T) {
	withRegistry(t, []Rule{
		{ID: "a", Since: "1.0.0", Severity: AutoFix, Apply: func(e *internal.Entry) Result {
			return Result{Changed: true}
		}},
		{ID: "b", Since: "1.4.0", Severity: AutoFix, Apply: func(e *internal.Entry) Result {
			return Result{Changed: true}
		}},
	})
	e := &internal.Entry{Key: "x"} // no bibdbversion → 0.0.0
	rep := Run(e, RunOpts{})
	if len(rep.PerRule) != 2 {
		t.Errorf("expected both rules to fire on legacy entry, got %d (%v)", len(rep.PerRule), rep.PerRule)
	}
}

func TestRunReportOnlySkipsAutoFix(t *testing.T) {
	withRegistry(t, []Rule{
		{ID: "afix", Since: "1.4.0", Severity: AutoFix, Apply: func(e *internal.Entry) Result {
			return Result{Changed: true}
		}},
		{ID: "rep", Since: "1.4.0", Severity: Report, Apply: func(e *internal.Entry) Result {
			return Result{NeedsExternal: []string{"missing X"}}
		}},
	})
	e := &internal.Entry{Key: "x"}
	rep := Run(e, RunOpts{ReportOnly: true})
	if _, ok := rep.PerRule["afix"]; ok {
		t.Errorf("AutoFix rule must not run under ReportOnly")
	}
	if _, ok := rep.PerRule["rep"]; !ok {
		t.Errorf("Report rule must still run under ReportOnly")
	}
	if rep.Changed {
		t.Errorf("Changed must be false under ReportOnly (no mutations)")
	}
}

func TestRunMinVersionOverrideReruns(t *testing.T) {
	// --min-version 0.0.0 should re-run a rule even on a stamped entry.
	withRegistry(t, []Rule{
		{ID: "a", Since: "1.0.0", Severity: AutoFix, Apply: func(e *internal.Entry) Result {
			return Result{Changed: true}
		}},
	})
	e := &internal.Entry{Key: "x", Fields: []internal.Field{{Name: "bibdbversion", Value: "1.4.0"}}}
	rep := Run(e, RunOpts{MinVersion: "1.4.0"})
	// MinVersion 1.4.0 raises threshold; entry stamped 1.4.0 → still skipped.
	if _, ok := rep.PerRule["a"]; ok {
		t.Errorf("rule should still be skipped when entry version equals threshold")
	}

	// MinVersion higher than entry version → fire.
	rep = Run(e, RunOpts{MinVersion: "1.5.0"})
	if _, ok := rep.PerRule["a"]; !ok {
		t.Errorf("rule should fire when MinVersion exceeds entry version")
	}
}

func TestRegisterDuplicateIDPanics(t *testing.T) {
	withRegistry(t, []Rule{{ID: "x", Since: "1.0.0"}})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate Register")
		}
	}()
	Register(Rule{ID: "x", Since: "1.1.0"})
}

func TestAllReturnsCopySortedByID(t *testing.T) {
	withRegistry(t, []Rule{
		{ID: "zebra", Since: "1.0.0"},
		{ID: "alpha", Since: "1.0.0"},
		{ID: "mike", Since: "1.0.0"},
	})
	got := All()
	if len(got) != 3 || got[0].ID != "alpha" || got[1].ID != "mike" || got[2].ID != "zebra" {
		t.Errorf("All() not sorted by ID, got: %+v", got)
	}
	// Mutating the returned slice must not affect the registry.
	got[0].ID = "mutated"
	if All()[0].ID == "mutated" {
		t.Error("All() must return a copy, not the underlying slice")
	}
}
