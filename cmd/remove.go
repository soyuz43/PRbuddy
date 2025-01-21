// cmd/remove.go

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/soyuz43/prbuddy-go/internal/hooks"
	"github.com/soyuz43/prbuddy-go/internal/utils"
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

		// 2. Remove the .git/pr_buddy_db directory
		repoPath, err := utils.GetRepoPath()
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error retrieving repository path: %v\n", err)
			return
		}

		prBuddyDBPath := filepath.Join(repoPath, ".git", "pr_buddy_db")
		if _, err := os.Stat(prBuddyDBPath); !os.IsNotExist(err) {
			err = os.RemoveAll(prBuddyDBPath)
			if err != nil {
				fmt.Printf("[PRBuddy-Go] Error deleting pr_buddy_db directory: %v\n", err)
			} else {
				fmt.Printf("[PRBuddy-Go] Deleted directory: %s\n", prBuddyDBPath)
			}
		} else {
			fmt.Printf("[PRBuddy-Go] Directory does not exist: %s\n", prBuddyDBPath)
		}

		fmt.Println("[PRBuddy-Go] Successfully removed all traces of PRBuddy-Go from the repository.")
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
