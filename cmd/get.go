package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print a BibTeX entry to stdout",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, err := resolveBackend()
		if err != nil {
			return err
		}

		store := internal.NewStore(backend.Path)
		entry, err := store.Read(args[0])
		if err != nil {
			return err
		}

		fmt.Print(internal.FormatEntry(entry))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
