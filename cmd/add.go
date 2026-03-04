package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var addType string
var addKey string
var addFields []string

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a BibTeX entry from flags or stdin",
	Long: `Add a BibTeX entry. Either pipe BibTeX via stdin:
  echo '@book{key, author={...}}' | bibdb add

Or use flags:
  bibdb add --type book --key smith2024 --field author="Smith, Ali" --field title="Test"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, err := resolveBackend()
		if err != nil {
			return err
		}

		store := internal.NewStore(backend.Path)
		repo := internal.NewRepo(backend.Path)

		var entry *internal.Entry

		// Check if stdin has data
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Reading from stdin
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
			var sb strings.Builder
			for scanner.Scan() {
				sb.WriteString(scanner.Text())
				sb.WriteString("\n")
			}
			entries, err := internal.Parse(sb.String())
			if err != nil {
				return fmt.Errorf("parsing stdin: %w", err)
			}
			if len(entries) == 0 {
				return fmt.Errorf("no entries found in stdin")
			}
			entry = &entries[0]
		} else {
			// Build from flags
			if addKey == "" {
				return fmt.Errorf("--key is required")
			}
			if addType == "" {
				addType = "misc"
			}
			entry = &internal.Entry{Type: addType, Key: addKey}
			for _, f := range addFields {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid field format %q, expected key=value", f)
				}
				entry.Fields = append(entry.Fields, internal.Field{Name: parts[0], Value: parts[1]})
			}
		}

		// Warn about missing required fields
		required := []string{"author", "title", "year"}
		for _, r := range required {
			if entry.Get(r) == "" {
				fmt.Fprintf(os.Stderr, "warning: missing field %q\n", r)
			}
		}

		if store.Exists(entry.Key) {
			return fmt.Errorf("entry %q already exists", entry.Key)
		}

		if err := store.Write(entry); err != nil {
			return err
		}

		if err := repo.SyncMutation(
			[]string{store.RelPath(entry.Key)},
			fmt.Sprintf("bibdb: add %s", entry.Key),
		); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git sync: %v\n", err)
		}

		fmt.Printf("Added %s\n", entry.Key)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addType, "type", "misc", "entry type (book, article, etc.)")
	addCmd.Flags().StringVar(&addKey, "key", "", "cite key")
	addCmd.Flags().StringSliceVar(&addFields, "field", nil, "field in key=value format (repeatable)")
	rootCmd.AddCommand(addCmd)
}
