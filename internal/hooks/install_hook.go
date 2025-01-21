// internal/hooks/install_hook.go

package hooks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InstallPostCommitHook installs the post-commit Git hook
func InstallPostCommitHook() error {
	repoPath, err := getRepoPath()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	postCommitPath := filepath.Join(hooksDir, "post-commit")

	hookContent := `#!/bin/bash
echo "[prbuddy-go] Detected commit. Running post-commit hook..."
prbuddy post-commit
`

	// Ensure the hooks directory exists
	err = os.MkdirAll(hooksDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Write the post-commit hook
	err = os.WriteFile(postCommitPath, []byte(hookContent), 0755)
	if err != nil {
		return fmt.Errorf("failed to write post-commit hook: %w", err)
	}

	fmt.Printf("[prbuddy-go] post-commit hook installed at %s\n", postCommitPath)
	return nil
}

// getRepoPath retrieves the current repository path
func getRepoPath() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get repository path: %w", err)
	}
	repoPath := strings.TrimSpace(out.String())
	return repoPath, nil
}
