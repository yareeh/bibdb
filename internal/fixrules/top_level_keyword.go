package fixrules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yareeh/bibdb/internal"
)

// topLevelKeywords are the Dewey-inspired category buckets every entry must
// have at least one of, as the first keyword. Established in skyebot's bib
// skill conventions.
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
			kw := strings.ToLower(e.Get("keywords"))
			if kw == "" {
				// required-fields covers this — don't double-report.
				return Result{}
			}
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
