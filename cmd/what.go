// cmd/what.go

package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

// whatCmd represents the what command
var whatCmd = &cobra.Command{
	Use:   "what",
	Short: "Summarize recent changes since the last commit.",
	Long:  `Analyzes staged, unstaged, and untracked changes in the repository and provides a natural language summary using the LLM.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[prbuddy-go] Running 'what' command...")

		// 1. Check if there are any commits in the repository
		commitCount, err := utils.ExecuteGitCommand("rev-list", "--count", "HEAD")
		if err != nil {
			fmt.Printf("[prbuddy-go] Error checking commits: %v\n", err)
			return
		}
		if commitCount == "0" {
			fmt.Println("[prbuddy-go] No commits found in the repository. Please make a commit first.")
			return
		}

		// 2. Retrieve unstaged changes
		unstagedChanges, err := utils.ExecuteGitCommand("diff", "HEAD")
		if err != nil {
			fmt.Printf("[prbuddy-go] Error getting unstaged git diff: %v\n", err)
			return
		}

		// 3. Retrieve staged changes
		stagedChanges, err := utils.ExecuteGitCommand("diff", "--cached", "HEAD")
		if err != nil {
			fmt.Printf("[prbuddy-go] Error getting staged git diff: %v\n", err)
			return
		}

		// 4. Retrieve untracked files
		untrackedFiles, err := utils.ExecuteGitCommand("ls-files", "--others", "--exclude-standard")
		if err != nil {
			fmt.Printf("[prbuddy-go] Error getting untracked files: %v\n", err)
			return
		}

		// 5. Combine diffs for the prompt
		fullDiff := ""
		if stagedChanges != "" {
			fullDiff += fmt.Sprintf("--- Staged Changes ---\n%s\n\n", stagedChanges)
		}
		if unstagedChanges != "" {
			fullDiff += fmt.Sprintf("--- Unstaged Changes ---\n%s\n\n", unstagedChanges)
		}
		if untrackedFiles != "" {
			fullDiff += fmt.Sprintf("--- Untracked Files ---\n%s\n\n", untrackedFiles)
		}

		// If there are no changes, exit
		if fullDiff == "" {
			fmt.Println("[prbuddy-go] No changes detected since the last commit.")
			return
		}

		// 6. Prepare the LLM prompt
		prompt := fmt.Sprintf(`
These are the git diffs for the repository, split into staged, unstaged, and untracked files. Each category may or may not contain changes:

# Staged Changes:

%s

# Unstaged Changes:

%s

# Untracked Files:

%s

---
!TASK::
1. Provide a meticulous natural language summary of each of the changes. Do so by file. Describe each change made in full.
2. List and separate changes for each file changed using numbered points, and using markdown standards in formatting.
3. Only describe the changes explicitly present in the diffs. Do not infer, speculate, or invent additional content.
4. Focus on helping the developer reorient themselves and where they left off.
`, stagedChanges, unstagedChanges, untrackedFiles)

		// 7. Call the LLM to summarize the changes
		summary, err := llm.GenerateSummary(prompt)
		if err != nil {
			fmt.Printf("[prbuddy-go] Error generating summary: %v\n", err)
			return
		}

		// 8. Display the summary
		fmt.Println("\n**What Have I Done Since the Last Commit:**")
		fmt.Println(summary)
	},
}

func init() {
	rootCmd.AddCommand(whatCmd)
}
