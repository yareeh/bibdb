package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var listType string
var listYear string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List entries in table format",
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, err := resolveBackend()
		if err != nil {
			return err
		}

		store := internal.NewStore(backend.Path)
		entries, err := store.List()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tTYPE\tYEAR\tAUTHOR\tTITLE")
		fmt.Fprintln(w, "---\t----\t----\t------\t-----")

		for _, e := range entries {
			if listType != "" && !strings.EqualFold(e.Type, listType) {
				continue
			}
			if listYear != "" && e.Get("year") != listYear {
				continue
			}

			title := e.Get("title")
			if len(title) > 60 {
				title = title[:57] + "..."
			}
			author := e.Get("author")
			if len(author) > 30 {
				author = author[:27] + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.Key, e.Type, e.Get("year"), author, title)
		}
		w.Flush()
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listType, "type", "", "filter by entry type")
	listCmd.Flags().StringVar(&listYear, "year", "", "filter by year")
	rootCmd.AddCommand(listCmd)
}
