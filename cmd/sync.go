package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync with remote (git pull --rebase + push)",
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, err := resolveBackend()
		if err != nil {
			return err
		}

		repo := internal.NewRepo(backend.Path)
		if err := repo.Sync(); err != nil {
			return err
		}

		fmt.Println("Synced")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
