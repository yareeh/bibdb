package fixrules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yareeh/bibdb/internal"
)

var validEntryTypes = map[string]bool{
	"article":       true,
	"book":          true,
	"inproceedings": true,
	"misc":          true,
	"online":        true,
	"techreport":    true,
}

func validEntryTypeList() string {
	out := make([]string, 0, len(validEntryTypes))
	for k := range validEntryTypes {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func init() {
	Register(Rule{
		ID:          "valid-entry-type",
		Since:       "1.4.0",
		Severity:    Report,
		Description: "Entry type must be one of: article, book, inproceedings, misc, online, techreport.",
		Apply: func(e *internal.Entry) Result {
			t := strings.ToLower(strings.TrimSpace(e.Type))
			if validEntryTypes[t] {
				return Result{}
			}
			return Result{
				NeedsExternal: []string{fmt.Sprintf("invalid entry type %q (allowed: %s)", e.Type, validEntryTypeList())},
			}
		},
	})
}
