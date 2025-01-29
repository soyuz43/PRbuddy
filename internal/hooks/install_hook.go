package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// Define the hook content
	prBuddyHookContent := `echo "` + colorutils.Cyan("[PRBuddy-Go] Commit detected. Generating pull request...") + `"

# Run the PR generation command
prbuddy-go post-commit --non-interactive

if [ $? -eq 0 ]; then
  echo "` + colorutils.Green("[PRBuddy-Go] Pull request generated successfully.") + `"
else
  echo "` + colorutils.Red("[PRBuddy-Go] Failed to generate pull request.") + `"
fi`

	postCommitPath := filepath.Join(hooksDir, "post-commit")

	// Check if the post-commit hook already exists
	if _, err := os.Stat(postCommitPath); err == nil {
		// Read the existing hook content
		existingContent, err := os.ReadFile(postCommitPath)
		if err != nil {
			return fmt.Errorf("failed to read existing post-commit hook: %w", err)
		}

		// Check if the PRBuddy hook content is already present
		if strings.Contains(string(existingContent), "prbuddy-go post-commit") {
			fmt.Println(colorutils.Green("[PRBuddy-Go] post-commit hook already contains PRBuddy logic. Skipping reinstallation."))
			return nil
		}

		// Append PRBuddy hook content to the existing hook
		updatedContent := string(existingContent) + "\n\n# Added by PRBuddy-Go\n" + prBuddyHookContent
		err = os.WriteFile(postCommitPath, []byte(updatedContent), 0755)
		if err != nil {
			return fmt.Errorf("failed to append PRBuddy logic to existing post-commit hook: %w", err)
		}
		fmt.Println(colorutils.Green("[PRBuddy-Go] post-commit hook updated with PRBuddy logic."))
	} else {
		// If the hook doesn't exist, create a new one
		newHookContent := `
						#!/bin/bash
						# Added by PRBuddy-Go
						` + prBuddyHookContent

		err = os.WriteFile(postCommitPath, []byte(newHookContent), 0755)
		if err != nil {
			return fmt.Errorf("failed to write post-commit hook: %w", err)
		}
		fmt.Printf(colorutils.Cyan("[PRBuddy-Go] post-commit hook installed at %s\n"), postCommitPath)
	}

	return nil
}
