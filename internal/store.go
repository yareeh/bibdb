package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	Root       string // data repo root
	EntriesDir string // typically "entries"
}

func NewStore(root string) *Store {
	return &Store{Root: root, EntriesDir: "entries"}
}

func (s *Store) entriesPath() string {
	return filepath.Join(s.Root, s.EntriesDir)
}

func (s *Store) entryPath(key string) string {
	shard := strings.ToLower(key)
	if len(shard) < 2 {
		shard = shard + strings.Repeat("_", 2-len(shard))
	} else {
		shard = shard[:2]
	}
	return filepath.Join(s.entriesPath(), shard, key+".bib")
}

// RelPath returns the path relative to the data repo root.
func (s *Store) RelPath(key string) string {
	p := s.entryPath(key)
	rel, err := filepath.Rel(s.Root, p)
	if err != nil {
		return p
	}
	return rel
}

func (s *Store) Read(key string) (*Entry, error) {
	data, err := os.ReadFile(s.entryPath(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("entry %q not found", key)
		}
		return nil, err
	}
	entries, err := Parse(string(data))
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entry found in file for key %q", key)
	}
	return &entries[0], nil
}

func (s *Store) Write(e *Entry) error {
	if err := ValidateKey(e.Key); err != nil {
		return err
	}
	path := s.entryPath(e.Key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(FormatEntry(e)), 0o644)
}

func (s *Store) Delete(key string) error {
	path := s.entryPath(key)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	// Try to remove empty shard directory
	dir := filepath.Dir(path)
	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		os.Remove(dir)
	}
	return nil
}

func (s *Store) Exists(key string) bool {
	_, err := os.Stat(s.entryPath(key))
	return err == nil
}

func (s *Store) List() ([]*Entry, error) {
	var result []*Entry
	base := s.entriesPath()
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return result, nil
	}

	shards, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}

	for _, shard := range shards {
		if !shard.IsDir() {
			continue
		}
		shardPath := filepath.Join(base, shard.Name())
		files, err := os.ReadDir(shardPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".bib") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(shardPath, f.Name()))
			if err != nil {
				continue
			}
			entries, err := Parse(string(data))
			if err != nil || len(entries) == 0 {
				continue
			}
			result = append(result, &entries[0])
		}
	}
	return result, nil
}
