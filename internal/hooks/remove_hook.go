// internal/hooks/remove_hook.go

package hooks

import (
	"fmt"
	"os"
	"path/filepath"
)

// RemovePostCommitHook removes the post-commit Git hook
func RemovePostCommitHook() error {
	repoPath, err := getRepoPath()
	if err != nil {
		return err
	}

	postCommitPath := filepath.Join(repoPath, ".git", "hooks", "post-commit")

	if _, err := os.Stat(postCommitPath); os.IsNotExist(err) {
		fmt.Printf("[prbuddy-go] No post-commit hook found at %s\n", postCommitPath)
		return nil
	}

	err = os.Remove(postCommitPath)
	if err != nil {
		return fmt.Errorf("failed to remove post-commit hook: %w", err)
	}

	fmt.Printf("[prbuddy-go] post-commit hook removed from %s\n", postCommitPath)
	return nil
}
