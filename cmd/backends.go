package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var backendsCmd = &cobra.Command{
	Use:   "backends",
	Short: "List configured backends",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := internal.LoadConfig()
		if err != nil {
			return err
		}

		if len(cfg.Backends) == 0 {
			fmt.Println("No backends configured. Run 'bibdb init <path>' to create one.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPATH\tREMOTE\tBRANCH\tDEFAULT")
		fmt.Fprintln(w, "----\t----\t------\t------\t-------")

		for name, b := range cfg.Backends {
			def := ""
			if name == cfg.Default {
				def = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", name, b.Path, b.Remote, b.Branch, def)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backendsCmd)
}
