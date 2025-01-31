// cmd/post_commit.go

package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

var (
	extensionActive   bool
	nonInteractive    bool
	extensionAttempts = 3
	extensionDelay    = 500 * time.Millisecond
)

// ConversationLog represents the structure for logging conversations
type ConversationLog struct {
	BranchName string               `json:"branch_name"`
	CommitHash string               `json:"commit_hash"`
	Messages   []contextpkg.Message `json:"messages"`
}

var postCommitCmd = &cobra.Command{
	Use:   "post-commit",
	Short: "Handle post-commit automation",
	Long:  `Generates PR drafts and coordinates with VS Code extension when available`,
	Run:   runPostCommit,
}

func init() {
	postCommitCmd.Flags().BoolVar(&extensionActive, "extension-active", false,
		"Indicates extension connectivity check")
	postCommitCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false,
		"Disable interactive prompts")
	rootCmd.AddCommand(postCommitCmd)
}

func runPostCommit(cmd *cobra.Command, args []string) {
	if !nonInteractive {
		fmt.Println("[PRBuddy-Go] Starting post-commit workflow...")
	}

	branchName, commitHash, draftPR, err := generateDraftPR()
	if err != nil {
		handleGenerationError(err)
		return
	}

	if extensionActive {
		if commErr := communicateWithExtension(branchName, commitHash, draftPR); commErr != nil {
			handleExtensionFailure(draftPR, commErr)
		}
	} else {
		presentTerminalOutput(draftPR)
	}

	if logErr := saveConversationLogs(branchName, commitHash, "Draft generated"); logErr != nil {
		fmt.Printf("[PRBuddy-Go] Logging error: %v\n", logErr)
	}

	if !nonInteractive {
		fmt.Println("[PRBuddy-Go] Post-commit workflow completed")
	}
}

func generateDraftPR() (string, string, string, error) {
	branchName, err := utils.ExecGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", "", "", fmt.Errorf("branch detection failed: %w", err)
	}

	commitHash, err := utils.ExecGit("rev-parse", "HEAD")
	if err != nil {
		return "", "", "", fmt.Errorf("commit hash retrieval failed: %w", err)
	}

	commitMessage, diffs, err := llm.GeneratePreDraftPR()
	if err != nil {
		return "", "", "", fmt.Errorf("pre-draft generation failed: %w", err)
	}

	if diffs == "" {
		return "", "", "", fmt.Errorf("no detectable changes")
	}

	draftPR, err := llm.GenerateDraftPR(commitMessage, diffs)
	if err != nil {
		return "", "", "", fmt.Errorf("draft generation failed: %w", err)
	}

	return strings.TrimSpace(branchName), strings.TrimSpace(commitHash), draftPR, nil
}

func communicateWithExtension(branch, hash, draft string) error {
	if err := activateExtension(); err != nil {
		return fmt.Errorf("extension activation: %w", err)
	}

	port, err := utils.ReadPortFile()
	if err != nil {
		return fmt.Errorf("port retrieval: %w", err)
	}

	return retryCommunication(port, branch, hash, draft)
}

func activateExtension() error {
	cmd := exec.Command("code", "--activate-extension", "prbuddy.extension")
	return cmd.Run()
}

func retryCommunication(port int, branch, hash, draft string) error {
	client := http.Client{Timeout: 2 * time.Second}
	payload := map[string]interface{}{
		"branch":    branch,
		"commit":    hash,
		"draft_pr":  draft,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	jsonPayload, err := utils.MarshalJSON(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	for i := 0; i < extensionAttempts; i++ {
		resp, err := client.Post(
			fmt.Sprintf("http://localhost:%d/extension", port),
			"application/json",
			strings.NewReader(string(jsonPayload)),
		)

		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}

		time.Sleep(extensionDelay)
	}

	return fmt.Errorf("failed after %d attempts", extensionAttempts)
}

func handleExtensionFailure(draft string, err error) {
	fmt.Printf("\n[PRBuddy-Go] Extension communication failed: %v\n", err)
	presentTerminalOutput(draft)
}

func presentTerminalOutput(draft string) {
	const line = "═══════════════════════════════════════════════════"
	fmt.Printf("\n%s\n🚀 Draft PR Generated\n%s\n%s\n%s\n\n",
		line, line, draft, line)
}

func saveConversationLogs(branch, hash, message string) error {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return fmt.Errorf("repo path detection: %w", err)
	}

	logDir := filepath.Join(repoPath, ".git", "pr_buddy_db",
		sanitizeBranchName(branch), fmt.Sprintf("commit-%s", hash[:7]))

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("log directory creation: %w", err)
	}

	conversation := ConversationLog{
		BranchName: branch,
		CommitHash: hash,
		Messages: []contextpkg.Message{
			{Role: "system", Content: "Initiated draft generation"},
			{Role: "assistant", Content: message},
		},
	}

	conversationJSON, err := utils.MarshalJSON(conversation)
	if err != nil {
		return err
	}

	if err := saveFile(logDir, "conversation.json", string(conversationJSON)); err != nil {
		return err
	}

	draftContext := []contextpkg.Message{
		{Role: "system", Content: "Initial draft context"},
		{Role: "assistant", Content: message},
	}

	draftContextJSON, err := utils.MarshalJSON(draftContext)
	if err != nil {
		return err
	}

	return saveFile(logDir, "draft_context.json", string(draftContextJSON))
}

func saveFile(dir, filename, content string) error {
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("file write failed: %w", err)
	}

	return nil
}

func sanitizeBranchName(branch string) string {
	return strings.ReplaceAll(strings.ReplaceAll(branch, "/", "_"), " ", "-")
}

func handleGenerationError(err error) {
	fmt.Printf("[PRBuddy-Go] Critical error: %v\n", err)
	fmt.Println("Failed to generate draft PR. Check git status and try again.")
}
