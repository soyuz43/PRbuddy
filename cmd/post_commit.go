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

type ConversationLog struct {
	BranchName string    `json:"branch_name"`
	CommitHash string    `json:"commit_hash"`
	Messages   []Message `json:"messages"`
}

type Message struct {
	From    string `json:"from"`
	Content string `json:"content"`
}

var postCommitCmd = &cobra.Command{
	Use:   "post-commit",
	Short: "Handle the post-commit hook.",
	Long:  `Generates a draft pull request based on the latest commit and logs the conversation.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[PRBuddy-Go] Running post-commit logic...")

		branchName, err := utils.ExecuteGitCommand("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error retrieving branch name: %v\n", err)
			return
		}

		commitHash, err := utils.ExecuteGitCommand("rev-parse", "HEAD")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error retrieving commit hash: %v\n", err)
			return
		}

		commitMessage, diffs, err := llm.GeneratePreDraftPR()
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error generating pre-draft PR: %v\n", err)
			return
		}

		if diffs == "" {
			fmt.Println("[PRBuddy-Go] No changes detected. No pull request draft generated.")
			return
		}

		draftPR, err := llm.GenerateDraftPR(commitMessage, diffs)
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error generating draft PR: %v\n", err)
			return
		}

		fmt.Println("\n**Draft PR Generated:**")
		fmt.Println(draftPR)

		err = saveConversationLogs(branchName, commitHash, "Generated draft PR successfully.")
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error saving conversation logs: %v\n", err)
		}

		fmt.Println("\n[PRBuddy-Go] Post-commit processing complete.")
	},
}

func init() {
	rootCmd.AddCommand(postCommitCmd)
}

func saveConversationLogs(branch, hash, message string) error {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	baseDir := filepath.Join(repoPath, ".git", "pr_buddy_db")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	sanitizedBranch := sanitizeBranchName(branch)
	commitDirName := fmt.Sprintf("%s-%s", sanitizedBranch, hash[:7])
	commitDir := filepath.Join(baseDir, commitDirName)

	if err := os.MkdirAll(commitDir, 0755); err != nil {
		return fmt.Errorf("failed to create commit directory: %w", err)
	}

	conversationFile := filepath.Join(commitDir, "conversation.json")
	draftFile := filepath.Join(commitDir, "draft_context.json")

	// Save conversation log
	conversation := ConversationLog{
		BranchName: branch,
		CommitHash: hash,
		Messages: []Message{
			{From: "User", Content: "Initiated draft PR creation."},
			{From: "PRBuddy-Go", Content: message},
		},
	}

	conversationData, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation log: %w", err)
	}

	// Save draft context
	initialContext := []llm.Message{
		{Role: "user", Content: "Initiated draft PR creation"},
		{Role: "assistant", Content: message},
	}
	draftData, err := json.MarshalIndent(initialContext, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal draft context: %w", err)
	}

	// Write files
	if err := os.WriteFile(conversationFile, conversationData, 0644); err != nil {
		return fmt.Errorf("failed to write conversation log: %w", err)
	}

	if err := os.WriteFile(draftFile, draftData, 0644); err != nil {
		return fmt.Errorf("failed to write draft context: %w", err)
	}

	fmt.Printf("[PRBuddy-Go] Saved logs at %s\n", commitDir)
	return nil
}

func sanitizeBranchName(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}
