// cmd/init.go

package cmd

import (
	"fmt"

	"os"
	"path/filepath"

	"github.com/soyuz43/prbuddy-go/internal/hooks"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize PRBuddy-Go in the current Git repository.",
	Long:  `Installs a post-commit hook and creates the .git/pr_buddy_db directory for future Pro features.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[prbuddy-go] Initializing PRBuddy-Go...")

		// 1. Install the post-commit hook
		if err := hooks.InstallPostCommitHook(); err != nil {
			fmt.Printf("[prbuddy-go] Error installing post-commit hook: %v\n", err)
			return
		}
		fmt.Println("[prbuddy-go] Post-commit hook installation complete.")

		// 2. Create the .git/pr_buddy_db directory
		repoPath, err := utils.GetRepoPath()
		if err != nil {
			fmt.Printf("[prbuddy-go] Error retrieving repository path: %v\n", err)
			return
		}

		prBuddyDBPath := filepath.Join(repoPath, ".git", "pr_buddy_db")
		err = os.MkdirAll(prBuddyDBPath, 0755)
		if err != nil {
			fmt.Printf("[prbuddy-go] Error creating pr_buddy_db directory: %v\n", err)
			return
		}

		fmt.Printf("[prbuddy-go] Created directory: %s\n", prBuddyDBPath)
		fmt.Println("[prbuddy-go] Initialization complete.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
