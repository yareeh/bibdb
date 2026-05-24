package fixrules

import (
	"fmt"
	"html"
	"regexp"

	"github.com/yareeh/bibdb/internal"
)

// utf8TextFields are the human-readable fields whose values should be proper
// UTF-8 — not HTML entities, not LaTeX accent macros.
var utf8TextFields = []string{"title", "author", "abstract", "keywords", "journal", "publisher", "originaltitle", "translator"}

// LaTeX accent macros that bibtex tooling sometimes emits. Mapped to the
// most likely Unicode replacement. Both `{\"o}` and `\"{o}` and `\"o`
// surface in real-world data; the regex below handles the wrapping braces.
var latexAccents = map[string]map[rune]rune{
	`"`: { // diaeresis
		'a': 'ä', 'A': 'Ä', 'o': 'ö', 'O': 'Ö', 'u': 'ü', 'U': 'Ü',
		'e': 'ë', 'E': 'Ë', 'i': 'ï', 'I': 'Ï', 'y': 'ÿ', 'Y': 'Ÿ',
	},
	`'`: { // acute
		'a': 'á', 'A': 'Á', 'e': 'é', 'E': 'É', 'i': 'í', 'I': 'Í',
		'o': 'ó', 'O': 'Ó', 'u': 'ú', 'U': 'Ú', 'y': 'ý', 'Y': 'Ý',
		'c': 'ć', 'C': 'Ć', 'n': 'ń', 'N': 'Ń', 's': 'ś', 'S': 'Ś',
	},
	"`": { // grave
		'a': 'à', 'A': 'À', 'e': 'è', 'E': 'È', 'i': 'ì', 'I': 'Ì',
		'o': 'ò', 'O': 'Ò', 'u': 'ù', 'U': 'Ù',
	},
	`^`: { // circumflex
		'a': 'â', 'A': 'Â', 'e': 'ê', 'E': 'Ê', 'i': 'î', 'I': 'Î',
		'o': 'ô', 'O': 'Ô', 'u': 'û', 'U': 'Û',
	},
	`~`: { // tilde
		'a': 'ã', 'A': 'Ã', 'n': 'ñ', 'N': 'Ñ', 'o': 'õ', 'O': 'Õ',
	},
	`c`: { // cedilla — \c{c} → ç
		'c': 'ç', 'C': 'Ç', 's': 'ş', 'S': 'Ş',
	},
}

// Matches \X{Y}, \X Y, or {\X Y} where X is an accent macro letter / symbol
// and Y is a single letter. The bracketed forms wrapping the whole thing
// (`{\"o}`) are handled by trying with and without the outer braces.
var (
	// Group 1: accent symbol; group 2: letter when braced ({\"o} or \"{o});
	// group 3: letter when unbraced (\"o). Critically: when the letter is
	// not braced, do NOT eat a trailing '}' — it almost always belongs to a
	// surrounding BibTeX brace-protection that singleCharBraceRe handles
	// after this pass.
	latexAccentRe = regexp.MustCompile(`\\([\"'\x60\^~c])(?:\{([A-Za-z])\}|([A-Za-z]))`)
	// LaTeX named single-letter specials. \b prevents partial matches inside
	// longer macros, and \s* consumes the trailing space LaTeX uses to
	// terminate the macro name (`\aa rsbok` → `årsbok`, no space).
	latexSpecialRe = regexp.MustCompile(`\\(ss|SS|ae|AE|oe|OE|aa|AA|l|L|o|O)\b\s*`)
)

var latexSpecials = map[string]rune{
	"ss": 'ß', "SS": 'ẞ', "ae": 'æ', "AE": 'Æ', "oe": 'œ', "OE": 'Œ',
	"aa": 'å', "AA": 'Å', "o": 'ø', "O": 'Ø', "l": 'ł', "L": 'Ł',
}

func decodeLatex(s string) string {
	// First the single-letter accent macros.
	s = latexAccentRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := latexAccentRe.FindStringSubmatch(m)
		accent := sub[1]
		// Letter comes from either the braced group (sub[2]) or the
		// unbraced group (sub[3]) — exactly one is non-empty.
		letterStr := sub[2]
		if letterStr == "" {
			letterStr = sub[3]
		}
		letter := []rune(letterStr)[0]
		if table, ok := latexAccents[accent]; ok {
			if r, ok := table[letter]; ok {
				return string(r)
			}
		}
		return m
	})
	// Then the named specials.
	s = latexSpecialRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := latexSpecialRe.FindStringSubmatch(m)
		if r, ok := latexSpecials[sub[1]]; ok {
			return string(r)
		}
		return m
	})
	// Strip BibTeX brace-protection that was wrapping the now-decoded char,
	// e.g. `{ö}` → `ö`. Only single-char or single-letter contents.
	s = singleCharBraceRe.ReplaceAllString(s, "$1")
	return s
}

var singleCharBraceRe = regexp.MustCompile(`\{(\pL)\}`)

func init() {
	Register(Rule{
		ID:          "utf8-encoding",
		Since:       "1.4.0",
		Severity:    AutoFix,
		Description: "Text fields use proper UTF-8 — decode HTML entities (&auml;, &#228;, &amp;) and LaTeX accent macros ({\\\"o}, \\^a, \\'e) into Unicode characters.",
		Apply: func(e *internal.Entry) Result {
			var msgs []string
			changed := false
			for _, name := range utf8TextFields {
				v := e.Get(name)
				if v == "" {
					continue
				}
				decoded := html.UnescapeString(v)
				decoded = decodeLatex(decoded)
				if decoded != v {
					e.Set(name, decoded)
					changed = true
					msgs = append(msgs, fmt.Sprintf("%s: decoded %s", name, summarise(v, decoded)))
				}
			}
			if !changed {
				return Result{}
			}
			return Result{Changed: true, Messages: msgs}
		},
	})
}

// summarise builds a compact "X → Y" diff snippet for logging. If either
// side is long, it's truncated; the goal is a useful one-line hint, not the
// full content.
func summarise(before, after string) string {
	const max = 60
	trim := func(s string) string {
		if len(s) <= max {
			return s
		}
		return s[:max] + "…"
	}
	return fmt.Sprintf("%q → %q", trim(before), trim(after))
}
