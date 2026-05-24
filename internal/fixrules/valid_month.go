package fixrules

import (
	"fmt"
	"strings"

	"github.com/yareeh/bibdb/internal"
)

var validMonths = map[string]bool{
	"january": true, "february": true, "march": true, "april": true,
	"may": true, "june": true, "july": true, "august": true,
	"september": true, "october": true, "november": true, "december": true,
}

// Short forms and common abbreviations -> canonical English month names.
var monthAliases = map[string]string{
	"jan": "January", "feb": "February", "mar": "March", "apr": "April",
	"jun": "June", "jul": "July", "aug": "August", "sep": "September",
	"sept": "September", "oct": "October", "nov": "November", "dec": "December",
}

func init() {
	Register(Rule{
		ID:          "valid-month",
		Since:       "1.4.0",
		Severity:    AutoFix,
		Description: "Month is a full English name (case-canonicalised; common abbreviations expanded).",
		Apply: func(e *internal.Entry) Result {
			cur := e.Get("month")
			if cur == "" {
				// required-fields covers this — don't double-report here.
				return Result{}
			}
			trimmed := strings.TrimSpace(cur)
			lower := strings.ToLower(strings.TrimSuffix(trimmed, "."))

			// Already canonical?
			if validMonths[lower] {
				canonical := strings.ToUpper(lower[:1]) + lower[1:]
				if canonical == cur {
					return Result{}
				}
				e.Set("month", canonical)
				return Result{
					Changed:  true,
					Messages: []string{fmt.Sprintf("month %q → %q", cur, canonical)},
				}
			}

			// Try alias.
			if canonical, ok := monthAliases[lower]; ok {
				e.Set("month", canonical)
				return Result{
					Changed:  true,
					Messages: []string{fmt.Sprintf("month %q → %q", cur, canonical)},
				}
			}

			// Unknown — surface so the user can fix it.
			return Result{
				NeedsExternal: []string{fmt.Sprintf("unrecognised month %q (expected full English name)", cur)},
			}
		},
	})
}
