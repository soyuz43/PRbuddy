package hooks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/soyuz43/prbuddy-go/internal/utils/colorutils"
)

func InstallPostCommitHook() error {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(repoPath, ".git", "hooks")

	// Ensure the hooks directory exists
	if _, err := os.Stat(hooksDir); os.IsNotExist(err) {
		err = os.MkdirAll(hooksDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create hooks directory: %w", err)
		}
	}

	// Install post-commit hook
	postCommitPath := filepath.Join(hooksDir, "post-commit")
	postCommitHookContent := `#!/bin/bash
echo "` + colorutils.Cyan("[PRBuddy-Go] Commit detected. Generating pull request...") + `"

# Run the PR generation command
prbuddy-go post-commit --non-interactive

if [ $? -eq 0 ]; then
  echo "` + colorutils.Green("[PRBuddy-Go] Pull request generated successfully.") + `"
else
  echo "` + colorutils.Red("[PRBuddy-Go] Failed to generate pull request.") + `"
fi
`

	err = os.WriteFile(postCommitPath, []byte(postCommitHookContent), 0755)
	if err != nil {
		return fmt.Errorf("failed to write post-commit hook: %w", err)
	}
	fmt.Printf(colorutils.Cyan("[PRBuddy-Go] post-commit hook installed at %s\n"), postCommitPath)

	return nil
}
