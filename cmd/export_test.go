package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yareeh/bibdb/internal"
	"github.com/yareeh/bibdb/internal/vocab"
)

func TestWriteEntityStubs(t *testing.T) {
	vocab.SetActive(nil)
	dir := t.TempDir()
	exportOutput = dir
	entries := []*internal.Entry{
		{Type: "article", Key: "a", Fields: []internal.Field{
			{Name: "author", Value: "Hattenstone, Simon"},
			{Name: "journal", Value: "The Guardian"},
		}},
		{Type: "article", Key: "b", Fields: []internal.Field{
			{Name: "author", Value: "Nelson, Eshe"},
			{Name: "journal", Value: "The Guardian"}, // shared → one stub
		}},
	}

	if err := writeEntityStubs(entries); err != nil {
		t.Fatalf("writeEntityStubs: %v", err)
	}
	entDir := filepath.Join(dir, "Entities")
	for _, name := range []string{"Simon Hattenstone", "Eshe Nelson", "The Guardian"} {
		if _, err := os.Stat(filepath.Join(entDir, name+".md")); err != nil {
			t.Errorf("expected stub %q.md: %v", name, err)
		}
	}

	// Existing stub with user content must NOT be overwritten on re-run.
	guardian := filepath.Join(entDir, "The Guardian.md")
	if err := os.WriteFile(guardian, []byte("# The Guardian\n\nMy notes about the paper.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeEntityStubs(entries); err != nil {
		t.Fatalf("writeEntityStubs 2: %v", err)
	}
	got, _ := os.ReadFile(guardian)
	if string(got) != "# The Guardian\n\nMy notes about the paper.\n" {
		t.Errorf("existing annotated stub was overwritten:\n%s", got)
	}
}
