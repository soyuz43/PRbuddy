// cmd/init.go

package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/hooks"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize PRBuddy in the current Git repository.",
	Long:  `Installs a post-commit hook to enable ephemeral PRBuddy features for local diffs.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[prbuddy-go] Initializing PRBuddy...")

		// 1. Install the post-commit hook
		if err := hooks.InstallPostCommitHook(); err != nil {
			fmt.Printf("[prbuddy-go] Error installing post-commit hook: %v\n", err)
			return
		}
		fmt.Println("[prbuddy-go] Post-commit hook installation complete.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
