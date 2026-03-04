package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var backendFlag string

var rootCmd = &cobra.Command{
	Use:   "bibdb",
	Short: "Git-backed BibTeX database manager",
	Long:  "bibdb manages BibTeX entries in git-backed repositories with sharded file storage.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&backendFlag, "backend", "", "backend to use (default: from config)")
}

func resolveBackend() (*internal.Backend, error) {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return nil, err
	}

	name := backendFlag
	if name == "" {
		name = os.Getenv("BIBDB_BACKEND")
	}
	if name == "" {
		name = cfg.Default
	}
	if name == "" {
		return nil, fmt.Errorf("no backend configured; run 'bibdb init <path>' first")
	}

	b, ok := cfg.Backends[name]
	if !ok {
		return nil, fmt.Errorf("backend %q not found in config", name)
	}
	return &b, nil
}
