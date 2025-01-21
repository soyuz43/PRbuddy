// cmd/post_commit.go

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

// ConversationLog represents the structure of the conversation to be saved
type ConversationLog struct {
	BranchName string    `json:"branch_name"`
	CommitHash string    `json:"commit_hash"`
	Messages   []Message `json:"messages"`
}

// Message represents a single interaction in the conversation
type Message struct {
	From    string `json:"from"`
	Content string `json:"content"`
}

// postCommitCmd represents the post-commit command
var postCommitCmd = &cobra.Command{
	Use:   "post-commit",
	Short: "Handle the post-commit hook.",
	Long:  `Generates a draft pull request based on the latest commit and logs the conversation.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[PRBuddy-Go] Running post-commit logic...")

		// 1. Retrieve current branch name
		branchName, err := utils.ExecuteGitCommand("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error retrieving branch name: %v\n", err)
			return
		}
		fmt.Printf("[PRBuddy-Go] Current Branch: %s\n", branchName)

		// 2. Retrieve latest commit hash
		commitHash, err := utils.ExecuteGitCommand("rev-parse", "HEAD")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error retrieving commit hash: %v\n", err)
			return
		}
		fmt.Printf("[PRBuddy-Go] Latest Commit Hash: %s\n", commitHash)

		// 3. Parse Git diffs (staged, unstaged, untracked)
		commitMessage, diffs, err := llm.GeneratePreDraftPR()
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error generating pre-draft PR: %v\n", err)
			return
		}

		if diffs == "" {
			fmt.Println("[PRBuddy-Go] No changes detected. No pull request draft generated.")
			return
		}

		// 4. Generate draft PR via LLM
		draftPR, err := llm.GenerateDraftPR(commitMessage, diffs)
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error generating draft PR: %v\n", err)
			return
		}

		// 5. Display the draft PR
		fmt.Println("\n**Draft PR Generated:**")
		fmt.Println(draftPR)

		// 6. Create directory and save conversation logs
		err = saveConversationLogs(branchName, commitHash, "Generated draft PR successfully.")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error saving conversation logs: %v\n", err)
			return
		}

		fmt.Println("\n[PRBuddy-Go] Post-commit processing complete.")
	},
}

func init() {
	rootCmd.AddCommand(postCommitCmd)
}

// saveConversationLogs creates the necessary directory and saves the conversation log
func saveConversationLogs(branch, hash, message string) error {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	// Define the base directory
	baseDir := filepath.Join(repoPath, ".git", "pr_buddy_db")
	err = os.MkdirAll(baseDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create base directory (%s): %w", baseDir, err)
	}

	// Sanitize branch name for filesystem
	sanitizedBranch := sanitizeBranchName(branch)

	// Define the commit-specific directory
	commitDirName := fmt.Sprintf("%s-%s", sanitizedBranch, hash[:7]) // Using first 7 chars of hash
	commitDir := filepath.Join(baseDir, commitDirName)

	err = os.MkdirAll(commitDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create commit directory (%s): %w", commitDir, err)
	}

	// Define the conversation.json file path
	conversationFile := filepath.Join(commitDir, "conversation.json")

	// Create the conversation log
	conversation := ConversationLog{
		BranchName: branch,
		CommitHash: hash,
		Messages: []Message{
			{
				From:    "User",
				Content: "Initiated draft PR creation.",
			},
			{
				From:    "PRBuddy-Go",
				Content: message,
			},
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation log to JSON: %w", err)
	}

	// Write to file
	err = os.WriteFile(conversationFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write conversation log to file (%s): %w", conversationFile, err)
	}

	fmt.Printf("[PRBuddy-Go] Conversation log saved at %s\n", conversationFile)
	return nil
}

// sanitizeBranchName removes or replaces characters that are problematic in directory names
func sanitizeBranchName(branch string) string {
	// Replace slashes with dashes and remove other unwanted characters
	sanitized := strings.ReplaceAll(branch, "/", "-")
	// Add more sanitization rules if necessary
	return sanitized
}
