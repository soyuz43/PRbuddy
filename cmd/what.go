// cmd/what.go

package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/spf13/cobra"
)

var whatCmd = &cobra.Command{
	Use:   "what",
	Short: "Summarize recent changes since the last commit.",
	Long:  `Analyzes staged, unstaged, and untracked changes in the repository and provides a natural language summary using the LLM.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[prbuddy-go] Running 'what' command...")

		summary, err := llm.GenerateWhatSummary()
		if err != nil {
			if err.Error() == "no commits found in the repository" {
				fmt.Println("[prbuddy-go] No commits found in the repository. Please make a commit first.")
				return
			}
			fmt.Printf("[prbuddy-go] Error generating summary: %v\n", err)
			return
		}

		fmt.Println("\n**What Have I Done Since the Last Commit:**")
		fmt.Println(summary)
	},
}

func init() {
	rootCmd.AddCommand(whatCmd)
}
