// cmd/remove.go

package cmd

import (
	"fmt"
	"os"

	"github.com/soyuz43/prbuddy-go/internal/database"
	"github.com/soyuz43/prbuddy-go/internal/hooks"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove PRBuddy from the repository.",
	Long:  `Deletes the prbuddy.db file, removes the post-commit hook, and cleans up other traces of PRBuddy.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[prbuddy-go] Running remove command...")

		// 1. Delete the database
		err := database.DeleteDatabase("prbuddy.db")
		if err != nil {
			fmt.Printf("[prbuddy-go] Error deleting database: %v\n", err)
		} else {
			fmt.Println("[prbuddy-go] Deleted the database file: prbuddy.db")
		}

		// 2. Remove the post-commit hook
		err = hooks.RemovePostCommitHook()
		if err != nil {
			fmt.Printf("[prbuddy-go] Error removing post-commit hook: %v\n", err)
		} else {
			fmt.Println("[prbuddy-go] Removed the post-commit hook.")
		}

		// 3. Remove ChromaDB storage directory
		chromaDBPath := "chromadb_storage"
		err = os.RemoveAll(chromaDBPath)
		if err != nil {
			fmt.Printf("[prbuddy-go] Error deleting ChromaDB storage directory (%s): %v\n", chromaDBPath, err)
		} else {
			fmt.Printf("[prbuddy-go] Deleted the ChromaDB storage directory: %s\n", chromaDBPath)
		}

		fmt.Println("[prbuddy-go] Successfully removed all traces of PRBuddy from the repository.")
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
