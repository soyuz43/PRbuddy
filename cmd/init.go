package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/hooks"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize PRBuddy-Go in the current Git repository.",
	Long: `Installs a post-commit hook (optionally) and creates the .git/pr_buddy_db directory.
If you choose not to install the post-commit hook now, you can install it later manually.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[PRBuddy-Go] Initializing PRBuddy-Go...")

		// 1. Prompt the user about installing the post-commit hook
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("[PRBuddy-Go] Generate pr automatically on commit?  [y/N] ")

		userInput, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error reading input: %v\n", err)
			// Fall back to skipping the hook if there's an I/O error
			userInput = "n"
		}
		userInput = strings.TrimSpace(strings.ToLower(userInput))

		if userInput == "y" || userInput == "yes" {
			// Attempt to install the post-commit hook
			if err := hooks.InstallPostCommitHook(); err != nil {
				fmt.Printf("[PRBuddy-Go] Error installing post-commit hook: %v\n", err)
			} else {
				fmt.Println("[PRBuddy-Go] Post-commit hook installation complete.")
			}
		} else {
			fmt.Println("[PRBuddy-Go] Skipping post-commit hook installation.")
		}

		// 2. Create .git/pr_buddy_db directory
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
