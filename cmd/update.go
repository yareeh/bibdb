package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var updateType string
var updateFields []string

var updateCmd = &cobra.Command{
	Use:   "update <key>",
	Short: "Rewrite an entry's type and fields, preserving its cite key",
	Long: `Replace all fields of an existing entry. The cite key is preserved.
Provide new content via stdin (BibTeX) or flags:

  echo '@book{smith2024, title={New Title}}' | bibdb update smith2024
  bibdb update smith2024 --type article --field title="New Title" --field year=2024`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, err := resolveBackend()
		if err != nil {
			return err
		}

		key := args[0]
		store := internal.NewStore(backend.Path)
		repo := internal.NewRepo(backend.Path)

		if !store.Exists(key) {
			return fmt.Errorf("entry %q not found", key)
		}

		var entry *internal.Entry

		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
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
			if !cmd.Flags().Changed("type") && len(updateFields) == 0 {
				return fmt.Errorf("provide --type and/or --field flags, or pipe BibTeX via stdin")
			}
			existing, err := store.Read(key)
			if err != nil {
				return err
			}
			entryType := existing.Type
			if cmd.Flags().Changed("type") {
				entryType = updateType
			}
			entry = &internal.Entry{Type: entryType, Key: key}
			for _, f := range updateFields {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid field format %q, expected key=value", f)
				}
				entry.Fields = append(entry.Fields, internal.Field{Name: parts[0], Value: parts[1]})
			}
		}

		// Always preserve the original cite key
		entry.Key = key

		if err := store.Write(entry); err != nil {
			return err
		}

		if err := repo.SyncMutation(
			[]string{store.RelPath(key)},
			fmt.Sprintf("bibdb: update %s", key),
		); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git sync: %v\n", err)
		}

		fmt.Printf("Updated %s\n", key)
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateType, "type", "", "entry type (book, article, etc.)")
	updateCmd.Flags().StringSliceVar(&updateFields, "field", nil, "field in key=value format (repeatable)")
	rootCmd.AddCommand(updateCmd)
}
