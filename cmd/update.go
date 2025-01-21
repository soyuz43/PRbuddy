// cmd/update.go

package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Generate a pull request draft based on the latest changes.",
	Long:  `Parses Git diffs (staged, unstaged, and untracked) and generates a pull request draft.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[PRBuddy-Go] Running update command...")

		// 1. Gather local diffs (staged, unstaged, and untracked)
		stagedDiff, err := utils.ExecuteGitCommand("diff", "--cached", "HEAD")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error getting staged diff: %v\n", err)
			return
		}
		unstagedDiff, err := utils.ExecuteGitCommand("diff", "HEAD")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error getting unstaged diff: %v\n", err)
			return
		}
		untrackedFiles, err := utils.ExecuteGitCommand("ls-files", "--others", "--exclude-standard")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error getting untracked files: %v\n", err)
			return
		}

		// Combine the diffs into a single string
		fullDiffs := ""
		if stagedDiff != "" {
			fullDiffs += fmt.Sprintf("--- Staged Changes ---\n%s\n\n", stagedDiff)
		}
		if unstagedDiff != "" {
			fullDiffs += fmt.Sprintf("--- Unstaged Changes ---\n%s\n\n", unstagedDiff)
		}
		if untrackedFiles != "" {
			fullDiffs += fmt.Sprintf("--- Untracked Files ---\n%s\n\n", untrackedFiles)
		}

		// If there's nothing to show, we stop
		if fullDiffs == "" {
			fmt.Println("[PRBuddy-Go] No changes detected. No pull request draft generated.")
			return
		}

		// 2. Generate PR draft via LLM
		draftPR, err := llm.GenerateDraftPR(fullDiffs, "")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error generating PR draft: %v\n", err)
			return
		}

		// 3. Display the draft PR
		fmt.Println("\n**Pull Request Draft Generated:**")
		fmt.Println(draftPR)

		fmt.Println("\n[PRBuddy-Go] Update process complete.")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
