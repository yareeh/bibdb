package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var removeFieldsFields []string

var removeFieldsCmd = &cobra.Command{
	Use:   "remove-fields <key>",
	Short: "Remove individual fields from an entry",
	Long: `Remove specific fields from an existing entry.

  bibdb remove-fields smith2024 --field note --field url`,
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

		if len(removeFieldsFields) == 0 {
			return fmt.Errorf("provide at least one --field name to remove")
		}

		entry, err := store.Read(key)
		if err != nil {
			return err
		}

		toRemove := make(map[string]bool, len(removeFieldsFields))
		for _, f := range removeFieldsFields {
			toRemove[strings.ToLower(f)] = true
		}

		filtered := entry.Fields[:0]
		for _, f := range entry.Fields {
			if !toRemove[strings.ToLower(f.Name)] {
				filtered = append(filtered, f)
			}
		}
		entry.Fields = filtered

		if err := store.Write(entry); err != nil {
			return err
		}

		if err := repo.SyncMutation(
			[]string{store.RelPath(key)},
			fmt.Sprintf("bibdb: remove-fields %s", key),
		); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git sync: %v\n", err)
		}

		fmt.Printf("Updated %s\n", key)
		return nil
	},
}

func init() {
	removeFieldsCmd.Flags().StringSliceVar(&removeFieldsFields, "field", nil, "field name to remove (repeatable)")
	rootCmd.AddCommand(removeFieldsCmd)
}
