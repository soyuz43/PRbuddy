// cmd/post_commit.go

package cmd

import (
	"bufio"
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

var (
	extensionActive   bool
	extensionAttempts = 3
	extensionDelay    = 500 * time.Millisecond
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
	Short: "Handle post-commit automation",
	Long:  `Generates PR drafts and coordinates with VS Code extension when available`,
	Run:   runPostCommit,
}

func init() {
	postCommitCmd.Flags().BoolVar(&extensionActive, "extension-active", false,
		"Indicates extension connectivity check")
	rootCmd.AddCommand(postCommitCmd)
}

func runPostCommit(cmd *cobra.Command, args []string) {
	fmt.Println("[PRBuddy-Go] Starting post-commit workflow...")

	// Add interactive confirmation prompt
	if !shouldGeneratePR() {
		fmt.Println("[PRBuddy-Go] Draft PR generation cancelled by user.")
		return
	}

	// Generate draft PR
	branchName, commitHash, draftPR, err := generateDraftPR()
	if err != nil {
		handleGenerationError(err)
		return
	}

	// Handle extension communication if active
	if extensionActive {
		if commErr := communicateWithExtension(branchName, commitHash, draftPR); commErr != nil {
			handleExtensionFailure(draftPR, commErr)
		}
	} else {
		presentTerminalOutput(draftPR)
	}

	// Save conversation logs
	if logErr := saveConversationLogs(branchName, commitHash, "Draft generated"); logErr != nil {
		fmt.Printf("[PRBuddy-Go] Logging error: %v\n", logErr)
	}

	fmt.Println("[PRBuddy-Go] Post-commit workflow completed")
}

// shouldGeneratePR prompts the user for confirmation and returns true if they want to proceed
func shouldGeneratePR() bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\n[PRBuddy-Go] Would you like to generate a draft pull request? ([y]/n) ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("[PRBuddy-Go] Error reading input: %v\n", err)
			return true // Default to generate on error
		}

		// Clean and normalize input
		cleanInput := strings.TrimSpace(strings.ToLower(input))

		switch cleanInput {
		case "", "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("[PRBuddy-Go] Please answer with 'y', 'n', or press Enter to accept default.")
			continue
		}
	}
}

func generateDraftPR() (string, string, string, error) {
	branchName, err := utils.ExecuteGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", "", "", fmt.Errorf("branch detection failed: %w", err)
	}

	commitHash, err := utils.ExecuteGitCommand("rev-parse", "HEAD")
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
			strings.NewReader(jsonPayload),
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
		Messages: []Message{
			{From: "System", Content: "Initiated draft generation"},
			{From: "PRBuddy-Go", Content: message},
		},
	}

	conversationJSON, err := utils.MarshalJSON(conversation)
	if err != nil {
		return err
	}

	if err := saveFile(logDir, "conversation.json", conversationJSON); err != nil {
		return err
	}

	draftContext := []llm.Message{
		{Role: "system", Content: "Initial draft context"},
		{Role: "assistant", Content: message},
	}

	draftContextJSON, err := utils.MarshalJSON(draftContext)
	if err != nil {
		return err
	}

	return saveFile(logDir, "draft_context.json", draftContextJSON)
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
