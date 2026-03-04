package internal

import (
	"fmt"
	"strings"
	"unicode"
)

// Parse parses one or more BibTeX entries from a string.
func Parse(input string) ([]Entry, error) {
	p := &parser{input: input}
	return p.parseAll()
}

type parser struct {
	input string
	pos   int
}

func (p *parser) parseAll() ([]Entry, error) {
	var entries []Entry
	for {
		p.skipNonEntry()
		if p.pos >= len(p.input) {
			break
		}
		if p.input[p.pos] != '@' {
			p.pos++
			continue
		}
		entry, err := p.parseEntry()
		if err != nil {
			return entries, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (p *parser) skipNonEntry() {
	for p.pos < len(p.input) {
		if p.input[p.pos] == '%' {
			// Skip comment line
			for p.pos < len(p.input) && p.input[p.pos] != '\n' {
				p.pos++
			}
			continue
		}
		if p.input[p.pos] == '@' {
			return
		}
		if unicode.IsSpace(rune(p.input[p.pos])) {
			p.pos++
			continue
		}
		// Skip any other non-@ character
		p.pos++
	}
}

func (p *parser) parseEntry() (Entry, error) {
	var e Entry
	p.pos++ // skip @

	// Read entry type
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != '{' && !unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
	e.Type = strings.ToLower(p.input[start:p.pos])

	p.skipWhitespace()
	if p.pos >= len(p.input) || p.input[p.pos] != '{' {
		return e, fmt.Errorf("expected '{' after entry type %q at position %d", e.Type, p.pos)
	}
	p.pos++ // skip {

	// Read cite key
	p.skipWhitespace()
	start = p.pos
	for p.pos < len(p.input) && p.input[p.pos] != ',' && p.input[p.pos] != '}' {
		p.pos++
	}
	e.Key = strings.TrimSpace(p.input[start:p.pos])

	if p.pos < len(p.input) && p.input[p.pos] == ',' {
		p.pos++ // skip ,
	}

	// Read fields
	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}
		if p.input[p.pos] == '}' {
			p.pos++
			break
		}

		// Read field name
		start = p.pos
		for p.pos < len(p.input) && p.input[p.pos] != '=' && p.input[p.pos] != '}' && !unicode.IsSpace(rune(p.input[p.pos])) {
			p.pos++
		}
		fieldName := strings.TrimSpace(p.input[start:p.pos])
		if fieldName == "" {
			if p.pos < len(p.input) {
				p.pos++
			}
			continue
		}

		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] != '=' {
			// Might be trailing content before }
			continue
		}
		p.pos++ // skip =
		p.skipWhitespace()

		// Read field value
		value, err := p.parseValue()
		if err != nil {
			return e, fmt.Errorf("parsing field %q: %w", fieldName, err)
		}

		e.Fields = append(e.Fields, Field{Name: fieldName, Value: value})

		p.skipWhitespace()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++
		}
	}

	return e, nil
}

func (p *parser) parseValue() (string, error) {
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("unexpected end of input")
	}

	switch p.input[p.pos] {
	case '{':
		return p.parseBraced()
	case '"':
		return p.parseQuoted()
	default:
		// Bare value (number or macro)
		start := p.pos
		for p.pos < len(p.input) && p.input[p.pos] != ',' && p.input[p.pos] != '}' && !unicode.IsSpace(rune(p.input[p.pos])) {
			p.pos++
		}
		return strings.TrimSpace(p.input[start:p.pos]), nil
	}
}

func (p *parser) parseBraced() (string, error) {
	p.pos++ // skip opening {
	depth := 1
	var b strings.Builder
	for p.pos < len(p.input) && depth > 0 {
		ch := p.input[p.pos]
		if ch == '{' {
			depth++
			b.WriteByte(ch)
		} else if ch == '}' {
			depth--
			if depth > 0 {
				b.WriteByte(ch)
			}
		} else {
			b.WriteByte(ch)
		}
		p.pos++
	}
	if depth != 0 {
		return b.String(), fmt.Errorf("unmatched brace")
	}
	return b.String(), nil
}

func (p *parser) parseQuoted() (string, error) {
	p.pos++ // skip opening "
	var b strings.Builder
	braceDepth := 0
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == '{' {
			braceDepth++
			b.WriteByte(ch)
		} else if ch == '}' {
			braceDepth--
			b.WriteByte(ch)
		} else if ch == '"' && braceDepth == 0 {
			p.pos++
			return b.String(), nil
		} else {
			b.WriteByte(ch)
		}
		p.pos++
	}
	return b.String(), fmt.Errorf("unmatched quote")
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}
