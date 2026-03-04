package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old-key> <new-key>",
	Short: "Rename a cite key",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, err := resolveBackend()
		if err != nil {
			return err
		}

		oldKey, newKey := args[0], args[1]
		store := internal.NewStore(backend.Path)
		repo := internal.NewRepo(backend.Path)

		entry, err := store.Read(oldKey)
		if err != nil {
			return err
		}

		if store.Exists(newKey) {
			return fmt.Errorf("entry %q already exists", newKey)
		}

		oldRelPath := store.RelPath(oldKey)
		entry.Key = newKey

		if err := store.Write(entry); err != nil {
			return err
		}
		if err := store.Delete(oldKey); err != nil {
			return err
		}

		if err := repo.SyncMutation(
			[]string{oldRelPath, store.RelPath(newKey)},
			fmt.Sprintf("bibdb: rename %s -> %s", oldKey, newKey),
		); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git sync: %v\n", err)
		}

		fmt.Printf("Renamed %s -> %s\n", oldKey, newKey)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
