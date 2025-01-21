// cmd/post_commit.go

package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/spf13/cobra"
)

// postCommitCmd represents the post-commit command
var postCommitCmd = &cobra.Command{
	Use:   "post-commit",
	Short: "Handle the post-commit hook.",
	Long:  `Generates a draft pull request based on the latest commit.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[prbuddy-go] Running post-commit logic...")

		// 1. Generate a draft PR from the latest commit
		commitMessage, diffs, err := llm.GeneratePreDraftPR()
		if err != nil {
			fmt.Printf("[prbuddy-go] Error generating pre-draft PR: %v\n", err)
			return
		}

		// 2. Generate draft PR via LLM
		draftPR, err := llm.GenerateDraftPR(commitMessage, diffs)
		if err != nil {
			fmt.Printf("[prbuddy-go] Error generating draft PR: %v\n", err)
			return
		}

		// Note: Embedding and ChromaDB querying steps have been removed.

		// 3. Display the draft PR
		fmt.Println("\n**Draft PR Generated:**")
		fmt.Println(draftPR)

		fmt.Println("\n[prbuddy-go] Post-commit processing complete.")
	},
}

func init() {
	rootCmd.AddCommand(postCommitCmd)
}
