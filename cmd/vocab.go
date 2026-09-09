package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yareeh/bibdb/internal"
	"github.com/yareeh/bibdb/internal/vocab"
)

var vocabFacet string

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
	vocabCmd.AddCommand(vocabListCmd, vocabCanonicalCmd, vocabNormalizeCmd, vocabTagsCmd, vocabCheckCmd, vocabTreeCmd)
	rootCmd.AddCommand(vocabCmd)
}
