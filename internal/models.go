package internal

import (
	"fmt"
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

// ShardKey returns the first 2 lowercase characters of the cite key.
func (e *Entry) ShardKey() string {
	key := strings.ToLower(e.Key)
	if len(key) < 2 {
		return key
	}
	return key[:2]
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
