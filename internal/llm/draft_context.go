// internal/llm/draft_context.go

package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// SaveDraftContext saves conversation messages to disk for a specific branch/commit
func SaveDraftContext(branchName, commitHash string, context []contextpkg.Message) error {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	sanitizedBranch := utils.SanitizeBranchName(branchName)
	commitDir := filepath.Join(repoPath, ".git", "pr_buddy_db",
		fmt.Sprintf("%s-%s", sanitizedBranch, commitHash[:7]))

	if err := os.MkdirAll(commitDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	draftContextJSON, err := utils.MarshalJSON(context)
	if err != nil {
		return fmt.Errorf("failed to marshal draft context: %w", err)
	}

	if err := saveFile(commitDir, "draft_context.json", string(draftContextJSON)); err != nil {
		return fmt.Errorf("failed to save draft context: %w", err)
	}

	return nil
}

// LoadDraftContext retrieves saved conversation context for a specific branch/commit
func LoadDraftContext(branchName, commitHash string) ([]contextpkg.Message, error) {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository path: %w", err)
	}

	sanitizedBranch := utils.SanitizeBranchName(branchName)
	filePath := filepath.Join(repoPath, ".git", "pr_buddy_db",
		fmt.Sprintf("%s-%s", sanitizedBranch, commitHash[:7]), "draft_context.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read draft context file: %w", err)
	}

	var context []contextpkg.Message
	if err := json.Unmarshal(data, &context); err != nil {
		return nil, fmt.Errorf("failed to unmarshal draft context: %w", err)
	}

	return context, nil
}

// saveFile writes content to a specified file within a directory
func saveFile(dir, filename, content string) error {
	path := filepath.Join(dir, filename)
	if err := utils.WriteFile(path, []byte(content)); err != nil {
		return fmt.Errorf("file write failed: %w", err)
	}

	return nil
}
