// internal/llm/draft_context.go

package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// SaveDraftContext saves conversation messages to disk for a specific branch/commit
func SaveDraftContext(branchName, commitHash string, context []Message) error {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	sanitizedBranch := sanitizeBranchName(branchName)
	commitDir := filepath.Join(repoPath, ".git", "pr_buddy_db",
		fmt.Sprintf("%s-%s", sanitizedBranch, commitHash[:7]))

	if err := os.MkdirAll(commitDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	draftContextJSON, err := utils.MarshalJSON(context)
	if err != nil {
		return err
	}

	if err := saveFile(commitDir, "draft_context.json", draftContextJSON); err != nil {
		return err
	}

	return nil
}

func saveFile(dir, filename, content string) error {
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("file write failed: %w", err)
	}

	return nil
}

// LoadDraftContext retrieves saved conversation context for a specific branch/commit
func LoadDraftContext(branchName, commitHash string) ([]Message, error) {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository path: %w", err)
	}

	sanitizedBranch := sanitizeBranchName(branchName)
	filePath := filepath.Join(repoPath, ".git", "pr_buddy_db",
		fmt.Sprintf("%s-%s", sanitizedBranch, commitHash[:7]), "draft_context.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var context []Message
	if err := json.Unmarshal(data, &context); err != nil {
		return context, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	return context, nil
}

func sanitizeBranchName(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}
