// cmd/what.go

package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

var whatCmd = &cobra.Command{
	Use:   "what",
	Short: "Summarize recent changes since the last commit.",
	Long: `Analyzes staged, unstaged, and untracked changes in the repository 
and provides a natural language summary using the LLM.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[PRBuddy-Go] Running 'what' command...")

		// Check if there are any commits in the repository
		commitCount, err := utils.ExecGit("rev-list", "--count", "HEAD")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error checking commits: %v\n", err)
			return
		}
		if commitCount == "0" {
			fmt.Println("[PRBuddy-Go] No commits found in the repository. Please make a commit first.")
			return
		}

		// Generate and display the summary
		summary, err := llm.GenerateWhatSummary()
		if err != nil {
			if err.Error() == "no changes detected since the last commit" {
				fmt.Println("[PRBuddy-Go] No changes detected.")
				return
			}
			fmt.Printf("[PRBuddy-Go] Error generating summary: %v\n", err)
			return
		}

		fmt.Println("\n**What Have I Done Since the Last Commit:**")
		fmt.Println(summary)
	},
}

func init() {
	rootCmd.AddCommand(whatCmd)
}
