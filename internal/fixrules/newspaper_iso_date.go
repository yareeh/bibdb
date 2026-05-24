package fixrules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yareeh/bibdb/internal"
)

// newspaperIndicators are journal-name substrings (case-insensitive) that
// strongly suggest a newspaper or daily publication. When matched, the
// `number` field should carry an ISO date (YYYY-MM-DD) per the bib skill
// convention for newspaper articles.
var newspaperIndicators = []string{
	"helsingin sanomat",
	"hs.fi",
	"journal",
	"times",
	"post",
	"guardian",
	"wsj",
	"wall street",
}

var isoDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func init() {
	Register(Rule{
		ID:          "newspaper-iso-date",
		Since:       "1.4.0",
		Severity:    Report,
		Description: "Newspaper @article entries carry an ISO date (YYYY-MM-DD) in the number field.",
		Apply: func(e *internal.Entry) Result {
			if strings.ToLower(strings.TrimSpace(e.Type)) != "article" {
				return Result{}
			}
			journal := strings.ToLower(e.Get("journal"))
			if journal == "" {
				return Result{}
			}
			matched := false
			for _, ind := range newspaperIndicators {
				if strings.Contains(journal, ind) {
					matched = true
					break
				}
			}
			if !matched {
				return Result{}
			}
			number := strings.TrimSpace(e.Get("number"))
			if number == "" {
				return Result{
					NeedsExternal: []string{fmt.Sprintf("newspaper article %q has empty number — expected ISO date (YYYY-MM-DD)", e.Key)},
				}
			}
			if !isoDateRe.MatchString(number) {
				return Result{
					NeedsExternal: []string{fmt.Sprintf("newspaper article %q has non-ISO number %q — expected YYYY-MM-DD", e.Key, number)},
				}
			}
			return Result{}
		},
	})
}
