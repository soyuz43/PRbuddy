// cmd/post_commit.go

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	Run:   runPostCommit,
}

func init() {
	rootCmd.AddCommand(postCommitCmd)
}

func runPostCommit(cmd *cobra.Command, args []string) {
	fmt.Println("[PRBuddy-Go] Running post-commit logic...")

	// Get extension status
	extensionInstalled, err := utils.CheckExtensionInstalled()
	if err != nil {
		fmt.Printf("[PRBuddy-Go] Extension check error: %v\n", err)
	}

	// Generate draft PR
	branchName, commitHash, draftPR, err := generateDraftPR()
	if err != nil {
		fmt.Printf("[PRBuddy-Go] Error generating draft: %v\n", err)
		return
	}

	// Handle communication based on extension presence
	if extensionInstalled {
		if err := communicateWithExtension(branchName, commitHash, draftPR); err != nil {
			handleExtensionFailure(draftPR, err)
		}
	} else {
		fmt.Println("\n**Draft PR Generated:**")
		fmt.Println(draftPR)
	}

	// Save logs regardless of output method
	if err := saveConversationLogs(branchName, commitHash, "Generated draft PR successfully."); err != nil {
		fmt.Printf("[PRBuddy-Go] Error saving logs: %v\n", err)
	}

	fmt.Println("\n[PRBuddy-Go] Post-commit processing complete.")
}

func generateDraftPR() (string, string, string, error) {
	branchName, err := utils.ExecuteGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", "", "", fmt.Errorf("error retrieving branch name: %w", err)
	}

	commitHash, err := utils.ExecuteGitCommand("rev-parse", "HEAD")
	if err != nil {
		return "", "", "", fmt.Errorf("error retrieving commit hash: %w", err)
	}

	commitMessage, diffs, err := llm.GeneratePreDraftPR()
	if err != nil {
		return "", "", "", fmt.Errorf("error generating pre-draft: %w", err)
	}

	if diffs == "" {
		return "", "", "", fmt.Errorf("no changes detected")
	}

	draftPR, err := llm.GenerateDraftPR(commitMessage, diffs)
	if err != nil {
		return "", "", "", fmt.Errorf("error generating draft: %w", err)
	}

	return branchName, commitHash, draftPR, nil
}

func communicateWithExtension(branch, hash, draft string) error {
	// Try to activate extension
	cmd := exec.Command("code", "--activate-extension", "your.extension-id")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to activate extension: %w", err)
	}

	// Get server port
	port, err := utils.ReadPortFile()
	if err != nil {
		return fmt.Errorf("failed to get server port: %w", err)
	}

	// Send data to extension
	client := http.Client{Timeout: 5 * time.Second}
	payload := map[string]interface{}{
		"branch":    branch,
		"commit":    hash,
		"draft_pr":  draft,
		"timestamp": time.Now().Unix(),
	}

	resp, err := client.Post(
		fmt.Sprintf("http://localhost:%d/extension", port),
		"application/json",
		strings.NewReader(toJSON(payload)),
	)

	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("extension communication failed")
	}

	return nil
}

func handleExtensionFailure(draft string, err error) {
	fmt.Printf("\n[PRBuddy-Go] Instaflow Extension not responding (%v), defaulting to terminal output.\n", err)
	fmt.Println("\n**Draft PR Generated:**")
	fmt.Println(draft)
}

func toJSON(data interface{}) string {
	b, _ := json.Marshal(data)
	return string(b)
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
