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
	fixCmd.Flags().IntVarP(&fixLimit, "limit", "n", 0, "stop after fixing this many entries (0 = no limit)")
	fixCmd.Flags().BoolVarP(&fixVerbose, "verbose", "v", false, "print each fixed entry to stdout as it proceeds")
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
	var changedPaths []string
	var reportLines []string
	changedCount := 0

	for _, e := range entries {
		rep := fixrules.Run(e, opts)

		// Print verbose progress before the potential mutation so the user
		// sees what happened on this entry whether or not we persist it.
		if fixVerbose || rep.Changed || hasReports(rep) {
			emitProgress(os.Stdout, e.Key, rep, fixDryRun, fixVerbose)
		}
		for _, res := range rep.PerRule {
			for _, msg := range res.NeedsExternal {
				reportLines = append(reportLines, fmt.Sprintf("%s: %s", e.Key, msg))
			}
		}

		if !rep.Changed {
			continue
		}
		changedCount++

		// Stamp the entry up to the highest Since among rules that ran on it.
		// For a full default run (no --rule filter, no --report-only, no
		// --min-version override) this is effectively `version.Current()`.
		stamp := rep.StampVersion
		if !fixReportOnly && len(fixRules) == 0 && fixMinVersion == "" {
			stamp = current
		}
		if stamp != "" {
			internal.StampVersion(e, stamp)
		}

		if fixDryRun {
			continue
		}
		if err := store.Write(e); err != nil {
			return fmt.Errorf("writing %s: %w", e.Key, err)
		}
		changedPaths = append(changedPaths, store.RelPath(e.Key))

		if fixLimit > 0 && changedCount >= fixLimit {
			break
		}
	}

	// Surface report-severity findings together at the end so they don't
	// drown the changed-entry progress output.
	if len(reportLines) > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Issues that need external repair:")
		sort.Strings(reportLines)
		for _, line := range reportLines {
			fmt.Fprintln(os.Stderr, "  "+line)
		}
	}

	if fixDryRun {
		fmt.Printf("Dry run: %d entr%s would change.\n", changedCount, pluralY(changedCount))
		return nil
	}
	if len(changedPaths) > 0 {
		msg := commitMessage(changedPaths)
		if err := repo.SyncMutation(changedPaths, msg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git sync: %v\n", err)
		}
	}
	fmt.Printf("Fixed %d entr%s.\n", changedCount, pluralY(changedCount))
	return nil
}

func hasReports(rep fixrules.RunReport) bool {
	for _, res := range rep.PerRule {
		if len(res.NeedsExternal) > 0 {
			return true
		}
	}
	return false
}

func emitProgress(w *os.File, key string, rep fixrules.RunReport, dryRun, verbose bool) {
	var lines []string
	for id, res := range rep.PerRule {
		if !res.Changed && len(res.NeedsExternal) == 0 && !verbose {
			continue
		}
		for _, msg := range res.Messages {
			lines = append(lines, fmt.Sprintf("    [%s] %s", id, msg))
		}
		for _, msg := range res.NeedsExternal {
			lines = append(lines, fmt.Sprintf("    [%s] (report) %s", id, msg))
		}
	}
	if len(lines) == 0 && !verbose {
		return
	}
	prefix := "fix"
	if dryRun {
		prefix = "would-fix"
	}
	if rep.Changed {
		fmt.Fprintf(w, "%s %s\n", prefix, key)
	} else if verbose {
		fmt.Fprintf(w, "scan %s (no changes)\n", key)
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
