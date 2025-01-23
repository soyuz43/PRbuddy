// internal/hooks/remove_hook.go

package hooks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// RemovePostCommitHook removes the post-commit Git hook
func RemovePostCommitHook() error {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return err
	}

	postCommitPath := filepath.Join(repoPath, ".git", "hooks", "post-commit")

	if _, err := os.Stat(postCommitPath); os.IsNotExist(err) {
		fmt.Printf("[PRBuddy-Go] No post-commit hook found at %s\n", postCommitPath)
		return nil
	}

	err = os.Remove(postCommitPath)
	if err != nil {
		return fmt.Errorf("failed to remove post-commit hook: %w", err)
	}

	fmt.Printf("[PRBuddy-Go] post-commit hook removed from %s\n", postCommitPath)
	return nil
}
