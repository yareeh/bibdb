package fixrules

import (
	"fmt"
	"regexp"

	"github.com/yareeh/bibdb/internal"
)

var citeKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func init() {
	Register(Rule{
		ID:          "key-format",
		Since:       "1.4.0",
		Severity:    Report,
		Description: "Citation key matches ^[a-z][a-z0-9_]*$ — lowercase, alphanumeric and underscores only, leading letter.",
		Apply: func(e *internal.Entry) Result {
			if citeKeyRe.MatchString(e.Key) {
				return Result{}
			}
			return Result{
				NeedsExternal: []string{fmt.Sprintf("invalid citation key %q (must match ^[a-z][a-z0-9_]*$ — use `bibdb rename %s <new>` after deciding a new key)", e.Key, e.Key)},
			}
		},
	})
}
