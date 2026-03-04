package internal

import (
	"regexp"
	"strings"
)

// LaTeX accent command → base letter → UTF-8 character
var latexAccents = map[string]map[rune]rune{
	`"`: { // umlaut/diaeresis
		'a': 'ä', 'e': 'ë', 'i': 'ï', 'o': 'ö', 'u': 'ü', 'y': 'ÿ',
		'A': 'Ä', 'E': 'Ë', 'I': 'Ï', 'O': 'Ö', 'U': 'Ü', 'Y': 'Ÿ',
	},
	`'`: { // acute
		'a': 'á', 'e': 'é', 'i': 'í', 'o': 'ó', 'u': 'ú', 'y': 'ý',
		'A': 'Á', 'E': 'É', 'I': 'Í', 'O': 'Ó', 'U': 'Ú', 'Y': 'Ý',
		'c': 'ć', 'C': 'Ć', 'n': 'ń', 'N': 'Ń', 's': 'ś', 'S': 'Ś',
		'z': 'ź', 'Z': 'Ź',
	},
	"`": { // grave
		'a': 'à', 'e': 'è', 'i': 'ì', 'o': 'ò', 'u': 'ù',
		'A': 'À', 'E': 'È', 'I': 'Ì', 'O': 'Ò', 'U': 'Ù',
	},
	"^": { // circumflex
		'a': 'â', 'e': 'ê', 'i': 'î', 'o': 'ô', 'u': 'û',
		'A': 'Â', 'E': 'Ê', 'I': 'Î', 'O': 'Ô', 'U': 'Û',
	},
	"~": { // tilde
		'a': 'ã', 'n': 'ñ', 'o': 'õ',
		'A': 'Ã', 'N': 'Ñ', 'O': 'Õ',
	},
	"=": { // macron
		'a': 'ā', 'e': 'ē', 'i': 'ī', 'o': 'ō', 'u': 'ū',
		'A': 'Ā', 'E': 'Ē', 'I': 'Ī', 'O': 'Ō', 'U': 'Ū',
	},
	".": { // dot above
		'z': 'ż', 'Z': 'Ż',
	},
	"c": { // cedilla
		'c': 'ç', 'C': 'Ç', 's': 'ş', 'S': 'Ş',
	},
	"v": { // caron/háček
		'c': 'č', 'C': 'Č', 's': 'š', 'S': 'Š', 'z': 'ž', 'Z': 'Ž',
		'r': 'ř', 'R': 'Ř', 'e': 'ě', 'E': 'Ě', 'n': 'ň', 'N': 'Ň',
	},
	"u": { // breve
		'a': 'ă', 'A': 'Ă',
	},
	"H": { // double acute
		'o': 'ő', 'O': 'Ő', 'u': 'ű', 'U': 'Ű',
	},
}

// Matches {\"a}, {\`e}, {\'o}, etc. and also \"a, \'e without braces
var latexAccentRe = regexp.MustCompile(`\{\\([\"'\x60^~=.cuvH])(?:\{([a-zA-Z])\}|([a-zA-Z]))\}|\\([\"'\x60^~=.cuvH])(?:\{([a-zA-Z])\}|([a-zA-Z]))`)

// Other common LaTeX commands
var latexSpecial = map[string]string{
	`\aa`:  "å", `\AA`: "Å",
	`\ae`:  "æ", `\AE`: "Æ",
	`\oe`:  "œ", `\OE`: "Œ",
	`\o`:   "ø", `\O`:  "Ø",
	`\ss`:  "ß",
	`\i`:   "ı",
	`\l`:   "ł", `\L`: "Ł",
	`\&`:   "&",
	`\#`:   "#",
	`\%`:   "%",
	`\$`:   "$",
	`\_`:   "_",
	`\{`:   "{",
	`\}`:   "}",
	`\textendash`:  "–",
	`\textemdash`:  "—",
}

// LaTeXToUTF8 converts LaTeX accent commands to UTF-8 characters.
func LaTeXToUTF8(s string) string {
	// Replace accent patterns
	s = latexAccentRe.ReplaceAllStringFunc(s, func(match string) string {
		sub := latexAccentRe.FindStringSubmatch(match)
		// Groups: 1=braced cmd, 2=braced letter in braces, 3=braced letter bare,
		//         4=bare cmd, 5=bare letter in braces, 6=bare letter bare
		var cmd string
		var letter rune
		if sub[1] != "" {
			cmd = sub[1]
			if sub[2] != "" {
				letter = rune(sub[2][0])
			} else {
				letter = rune(sub[3][0])
			}
		} else {
			cmd = sub[4]
			if sub[5] != "" {
				letter = rune(sub[5][0])
			} else {
				letter = rune(sub[6][0])
			}
		}
		if m, ok := latexAccents[cmd]; ok {
			if r, ok := m[letter]; ok {
				return string(r)
			}
		}
		return match
	})

	// Replace special commands (longer ones first to avoid partial matches)
	for _, cmd := range []string{
		`\textendash`, `\textemdash`,
		`\aa`, `\AA`, `\ae`, `\AE`, `\oe`, `\OE`, `\ss`,
		`\o`, `\O`, `\i`, `\l`, `\L`,
		`\&`, `\#`, `\%`, `\$`, `\_`, `\{`, `\}`,
	} {
		if strings.Contains(s, cmd) {
			s = strings.ReplaceAll(s, cmd, latexSpecial[cmd])
		}
	}

	return s
}

// ConvertLaTeX converts LaTeX accent commands to UTF-8 in all fields of an entry.
func ConvertLaTeX(e *Entry) {
	for i := range e.Fields {
		e.Fields[i].Value = LaTeXToUTF8(e.Fields[i].Value)
	}
}
