package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yareeh/bibdb/internal"
	"github.com/yareeh/bibdb/internal/vocab"
)

var (
	vocabFacet            string
	vocabCoverageMinCount int
	vocabCoverageLimit    int
	vocabCoverageJSON     bool
)

var vocabCmd = &cobra.Command{
	Use:   "vocab",
	Short: "Inspect the controlled keyword vocabulary (taxonomy.yaml)",
	Long: `Query the taxonomy.yaml concept scheme at the data-repo root — the single
source of truth for keyword categorization shared by bibdb, skyebot, and the
skye skills.

  bibdb vocab list                 # every canonical term, with facet + parent
  bibdb vocab list --facet top     # just the top-level categories, one per line
  bibdb vocab canonical LLM        # resolve a synonym → canonical prefLabel
  bibdb vocab tags "AI, Iran"      # faceted Obsidian hashtag run (#place/iran …)
  bibdb vocab coverage             # uncurated keywords by frequency (curation TODO)
  bibdb vocab check smith2026foo   # lint one entry's keywords against the scheme
  bibdb vocab tree                 # print the broader/narrower hierarchy`,
}

// loadVocabOrErr resolves the backend and loads its taxonomy, erroring if none
// is present (unlike vocab.Load, which treats a missing file as nil).
func loadVocabOrErr() (*vocab.Vocabulary, *internal.Backend, error) {
	backend, err := resolveBackend()
	if err != nil {
		return nil, nil, err
	}
	v, err := vocab.Load(backend.Path)
	if err != nil {
		return nil, nil, err
	}
	if v == nil {
		return nil, nil, fmt.Errorf("no %s found in %s", vocab.FileName, backend.Path)
	}
	return v, backend, nil
}

var vocabListCmd = &cobra.Command{
	Use:   "list",
	Short: "List canonical terms (optionally filtered by facet)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		v, _, err := loadVocabOrErr()
		if err != nil {
			return err
		}
		concepts := v.Concepts()
		sort.Slice(concepts, func(i, j int) bool { return concepts[i].PrefLabel < concepts[j].PrefLabel })
		for _, c := range concepts {
			if vocabFacet != "" && c.Facet != vocabFacet {
				continue
			}
			if vocabFacet != "" {
				// Filtered output is script-friendly: one prefLabel per line.
				fmt.Println(c.PrefLabel)
			} else {
				fmt.Printf("%s\t%s\t%s\n", c.PrefLabel, c.Facet, c.Broader)
			}
		}
		return nil
	},
}

var vocabCanonicalCmd = &cobra.Command{
	Use:   "canonical <term>",
	Short: "Resolve a keyword to its canonical prefLabel (exit 1 if unknown)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		v, _, err := loadVocabOrErr()
		if err != nil {
			return err
		}
		canon, ok := v.Canonical(strings.Join(args, " "))
		fmt.Println(canon)
		if !ok {
			// Print the (trimmed) input on stdout so callers can always use it,
			// but signal "not in vocabulary" via a non-zero exit.
			os.Exit(1)
		}
		return nil
	},
}

var vocabNormalizeCmd = &cobra.Command{
	Use:   "normalize <keywords>",
	Short: "Print the canonical, de-duplicated form of a comma-separated keywords string",
	Long: `Reads a comma-separated keywords string (pass it as one quoted argument) and
prints the canonical, de-duplicated form — the same normalization the
keywords-vocab fix rule applies. Intended for callers (skyebot, skye skills)
to normalize LLM-generated keywords before 'bibdb add'.

  bibdb vocab normalize "AI, LLM, finland, AI"
  → artificial intelligence, large language models, Finland`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		v, _, err := loadVocabOrErr()
		if err != nil {
			return err
		}
		joined, _, _ := v.NormalizeKeywords(strings.Join(args, " "))
		fmt.Println(joined)
		return nil
	},
}

var vocabTagsCmd = &cobra.Command{
	Use:   "tags <keywords>",
	Short: "Print the faceted, de-duplicated Obsidian hashtag run for a comma-separated keywords string",
	Long: `Reads a comma-separated keywords string (pass it as one quoted argument) and
prints the space-separated Obsidian hashtag run exactly as 'bibdb export' would
render it: entity facets get a '<facet>/' prefix (#place/iran, #person/mr-t),
duplicates that collapse to the same tag are dropped. Intended for callers
(skyebot /z, /food, /sport) so every note tags identically to the References
notes, from this single source of truth.

  bibdb vocab tags "social sciences, Mr T, Lawrence Tureaud, Iran"
  → #social-sciences #person/mr-t #place/iran`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		v, _, err := loadVocabOrErr()
		if err != nil {
			return err
		}
		tags := internal.KeywordTags(v, strings.Join(args, " "))
		fmt.Println(strings.Join(tags, " "))
		return nil
	},
}

// coverageStat is one uncurated keyword and where it appears.
type coverageStat struct {
	Term     string   `json:"term"`
	Count    int      `json:"count"`
	Examples []string `json:"examples"`
}

// scanCoverage tallies keywords not in the taxonomy across entries, counting
// each term once per entry, and returns those used in >= minCount entries,
// sorted by count desc then term asc.
func scanCoverage(v *vocab.Vocabulary, entries []*internal.Entry, minCount int) []*coverageStat {
	counts := map[string]*coverageStat{}
	for _, e := range entries {
		kw := e.Get("keywords")
		if kw == "" {
			continue
		}
		seen := map[string]bool{}
		for _, p := range strings.Split(kw, ",") {
			t := strings.TrimSpace(p)
			if t == "" {
				continue
			}
			if _, known := v.Canonical(t); known {
				continue // already in the taxonomy
			}
			key := strings.ToLower(t)
			if seen[key] {
				continue // count each term once per entry
			}
			seen[key] = true
			s := counts[key]
			if s == nil {
				s = &coverageStat{Term: t}
				counts[key] = s
			}
			s.Count++
			if len(s.Examples) < 3 {
				s.Examples = append(s.Examples, e.Key)
			}
		}
	}
	stats := make([]*coverageStat, 0, len(counts))
	for _, s := range counts {
		if s.Count >= minCount {
			stats = append(stats, s)
		}
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Count != stats[j].Count {
			return stats[i].Count > stats[j].Count
		}
		return stats[i].Term < stats[j].Term
	})
	return stats
}

var vocabCoverageCmd = &cobra.Command{
	Use:   "coverage",
	Short: "Report keywords used across entries that are NOT in the taxonomy (curation candidates)",
	Long: `Scan every entry's keywords and tally those not in taxonomy.yaml — the
uncurated terms that fall through to bare #topic tags. Ranked by how many
entries use each, so the highest-impact concepts to curate surface first. One
scan covers both existing items and any new ones since the last curation pass.

  bibdb vocab coverage                 # terms in >=2 entries, top 50
  bibdb vocab coverage --min-count 1   # everything, including one-offs
  bibdb vocab coverage --json          # machine-readable {term,count,examples}`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		v, backend, err := loadVocabOrErr()
		if err != nil {
			return err
		}
		entries, err := internal.NewStore(backend.Path).List()
		if err != nil {
			return err
		}
		stats := scanCoverage(v, entries, vocabCoverageMinCount)
		if vocabCoverageLimit > 0 && len(stats) > vocabCoverageLimit {
			stats = stats[:vocabCoverageLimit]
		}
		if vocabCoverageJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(stats)
		}
		fmt.Printf("%d uncurated term(s) in >=%d entries (of %d scanned):\n",
			len(stats), vocabCoverageMinCount, len(entries))
		for _, s := range stats {
			fmt.Printf("%5d  %s\n", s.Count, s.Term)
		}
		return nil
	},
}

var vocabCheckCmd = &cobra.Command{
	Use:   "check <key>",
	Short: "Lint one entry's keywords against the taxonomy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		v, backend, err := loadVocabOrErr()
		if err != nil {
			return err
		}
		e, err := internal.NewStore(backend.Path).Read(args[0])
		if err != nil {
			return err
		}
		var kws []string
		for _, p := range strings.Split(e.Get("keywords"), ",") {
			if t := strings.TrimSpace(p); t != "" {
				kws = append(kws, t)
			}
		}
		if len(kws) == 0 {
			fmt.Println("(no keywords)")
			os.Exit(1)
		}
		for _, k := range kws {
			c, ok := v.Lookup(k)
			if !ok {
				fmt.Printf("  %-32s  unknown → kept as topic\n", k)
				continue
			}
			note := ""
			if !strings.EqualFold(c.PrefLabel, k) {
				note = fmt.Sprintf("  (canonical: %s)", c.PrefLabel)
			}
			fmt.Printf("  %-32s  [%s]%s\n", k, v.FacetOf(c), note)
		}
		if v.HasTopLevel(kws) {
			fmt.Println("top-level: ok")
			return nil
		}
		tops := v.TopLevels()
		sort.Strings(tops)
		fmt.Printf("top-level: MISSING (expected one of: %s)\n", strings.Join(tops, ", "))
		os.Exit(1)
		return nil
	},
}

var vocabTreeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Print the broader/narrower concept hierarchy",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		v, _, err := loadVocabOrErr()
		if err != nil {
			return err
		}
		concepts := v.Concepts()

		// Index children by parent id; collect roots (no broader).
		children := make(map[string][]vocab.Concept)
		var roots []vocab.Concept
		for _, c := range concepts {
			if c.Broader == "" {
				roots = append(roots, c)
			} else {
				children[c.Broader] = append(children[c.Broader], c)
			}
		}
		byLabel := func(cs []vocab.Concept) {
			sort.Slice(cs, func(i, j int) bool { return cs[i].PrefLabel < cs[j].PrefLabel })
		}
		byLabel(roots)

		var walk func(c vocab.Concept, depth int)
		walk = func(c vocab.Concept, depth int) {
			facet := c.Facet
			if facet == "" {
				facet = v.Scheme.DefaultFacet
			}
			fmt.Printf("%s%s  [%s]\n", strings.Repeat("  ", depth), c.PrefLabel, facet)
			kids := children[c.ID]
			byLabel(kids)
			for _, k := range kids {
				walk(k, depth+1)
			}
		}
		// Top-level categories first, then remaining roots (places, people…).
		var tops, others []vocab.Concept
		for _, r := range roots {
			if r.Facet == "top" {
				tops = append(tops, r)
			} else {
				others = append(others, r)
			}
		}
		for _, r := range tops {
			walk(r, 0)
		}
		for _, r := range others {
			walk(r, 0)
		}
		return nil
	},
}

func init() {
	vocabListCmd.Flags().StringVar(&vocabFacet, "facet", "", "only terms in this facet (top|topic|place|person|org|genre|time)")
	vocabCoverageCmd.Flags().IntVar(&vocabCoverageMinCount, "min-count", 2, "only terms used in at least this many entries")
	vocabCoverageCmd.Flags().IntVar(&vocabCoverageLimit, "limit", 50, "cap the number of terms printed (0 = no cap)")
	vocabCoverageCmd.Flags().BoolVar(&vocabCoverageJSON, "json", false, "emit JSON ({term,count,examples}) instead of a table")
	vocabCmd.AddCommand(vocabListCmd, vocabCanonicalCmd, vocabNormalizeCmd, vocabTagsCmd, vocabCoverageCmd, vocabCheckCmd, vocabTreeCmd)
	rootCmd.AddCommand(vocabCmd)
}
