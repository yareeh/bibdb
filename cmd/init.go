package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init <path>",
	Short: "Initialize a new data repo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := filepath.Abs(internal.ExpandPath(args[0]))
		if err != nil {
			return err
		}

		// Create directory structure
		if err := os.MkdirAll(filepath.Join(path, "entries"), 0o755); err != nil {
			return err
		}

		// Create marker file
		marker := map[string]string{"entries_dir": "entries"}
		data, _ := yaml.Marshal(marker)
		if err := os.WriteFile(filepath.Join(path, ".bibdb.yaml"), data, 0o644); err != nil {
			return err
		}

		// Initialize git repo
		repo := internal.NewRepo(path)
		if !repo.IsRepo() {
			if err := repo.Init(); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			fmt.Println("Initialized git repository")
		}

		// Initial commit
		repo.Add(".")
		repo.Commit("bibdb: initialize data repo")

		// Register as backend in config
		cfg, err := internal.LoadConfig()
		if err != nil {
			return err
		}

		name := filepath.Base(path)
		cfg.Backends[name] = internal.Backend{
			Path:   path,
			Remote: "origin",
			Branch: "main",
		}
		if cfg.Default == "" {
			cfg.Default = name
		}

		if err := internal.SaveConfig(cfg); err != nil {
			return err
		}

		fmt.Printf("Initialized bibdb data repo at %s\n", path)
		fmt.Printf("Registered as backend %q\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
