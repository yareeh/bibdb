package internal

import (
	"fmt"
	"regexp"
	"strings"
)

type Field struct {
	Name  string
	Value string
}

type Entry struct {
	Type   string  // "book", "article", "misc", etc.
	Key    string  // cite key
	Fields []Field // ordered for round-trip fidelity
}

func (e *Entry) Get(name string) string {
	lower := strings.ToLower(name)
	for _, f := range e.Fields {
		if strings.ToLower(f.Name) == lower {
			return f.Value
		}
	}
	return ""
}

func (e *Entry) Set(name, value string) {
	lower := strings.ToLower(name)
	for i, f := range e.Fields {
		if strings.ToLower(f.Name) == lower {
			e.Fields[i].Value = value
			return
		}
	}
	e.Fields = append(e.Fields, Field{Name: name, Value: value})
}

// ValidateKey checks that the cite key doesn't contain characters
// that are problematic in filesystems.
func ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("empty cite key")
	}
	for _, c := range key {
		switch c {
		case ':', '/', '\\', '<', '>', '"', '|', '?', '*', ' ', '\t', '\n':
			return fmt.Errorf("cite key %q contains invalid character %q", key, c)
		}
	}
	if key == "." || key == ".." {
		return fmt.Errorf("cite key %q is a reserved name", key)
	}
	return nil
}

// CiteKeyRe is the canonical cite-key format: lowercase ASCII letters, digits
// and underscores, starting with a letter. It is the single definition shared
// by the key-format fix rule and the write-time enforcement below.
var CiteKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ValidateKeyFormat enforces CiteKeyRe. It is stricter than ValidateKey (which
// only rejects filesystem-hostile characters) and is applied when creating or
// renaming entries, so a non-ASCII key — which would otherwise slice into an
// invalid, uncheckoutable multibyte shard directory — can never be persisted.
func ValidateKeyFormat(key string) error {
	if !CiteKeyRe.MatchString(key) {
		return fmt.Errorf("invalid cite key %q: must match %s (lowercase ASCII letters, digits and underscore; leading letter)", key, CiteKeyRe.String())
	}
	return nil
}

// Shard returns the storage shard for a key: its first two runes, lowercased,
// padded with underscores for very short keys. Rune-based rather than
// byte-based so a stray multibyte key can never produce an invalid directory
// name (e.g. "jäger" → "jä", never a lone 0xC3 byte).
func Shard(key string) string {
	r := []rune(strings.ToLower(key))
	switch {
	case len(r) == 0:
		return "__"
	case len(r) < 2:
		return string(r) + strings.Repeat("_", 2-len(r))
	default:
		return string(r[:2])
	}
}

// ShardKey returns the storage shard for this entry's key.
func (e *Entry) ShardKey() string {
	return Shard(e.Key)
}

// VersionFieldName is the BibTeX field name that records which bibdb release
// last created or fixed an entry. It is a single lowercase word for maximum
// parser compatibility.
const VersionFieldName = "bibdbversion"

// StampVersion writes (or updates in place) the bibdbversion metadata field
// on an entry. The field is the bookkeeping signal the fix command uses to
// skip entries that are already up to date.
func StampVersion(e *Entry, v string) {
	e.Set(VersionFieldName, v)
}

// EntryVersion returns the stored bibdbversion, or "0.0.0" when missing — so
// legacy entries (created before the field existed) are treated as needing
// every rule.
func EntryVersion(e *Entry) string {
	v := e.Get(VersionFieldName)
	if v == "" {
		return "0.0.0"
	}
	return v
}
