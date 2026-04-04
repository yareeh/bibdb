package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yareeh/bibdb/internal"
	"gopkg.in/yaml.v3"
)

// setupTestBackend creates a temp dir with a config pointing to it and seeds it with entries.
func setupTestBackend(t *testing.T, entries []*internal.Entry) (store *internal.Store, cleanup func()) {
	t.Helper()

	dataDir := t.TempDir()
	configDir := t.TempDir()

	cfg := map[string]any{
		"default": "test",
		"backends": map[string]any{
			"test": map[string]any{
				"path": dataDir,
			},
		},
	}
	data, _ := yaml.Marshal(cfg)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), data, 0o644)

	t.Setenv("BIBDB_CONFIG_DIR", configDir)

	store = internal.NewStore(dataDir)
	for _, e := range entries {
		if err := store.Write(e); err != nil {
			t.Fatal(err)
		}
	}

	// Reset cobra flags between tests
	backendFlag = ""

	return store, func() {}
}

func runCmd(args ...string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func TestUpdateFromFlags(t *testing.T) {
	store, _ := setupTestBackend(t, []*internal.Entry{
		{
			Type: "book",
			Key:  "smith2020",
			Fields: []internal.Field{
				{Name: "title", Value: "Old Title"},
				{Name: "year", Value: "2020"},
			},
		},
	})

	err := runCmd("update", "smith2020", "--type", "article", "--field", fmt.Sprintf("title=%s", "New Title"), "--field", "year=2021")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	got, err := store.Read("smith2020")
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "smith2020" {
		t.Errorf("key changed: %q", got.Key)
	}
	if got.Type != "article" {
		t.Errorf("type = %q, want article", got.Type)
	}
	if got.Get("title") != "New Title" {
		t.Errorf("title = %q", got.Get("title"))
	}
	if got.Get("year") != "2021" {
		t.Errorf("year = %q", got.Get("year"))
	}
}

func TestUpdateNotFound(t *testing.T) {
	setupTestBackend(t, nil)

	err := runCmd("update", "missing2020", "--field", "title=X")
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestUpdateFieldsSet(t *testing.T) {
	store, _ := setupTestBackend(t, []*internal.Entry{
		{
			Type: "book",
			Key:  "jones2021",
			Fields: []internal.Field{
				{Name: "title", Value: "Original"},
				{Name: "year", Value: "2021"},
			},
		},
	})

	err := runCmd("update-fields", "jones2021", "--field", "year=2022", "--field", "note=updated")
	if err != nil {
		t.Fatalf("update-fields failed: %v", err)
	}

	got, err := store.Read("jones2021")
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("title") != "Original" {
		t.Errorf("title should be unchanged, got %q", got.Get("title"))
	}
	if got.Get("year") != "2022" {
		t.Errorf("year = %q, want 2022", got.Get("year"))
	}
	if got.Get("note") != "updated" {
		t.Errorf("note = %q, want updated", got.Get("note"))
	}
}

func TestUpdateFieldsNotFound(t *testing.T) {
	setupTestBackend(t, nil)

	err := runCmd("update-fields", "nope2020", "--field", "year=2020")
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestRemoveFields(t *testing.T) {
	store, _ := setupTestBackend(t, []*internal.Entry{
		{
			Type: "book",
			Key:  "white2022",
			Fields: []internal.Field{
				{Name: "title", Value: "A Title"},
				{Name: "year", Value: "2022"},
				{Name: "note", Value: "to remove"},
				{Name: "url", Value: "http://example.com"},
			},
		},
	})

	err := runCmd("remove-fields", "white2022", "--field", "note", "--field", "url")
	if err != nil {
		t.Fatalf("remove-fields failed: %v", err)
	}

	got, err := store.Read("white2022")
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("title") != "A Title" {
		t.Errorf("title changed unexpectedly")
	}
	if got.Get("note") != "" {
		t.Errorf("note should be removed, got %q", got.Get("note"))
	}
	if got.Get("url") != "" {
		t.Errorf("url should be removed, got %q", got.Get("url"))
	}
}

func TestRemoveFieldsNotFound(t *testing.T) {
	setupTestBackend(t, nil)

	err := runCmd("remove-fields", "nope2020", "--field", "note")
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}
