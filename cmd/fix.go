package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/yareeh/bibdb/internal"
	"github.com/yareeh/bibdb/internal/fixrules"
	"github.com/yareeh/bibdb/internal/version"
)

var (
	fixAll        bool
	fixRules      []string
	fixListRules  bool
	fixMarkdown   bool
	fixDryRun     bool
	fixReportOnly bool
	fixMinVersion string
	fixLimit      int
	fixVerbose    bool
	fixQuiet      bool
)

var fixCmd = &cobra.Command{
	Use:   "fix [<key>]",
	Short: "Apply registered fix rules to one entry or all entries.",
	Long: `Walks one or every entry and applies the registered rules. Each rule is
either an auto-fix (rewrites the entry in place) or a report (surfaces a
problem for human/LLM repair).

Entries carry a bibdbversion field stamping the last bibdb release that
fixed them. By default, a rule only runs against entries whose stored
version is below the rule's Since — so re-running fix --all is cheap.

Examples:
  bibdb fix smith2026foo              # one entry, all applicable rules
  bibdb fix --all                     # every entry
  bibdb fix --all --limit 5           # only fix the first 5 affected entries
  bibdb fix --all --dry-run           # show changes, don't write
  bibdb fix --all --rule utf8-encoding --rule tracking-params
  bibdb fix --list-rules              # show registered rules
  bibdb fix --list-rules --markdown   # regenerate Rules.md content`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFix,
}

func init() {
	fixCmd.Flags().BoolVar(&fixAll, "all", false, "iterate every entry")
	fixCmd.Flags().StringSliceVar(&fixRules, "rule", nil, "restrict to these rule IDs (repeatable)")
	fixCmd.Flags().BoolVar(&fixListRules, "list-rules", false, "print registered rules and exit")
	fixCmd.Flags().BoolVar(&fixMarkdown, "markdown", false, "with --list-rules, emit canonical Rules.md content")
	fixCmd.Flags().BoolVar(&fixDryRun, "dry-run", false, "apply rules in memory without writing entries")
	fixCmd.Flags().BoolVar(&fixReportOnly, "report-only", false, "skip AutoFix rules; only surface issues")
	fixCmd.Flags().StringVar(&fixMinVersion, "min-version", "", "override per-entry skip threshold (e.g. 0.0.0 to force re-run)")
	fixCmd.Flags().IntVarP(&fixLimit, "limit", "n", 0, "stop after touching this many entries (mutation or stamp; 0 = no limit)")
	fixCmd.Flags().BoolVarP(&fixVerbose, "verbose", "v", false, "print each scanned entry (including report-only and no-change) to stdout")
	fixCmd.Flags().BoolVarP(&fixQuiet, "quiet", "q", false, "suppress per-entry output (only the end-of-run summary is printed)")
	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) error {
	if fixListRules {
		return printRules(os.Stdout, fixMarkdown)
	}
	if !fixAll && len(args) != 1 {
		return fmt.Errorf("provide <key> or --all (see bibdb fix --help)")
	}
	if fixAll && len(args) == 1 {
		return fmt.Errorf("cannot combine <key> and --all")
	}

	backend, err := resolveBackend()
	if err != nil {
		return err
	}
	store := internal.NewStore(backend.Path)
	repo := internal.NewRepo(backend.Path)

	var entries []*internal.Entry
	if fixAll {
		entries, err = store.List()
		if err != nil {
			return err
		}
	} else {
		e, err := store.Read(args[0])
		if err != nil {
			return err
		}
		entries = []*internal.Entry{e}
	}

	opts := fixrules.RunOpts{
		OnlyIDs:    fixRules,
		MinVersion: fixMinVersion,
		ReportOnly: fixReportOnly,
	}

	current := version.Current()
	// In default-sweep mode (no filters), every entry the runner inspects
	// gets certified up to `current` even when only Report rules fired.
	// This silences future `fix --all` of the same release; new releases'
	// rules still fire because their Since exceeds the stored stamp.
	fullSweep := !fixReportOnly && len(fixRules) == 0 && fixMinVersion == ""

	var changedPaths []string
	stats := runStats{total: len(entries)}

	for _, e := range entries {
		stats.iterated++
		rep := fixrules.Run(e, opts)

		if len(rep.PerRule) == 0 {
			// Every rule's Since is already covered by the entry's stamp.
			stats.upToDate++
		}
		if rep.Changed {
			stats.autoFixed++
		}
		if rep.Reported {
			stats.reported++
		}

		// Decide whether to certify the entry's stamp.
		// - fullSweep AND at least one rule fired: certify to `current`,
		//   silencing future runs of the same release. If no rule fired
		//   the entry's existing stamp already covers everything; bumping
		//   it just to match the binary version causes pointless write
		//   churn on patch releases that add no rules.
		// - filtered run that changed something: certify to the highest
		//   Since among rules that actually ran (rep.StampVersion).
		// - filtered run that changed nothing: don't certify (the entry
		//   hasn't been seen by every rule, so we can't claim coverage).
		stamp := ""
		switch {
		case fullSweep && len(rep.PerRule) > 0:
			if !version.GTE(internal.EntryVersion(e), current) {
				stamp = current
			}
		case rep.Changed:
			stamp = rep.StampVersion
		}

		needsWrite := rep.Changed || stamp != ""
		// Emit per-entry output (header + indented rule detail) before any
		// mutation so the user sees keys as the run progresses. --quiet
		// suppresses these lines; the end-of-run summary still prints.
		if !fixQuiet {
			emitProgress(os.Stdout, e.Key, rep, stamp, fixDryRun, fixVerbose)
		}

		if !needsWrite {
			continue
		}
		if stamp != "" {
			internal.StampVersion(e, stamp)
		}
		if !rep.Changed && stamp != "" {
			stats.stampedClean++
		}
		stats.touched++

		if !fixDryRun {
			if err := store.Write(e); err != nil {
				return fmt.Errorf("writing %s: %w", e.Key, err)
			}
			changedPaths = append(changedPaths, store.RelPath(e.Key))
		}

		// --limit counts every entry the run touched (mutation OR stamp).
		// A stamp-only sweep through a stale store would otherwise scan
		// the whole database silently before the limit could trigger.
		if fixLimit > 0 && stats.touched >= fixLimit {
			stats.stoppedAtLimit = true
			break
		}
	}

	if !fixDryRun && len(changedPaths) > 0 {
		msg := commitMessage(changedPaths)
		if err := repo.SyncMutation(changedPaths, msg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git sync: %v\n", err)
		}
	}

	stats.print(os.Stdout, fixDryRun, fixLimit)
	return nil
}

// runStats accumulates per-entry outcomes so the run can finish with a
// useful summary rather than a single "Fixed N entries" line.
type runStats struct {
	total          int  // entries selected for this run
	iterated       int  // entries actually visited (= total unless --limit hit)
	upToDate       int  // no rules ran on this entry — already covered
	touched        int  // file written (content mutation or stamp only)
	autoFixed      int  // at least one AutoFix rule mutated content
	reported       int  // at least one Report rule surfaced an issue
	stampedClean   int  // touched purely to apply a version stamp, no mutation
	stoppedAtLimit bool // --limit reached, iteration was cut short
}

func (s runStats) remaining() int {
	r := s.total - s.iterated
	if r < 0 {
		return 0
	}
	return r
}

// print writes the end-of-run summary. Four core numbers (up-to-date /
// processed / fixed / unprocessed) plus a reported-issues line when
// non-zero. "Processed" = entry was touched (mutation or stamp).
func (s runStats) print(w *os.File, dryRun bool, limit int) {
	if s.iterated == 0 && s.total == 0 {
		fmt.Fprintln(w, "No entries selected.")
		return
	}
	if dryRun {
		fmt.Fprintln(w, "\nDry run — no entries were written.")
	} else {
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "  Already up to date: %d\n", s.upToDate)
	fmt.Fprintf(w, "  Processed:          %d\n", s.touched)
	fmt.Fprintf(w, "  Fixed (content):    %d\n", s.autoFixed)
	if s.reported > 0 {
		fmt.Fprintf(w, "  Reported issues:    %d\n", s.reported)
	}
	fmt.Fprintf(w, "  Unprocessed:        %d", s.remaining())
	if s.stoppedAtLimit {
		fmt.Fprintf(w, "  (--limit %d reached)", limit)
	}
	fmt.Fprintln(w)
}

// emitProgress writes one line per entry the run acted on, plus indented
// rule detail. Default output only includes Fixed entries — the inline
// stream then matches "what was fixed" 1:1 with the summary's Fixed count.
// -v adds report/stamp/skip lines for full visibility.
//
//   - Changed:                       "fix <key>"    + indented [rule] messages
//   - Reported (no Changed) + -v:    "report <key>" + indented [rule] (report) messages
//   - Stamping-only + -v:            "stamp <key>"
//   - No rules fired AND -v:         "skip <key>"
func emitProgress(w *os.File, key string, rep fixrules.RunReport, stamp string, dryRun, verbose bool) {
	header := ""
	showRuleDetail := false
	switch {
	case rep.Changed:
		if dryRun {
			header = "would-fix"
		} else {
			header = "fix"
		}
		showRuleDetail = true
	case rep.Reported && verbose:
		header = "report"
		showRuleDetail = true
	case stamp != "" && verbose:
		if dryRun {
			header = "would-stamp"
		} else {
			header = "stamp"
		}
	case verbose:
		header = "skip"
	}
	if header == "" {
		return
	}
	fmt.Fprintf(w, "%s %s\n", header, key)
	if !showRuleDetail {
		return
	}
	var lines []string
	for id, res := range rep.PerRule {
		for _, msg := range res.Messages {
			lines = append(lines, fmt.Sprintf("    [%s] %s", id, msg))
		}
		for _, msg := range res.NeedsExternal {
			lines = append(lines, fmt.Sprintf("    [%s] (report) %s", id, msg))
		}
	}
	sort.Strings(lines)
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
}

func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func commitMessage(paths []string) string {
	if len(paths) == 1 {
		// Extract the cite key from the path "entries/<shard>/<key>.bib"
		key := keyFromPath(paths[0])
		return fmt.Sprintf("bibdb: fix %s", key)
	}
	return fmt.Sprintf("bibdb: fix %d entries", len(paths))
}

func keyFromPath(p string) string {
	parts := strings.Split(p, "/")
	last := parts[len(parts)-1]
	return strings.TrimSuffix(last, ".bib")
}

// printRules emits the registry as a tab-separated table (default) or as
// Rules.md content (when markdown=true). The markdown form is the canonical
// content that lives at the repo root; regenerate with
//
//	bibdb fix --list-rules --markdown > Rules.md
func printRules(w *os.File, markdown bool) error {
	rules := fixrules.All()
	if markdown {
		fmt.Fprint(w, generateRulesMarkdown(rules))
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSINCE\tSEVERITY\tDESCRIPTION")
	for _, r := range rules {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.ID, r.Since, r.Severity, r.Description)
	}
	return tw.Flush()
}

func generateRulesMarkdown(rules []fixrules.Rule) string {
	var b strings.Builder
	b.WriteString("# bibdb fix rules\n\n")
	b.WriteString("This document is generated from the rule registry in `internal/fixrules/`.\n")
	b.WriteString("Regenerate with `bibdb fix --list-rules --markdown > Rules.md`. A unit test\n")
	b.WriteString("(`TestRulesMdInSync`) keeps it in sync with the registered rules.\n\n")
	b.WriteString("**Stability**: rule IDs are an API once shipped. Deprecate, never repurpose.\n\n")
	b.WriteString("| ID | Since | Severity | Description |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, r := range rules {
		b.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n", r.ID, r.Since, r.Severity, r.Description))
	}
	b.WriteString("\n## Discipline\n\n")
	b.WriteString("When a new bibdb release changes the criteria for a well-formed entry —\n")
	b.WriteString("new required field, stricter character set, additional auto-cleanup — add a\n")
	b.WriteString("corresponding `Rule` in `internal/fixrules/` with `Since = <next release>`.\n")
	b.WriteString("Re-running `bibdb fix --all` then brings every legacy entry up to date.\n")
	return b.String()
}
