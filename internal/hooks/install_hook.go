// internal/hooks/install_hook.go

package hooks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/soyuz43/prbuddy-go/internal/utils"
)

func InstallPostCommitHook() error {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	postCommitPath := filepath.Join(hooksDir, "post-commit")

	hookContent := `#!/bin/bash
echo "[PRBuddy-Go] Detected commit. Running post-commit hook..."

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

	fmt.Printf("[PRBuddy-Go] post-commit hook installed at %s\n", postCommitPath)
	return nil
}
