package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/soyuz43/prbuddy-go/internal/hooks"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize PRBuddy-Go in the current Git repository.",
	Long:  `Installs a post-commit hook and creates the .git/pr_buddy_db directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[PRBuddy-Go] Initializing PRBuddy-Go...")

		// 1. Install post-commit hook
		if err := hooks.InstallPostCommitHook(); err != nil {
			fmt.Printf("[PRBuddy-Go] Error installing post-commit hook: %v\n", err)
			return
		}
		fmt.Println("[PRBuddy-Go] Post-commit hook installation complete.")

		// 2. Create pr_buddy_db directory
		repoPath, err := utils.GetRepoPath()
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error retrieving repository path: %v\n", err)
			return
		}

		prBuddyDBPath := filepath.Join(repoPath, ".git", "pr_buddy_db")
		err = os.MkdirAll(prBuddyDBPath, 0750)
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error creating pr_buddy_db directory: %v\n", err)
			return
		}

		fmt.Printf("[PRBuddy-Go] Created directory: %s\n", prBuddyDBPath)
		fmt.Println("[PRBuddy-Go] Initialization complete.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
