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

func InstallPostCommitHook() error {
	repoPath, err := getRepoPath()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	postCommitPath := filepath.Join(hooksDir, "post-commit")

	hookContent := `#!/bin/bash
echo "[prbuddy-go] Detected commit. Running post-commit hook..."

EXTENSION_DIR="$(git rev-parse --git-dir)/prbuddy"
PORT_FILE="$EXTENSION_DIR/.prbuddy_port"

# Check for extension installation
if [ -f "$EXTENSION_DIR/.extension-installed" ]; then
  # Start server in background if not running
  prbuddy-go serve --background 2>/dev/null &
  
  # Wait briefly for server initialization
  sleep 0.5
  
  # Check for port file existence
  if [ -f "$PORT_FILE" ]; then
    # Server is running - let backend handle extension communication
    prbuddy-go post-commit --extension-active
  else
    # Fallback to terminal output
    prbuddy-go post-commit
  fi
else
  # Direct terminal output without extension
  prbuddy-go post-commit
fi
`

	err = os.MkdirAll(hooksDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

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
