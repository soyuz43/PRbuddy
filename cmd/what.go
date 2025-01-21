// cmd/what.go

package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/database"
	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/spf13/cobra"
)

// whatCmd represents the what command
var whatCmd = &cobra.Command{
	Use:   "what",
	Short: "Summarize recent changes since the last commit.",
	Long:  `Analyzes staged, unstaged, and untracked changes in the repository and provides a natural language summary using the LLM.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[prbuddy-go] Running 'what' command...")

		// 1. Check if there are any commits
		hasCommits, err := database.HasCommits()
		if err != nil {
			fmt.Printf("[prbuddy-go] Error checking commits: %v\n", err)
			return
		}
		if !hasCommits {
			fmt.Println("[prbuddy-go] No commits found in the repository. Please make a commit first.")
			return
		}

		// 2. Get unstaged changes
		unstagedChanges, err := database.GetGitDiff("HEAD")
		if err != nil {
			fmt.Printf("[prbuddy-go] Error getting unstaged git diff: %v\n", err)
			return
		}

		// 3. Get staged changes
		stagedChanges, err := database.GetGitDiff("--cached", "HEAD")
		if err != nil {
			fmt.Printf("[prbuddy-go] Error getting staged git diff: %v\n", err)
			return
		}

		// 4. Get untracked files
		untrackedFiles, err := database.GetUntrackedFiles()
		if err != nil {
			fmt.Printf("[prbuddy-go] Error getting untracked files: %v\n", err)
			return
		}

		// 5. Prepare the full diff
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

		if fullDiff == "" {
			fmt.Println("[prbuddy-go] No changes detected since the last commit.")
			return
		}

		// 6. Prepare the prompt for the LLM
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
