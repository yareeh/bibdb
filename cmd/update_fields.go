package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var updateFieldsFields []string

var updateFieldsCmd = &cobra.Command{
	Use:   "update-fields <key>",
	Short: "Set or update individual fields of an entry",
	Long: `Set or update specific fields of an existing entry without touching others.

  bibdb update-fields smith2024 --field year=2025 --field publisher="MIT Press"`,
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

		if len(updateFieldsFields) == 0 {
			return fmt.Errorf("provide at least one --field key=value")
		}

		entry, err := store.Read(key)
		if err != nil {
			return err
		}

		for _, f := range updateFieldsFields {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid field format %q, expected key=value", f)
			}
			entry.Set(parts[0], parts[1])
		}

		if err := store.Write(entry); err != nil {
			return err
		}

		if err := repo.SyncMutation(
			[]string{store.RelPath(key)},
			fmt.Sprintf("bibdb: update-fields %s", key),
		); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git sync: %v\n", err)
		}

		fmt.Printf("Updated %s\n", key)
		return nil
	},
}

func init() {
	updateFieldsCmd.Flags().StringSliceVar(&updateFieldsFields, "field", nil, "field in key=value format (repeatable)")
	rootCmd.AddCommand(updateFieldsCmd)
}
