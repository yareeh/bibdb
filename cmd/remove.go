package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yareeh/bibdb/internal"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove <key>",
	Short: "Delete an entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, err := resolveBackend()
		if err != nil {
			return err
		}

		key := args[0]
		store := internal.NewStore(backend.Path)
		repo := internal.NewRepo(backend.Path)

		if !store.Exists(key) {
			return fmt.Errorf("entry %q not found", key)
		}

		if !removeForce {
			fmt.Printf("Remove %q? [y/N] ", key)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "y") {
				fmt.Println("Cancelled")
				return nil
			}
		}

		relPath := store.RelPath(key)
		if err := store.Delete(key); err != nil {
			return err
		}

		if err := repo.SyncMutation(
			[]string{relPath},
			fmt.Sprintf("bibdb: remove %s", key),
		); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git sync: %v\n", err)
		}

		fmt.Printf("Removed %s\n", key)
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVar(&removeForce, "force", false, "skip confirmation")
	rootCmd.AddCommand(removeCmd)
}
