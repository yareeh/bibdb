package fixrules

import (
	"fmt"
	"strings"

	"github.com/yareeh/bibdb/internal"
)

// requiredFields are the BibTeX fields every well-formed bibdb entry must
// carry. Established by skyebot's bib skill conventions (see
// skyebot/src/skyebot/bibtex/rules.py).
var requiredFields = []string{"author", "title", "year", "month", "keywords", "abstract"}

func init() {
	Register(Rule{
		ID:          "required-fields",
		Since:       "1.4.0",
		Severity:    Report,
		Description: "author, title, year, month, keywords, and abstract must all be present and non-empty.",
		Apply: func(e *internal.Entry) Result {
			var missing []string
			for _, name := range requiredFields {
				if strings.TrimSpace(e.Get(name)) == "" {
					missing = append(missing, name)
				}
			}
			if len(missing) == 0 {
				return Result{}
			}
			return Result{
				NeedsExternal: []string{fmt.Sprintf("missing required field(s): %s", strings.Join(missing, ", "))},
			}
		},
	})
}
