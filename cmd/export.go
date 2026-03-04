package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var exportFormat string
var exportOutput string

var exportCmd = &cobra.Command{
	Use:   "export [key]",
	Short: "Export entries as markdown or concatenated .bib",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, err := resolveBackend()
		if err != nil {
			return err
		}

		store := internal.NewStore(backend.Path)

		var entries []*internal.Entry
		if len(args) == 1 {
			entry, err := store.Read(args[0])
			if err != nil {
				return err
			}
			entries = []*internal.Entry{entry}
		} else {
			entries, err = store.List()
			if err != nil {
				return err
			}
		}

		switch exportFormat {
		case "md":
			return exportMarkdown(entries)
		case "bib":
			return exportBib(entries)
		default:
			return fmt.Errorf("unknown format %q (use md or bib)", exportFormat)
		}
	},
}

func exportMarkdown(entries []*internal.Entry) error {
	if exportOutput == "" {
		return fmt.Errorf("--output is required for markdown export")
	}

	if err := os.MkdirAll(exportOutput, 0o755); err != nil {
		return err
	}

	for _, e := range entries {
		path := filepath.Join(exportOutput, e.Key+".md")
		if err := os.WriteFile(path, []byte(internal.FormatMarkdown(e)), 0o644); err != nil {
			return err
		}
	}

	fmt.Printf("Exported %d entries to %s\n", len(entries), exportOutput)
	return nil
}

func exportBib(entries []*internal.Entry) error {
	var output string
	for _, e := range entries {
		output += internal.FormatEntry(e) + "\n"
	}

	if exportOutput != "" {
		if err := os.WriteFile(exportOutput, []byte(output), 0o644); err != nil {
			return err
		}
		fmt.Printf("Exported %d entries to %s\n", len(entries), exportOutput)
	} else {
		fmt.Print(output)
	}
	return nil
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "bib", "output format (md or bib)")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "output directory (md) or file (bib)")
	rootCmd.AddCommand(exportCmd)
}
