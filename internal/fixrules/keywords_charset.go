package fixrules

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/yareeh/bibdb/internal"
)

// Characters that are safe at the source level for keywords. Letters and
// digits (any Unicode) are kept; the punctuation kept is what survives
// toTag's slug conversion intact: space, hyphen, underscore, slash. Anything
// else is stripped so the keyword still round-trips through bibdb's markdown
// export and Obsidian's tag parser without surprises.
func isKeywordSafeChar(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	switch r {
	case ' ', '-', '_', '/':
		return true
	}
	return false
}

func init() {
	Register(Rule{
		ID:          "keywords-charset",
		Since:       "1.4.0",
		Severity:    AutoFix,
		Description: "Strip Obsidian-unsafe punctuation from individual keywords (apostrophes, periods, ampersands, parentheses, …).",
		Apply: func(e *internal.Entry) Result {
			raw := e.Get("keywords")
			if raw == "" {
				return Result{}
			}
			parts := strings.Split(raw, ",")
			var msgs []string
			changed := false
			for i, p := range parts {
				original := p
				trimmed := strings.TrimSpace(p)
				if trimmed == "" {
					continue
				}
				var b strings.Builder
				for _, r := range trimmed {
					if isKeywordSafeChar(r) {
						b.WriteRune(r)
					}
				}
				// Collapse runs of spaces left behind by stripped chars
				cleaned := strings.Join(strings.Fields(b.String()), " ")
				if cleaned != trimmed {
					changed = true
					msgs = append(msgs, fmt.Sprintf("keyword %q → %q", trimmed, cleaned))
				}
				// Restore leading/trailing whitespace from the original split
				// to keep the joined output looking like the input.
				lead := original[:len(original)-len(strings.TrimLeft(original, " \t"))]
				parts[i] = lead + cleaned
			}
			if !changed {
				return Result{}
			}
			e.Set("keywords", strings.Join(parts, ","))
			return Result{Changed: true, Messages: msgs}
		},
	})
}
