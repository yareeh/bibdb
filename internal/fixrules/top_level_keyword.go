package fixrules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yareeh/bibdb/internal"
	"github.com/yareeh/bibdb/internal/vocab"
)

// topLevelKeywords are the Dewey-inspired category buckets every entry must
// have at least one of. Established in skyebot's bib skill conventions. This
// is the fallback used when no taxonomy.yaml is loaded; when one is, the rule
// reads the top-level categories (and the full broader hierarchy) from it.
var topLevelKeywords = []string{
	"computer science", "philosophy", "psychology", "religion",
	"social sciences", "language", "pure science", "technology",
	"arts", "recreation", "literature", "history", "geography",
}

func init() {
	Register(Rule{
		ID:          "top-level-keyword",
		Since:       "1.4.0",
		Severity:    Report,
		Description: "Keywords include at least one top-level category: computer science, philosophy, psychology, religion, social sciences, language, pure science, technology, arts, recreation, literature, history, geography.",
		Apply: func(e *internal.Entry) Result {
			raw := e.Get("keywords")
			if strings.TrimSpace(raw) == "" {
				// required-fields covers this — don't double-report.
				return Result{}
			}

			// Prefer the loaded taxonomy: a keyword satisfies the rule when its
			// broader chain reaches a top-level category, not merely when the
			// string contains one.
			if v := vocab.Active(); v != nil {
				var kws []string
				for _, p := range strings.Split(raw, ",") {
					if t := strings.TrimSpace(p); t != "" {
						kws = append(kws, t)
					}
				}
				if v.HasTopLevel(kws) {
					return Result{}
				}
				cats := v.TopLevels()
				sort.Strings(cats)
				return Result{
					NeedsExternal: []string{fmt.Sprintf("keywords missing top-level category (expected one of: %s)", strings.Join(cats, ", "))},
				}
			}

			// Fallback (no taxonomy loaded): substring match against the
			// hardcoded Dewey-inspired list.
			kw := strings.ToLower(raw)
			for _, cat := range topLevelKeywords {
				if strings.Contains(kw, cat) {
					return Result{}
				}
			}
			cats := make([]string, len(topLevelKeywords))
			copy(cats, topLevelKeywords)
			sort.Strings(cats)
			return Result{
				NeedsExternal: []string{fmt.Sprintf("keywords missing top-level category (expected one of: %s)", strings.Join(cats, ", "))},
			}
		},
	})
}
