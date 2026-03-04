package internal

import (
	"fmt"
	"strings"
)

// FormatEntry formats a single entry as a BibTeX string.
func FormatEntry(e *Entry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "@%s{%s", e.Type, e.Key)

	for _, f := range e.Fields {
		b.WriteString(",\n")
		fmt.Fprintf(&b, "  %s = {%s}", f.Name, f.Value)
	}

	b.WriteString("\n}\n")
	return b.String()
}
