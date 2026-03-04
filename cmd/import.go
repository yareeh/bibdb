package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var importCmd = &cobra.Command{
	Use:   "import <file.bib>",
	Short: "Import a monolithic .bib file into the current backend",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, err := resolveBackend()
		if err != nil {
			return err
		}

		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}

		entries, err := internal.Parse(string(data))
		if err != nil {
			return fmt.Errorf("parsing %s: %w", args[0], err)
		}

		store := internal.NewStore(backend.Path)
		repo := internal.NewRepo(backend.Path)

		repo.Pull()

		var paths []string
		var skipped int
		for i := range entries {
			internal.ConvertLaTeX(&entries[i])
			e := entries[i]
			if store.Exists(e.Key) {
				skipped++
				continue
			}
			if err := store.Write(&e); err != nil {
				fmt.Fprintf(os.Stderr, "warning: writing %s: %v\n", e.Key, err)
				continue
			}
			paths = append(paths, store.RelPath(e.Key))
		}

		fmt.Printf("Imported %d entries (%d skipped)\n", len(paths), skipped)

		if len(paths) > 0 {
			repo.Add(".")
			repo.Commit(fmt.Sprintf("bibdb: import %d entries", len(paths)))
			repo.Push()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
