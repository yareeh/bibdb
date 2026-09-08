package fixrules

import (
	"strings"

	"github.com/yareeh/bibdb/internal"
	"github.com/yareeh/bibdb/internal/vocab"
)

// keywords-vocab normalizes each keyword to its canonical prefLabel using the
// active taxonomy (altLabel → prefLabel) and removes duplicates. It is a no-op
// when no taxonomy is loaded, or when nothing semantic changes (pure spacing
// differences are left to keywords-charset / the writer).
func init() {
	Register(Rule{
		ID:          "keywords-vocab",
		Since:       "1.5.0",
		Severity:    AutoFix,
		Description: "Normalize keywords to their canonical form via the taxonomy (synonym → prefLabel) and remove duplicates.",
		Apply: func(e *internal.Entry) Result {
			v := vocab.Active()
			if v == nil {
				return Result{}
			}
			raw := e.Get("keywords")
			if strings.TrimSpace(raw) == "" {
				return Result{}
			}
			// Only rewrite when the canonical token sequence differs from the
			// input token sequence — entries already canonical are untouched
			// (spacing quirks aside).
			joined, changed, msgs := v.NormalizeKeywords(raw)
			if !changed {
				return Result{}
			}
			e.Set("keywords", joined)
			return Result{Changed: true, Messages: msgs}
		},
	})
}
