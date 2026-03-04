package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreWriteRead(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	e := &Entry{
		Type: "book",
		Key:  "smith2019spring",
		Fields: []Field{
			{Name: "author", Value: "Smith, Ali"},
			{Name: "title", Value: "Spring"},
			{Name: "year", Value: "2019"},
		},
	}

	if err := s.Write(e); err != nil {
		t.Fatal(err)
	}

	// Check file exists in correct shard
	path := filepath.Join(dir, "entries", "sm", "smith2019spring.bib")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not at expected path: %s", path)
	}

	got, err := s.Read("smith2019spring")
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "smith2019spring" {
		t.Errorf("key = %q", got.Key)
	}
	if got.Get("author") != "Smith, Ali" {
		t.Errorf("author = %q", got.Get("author"))
	}
}

func TestStoreShardCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	e := &Entry{Type: "book", Key: "McComb2019", Fields: []Field{{Name: "title", Value: "Test"}}}
	if err := s.Write(e); err != nil {
		t.Fatal(err)
	}

	// Shard should be "mc" (lowercase)
	path := filepath.Join(dir, "entries", "mc", "McComb2019.bib")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not at expected shard path: %s", path)
	}
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	e := &Entry{Type: "book", Key: "test2020", Fields: []Field{{Name: "title", Value: "Test"}}}
	s.Write(e)

	if !s.Exists("test2020") {
		t.Fatal("entry should exist")
	}

	if err := s.Delete("test2020"); err != nil {
		t.Fatal(err)
	}

	if s.Exists("test2020") {
		t.Fatal("entry should not exist")
	}
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	for _, key := range []string{"alpha2020", "beta2021", "gamma2022"} {
		s.Write(&Entry{Type: "book", Key: key, Fields: []Field{{Name: "title", Value: key}}})
	}

	entries, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestStoreRelPath(t *testing.T) {
	s := NewStore("/data")
	rel := s.RelPath("smith2019")
	if rel != filepath.Join("entries", "sm", "smith2019.bib") {
		t.Errorf("relpath = %q", rel)
	}
}
