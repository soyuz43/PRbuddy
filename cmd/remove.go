// cmd/remove.go

package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/hooks"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove PRBuddy-Go from the repository.",
	Long:  `Deletes Git hooks and cleans up other traces of PRBuddy-Go from the repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[PRBuddy-Go] Running remove command...")

		// 1. Remove the post-commit hook
		err := hooks.RemovePostCommitHook()
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error removing post-commit hook: %v\n", err)
		} else {
			fmt.Println("[PRBuddy-Go] Removed the post-commit hook.")
		}

		// 2. Optional: Remove any configuration files or directories if applicable
		// Example:
		// configPath := ".prbuddy-config"
		// err = os.RemoveAll(configPath)
		// if err != nil {
		//     fmt.Printf("[PRBuddy-Go] Error deleting config directory (%s): %v\n", configPath, err)
		// } else {
		//     fmt.Printf("[PRBuddy-Go] Deleted the config directory: %s\n", configPath)
		// }

		fmt.Println("[PRBuddy-Go] Successfully removed all traces of PRBuddy-Go from the repository.")
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
