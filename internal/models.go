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
