package fixrules

import (
	"github.com/yareeh/bibdb/internal"
	"github.com/yareeh/bibdb/internal/version"
)

// RunOpts controls which rules run and how their results are reported.
type RunOpts struct {
	// OnlyIDs, if non-empty, restricts the run to rules with these IDs.
	OnlyIDs []string
	// MinVersion overrides the per-rule Since skip threshold. An entry is
	// rescanned by rule r when MinVersion (if set) is < r.Since OR when the
	// entry's bibdbversion is < r.Since.
	MinVersion string
	// ReportOnly skips AutoFix rules entirely; only Report rules run.
	ReportOnly bool
}

// RunReport is the aggregate outcome of running selected rules against one
// entry.
type RunReport struct {
	// EntryKey echoes e.Key for context.
	EntryKey string
	// Changed is true when any AutoFix rule mutated the entry.
	Changed bool
	// Reported is true when any rule produced a NeedsExternal message —
	// useful to the CLI for deciding whether to print the entry header.
	Reported bool
	// PerRule is the result of every rule that actually ran (skipped rules
	// are not present).
	PerRule map[string]Result
	// StampVersion is the bibdbversion value to write after a successful
	// run. It is the highest Since among the rules that actually ran on the
	// entry — never higher, so partial runs don't fraudulently certify the
	// entry against rules that weren't executed.
	StampVersion string
}

// Run applies the selected rules to e and returns a report. The caller
// decides whether to persist the entry (RunReport.Changed) and stamp the
// version (RunReport.StampVersion).
func Run(e *internal.Entry, opts RunOpts) RunReport {
	report := RunReport{EntryKey: e.Key, PerRule: map[string]Result{}}
	entryVersion := internal.EntryVersion(e)

	for _, r := range selectRules(opts) {
		// Skip rules already satisfied by this entry's stamped version.
		threshold := r.Since
		if opts.MinVersion != "" && version.GTE(opts.MinVersion, threshold) {
			// User pinned the floor higher than this rule's Since: still
			// apply (--min-version overrides upward).
			threshold = opts.MinVersion
		}
		if version.GTE(entryVersion, threshold) {
			continue
		}
		if opts.ReportOnly && r.Severity == AutoFix {
			continue
		}
		res := r.Apply(e)
		report.PerRule[r.ID] = res
		if res.Changed {
			report.Changed = true
		}
		if len(res.NeedsExternal) > 0 {
			report.Reported = true
		}
		// Track the highest Since among rules that ran — that's what the
		// entry is "certified" up to after this run.
		if version.GTE(r.Since, report.StampVersion) {
			report.StampVersion = r.Since
		}
	}
	return report
}

func selectRules(opts RunOpts) []Rule {
	if len(opts.OnlyIDs) == 0 {
		return All()
	}
	set := make(map[string]bool, len(opts.OnlyIDs))
	for _, id := range opts.OnlyIDs {
		set[id] = true
	}
	var out []Rule
	for _, r := range All() {
		if set[r.ID] {
			out = append(out, r)
		}
	}
	return out
}
