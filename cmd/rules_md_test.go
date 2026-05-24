package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yareeh/bibdb/internal/fixrules"
)

// TestRulesMdInSync fails when the on-disk Rules.md doesn't match the
// content generated from the rule registry. Regenerate with:
//
//	bibdb fix --list-rules --markdown > Rules.md
//
// This trips immediately if anyone adds a rule and forgets the doc.
func TestRulesMdInSync(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(here), "..", "Rules.md")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	want := generateRulesMarkdown(fixrules.All())
	if string(got) != want {
		t.Fatalf("Rules.md is out of sync with the rule registry.\nRegenerate with:\n\n  bibdb fix --list-rules --markdown > Rules.md\n\nDiff (got vs want):\n=== got ===\n%s\n=== want ===\n%s", got, want)
	}
}
