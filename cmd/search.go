package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var searchField string

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search entries (case-insensitive substring)",
	Args:  cobra.ExactArgs(1),
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

		query := strings.ToLower(args[0])
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tTYPE\tYEAR\tAUTHOR\tTITLE")
		fmt.Fprintln(w, "---\t----\t----\t------\t-----")

		count := 0
		for _, e := range entries {
			if matchEntry(e, query, searchField) {
				title := e.Get("title")
				if len(title) > 60 {
					title = title[:57] + "..."
				}
				author := e.Get("author")
				if len(author) > 30 {
					author = author[:27] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.Key, e.Type, e.Get("year"), author, title)
				count++
			}
		}
		w.Flush()
		fmt.Fprintf(os.Stderr, "\n%d entries found\n", count)
		return nil
	},
}

func matchEntry(e *internal.Entry, query, field string) bool {
	if field != "" {
		return strings.Contains(strings.ToLower(e.Get(field)), query)
	}
	// Search key and all fields
	if strings.Contains(strings.ToLower(e.Key), query) {
		return true
	}
	for _, f := range e.Fields {
		if strings.Contains(strings.ToLower(f.Value), query) {
			return true
		}
	}
	return false
}

func init() {
	searchCmd.Flags().StringVar(&searchField, "field", "", "search only in specific field")
	rootCmd.AddCommand(searchCmd)
}
