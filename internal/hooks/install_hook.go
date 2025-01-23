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

	// Install pre-commit hook
	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	preCommitHookContent := `#!/bin/bash
echo "` + colorutils.Cyan("[PRBuddy-Go] Detected commit. Generate PR?") + `"

# Prompt the user
read -p "` + colorutils.Cyan("[PRBuddy-Go] Do you want to generate a PR for this commit? ([y]/n) ") + `" yn
case $yn in
  [Yy]*|"" )
    echo "1" > .git/prbuddy_run
    ;;
  [Nn]* )
    echo "0" > .git/prbuddy_run
    ;;
  * )
    echo "` + colorutils.Red("[PRBuddy-Go] Invalid input. Defaulting to 'no'.") + `"
    echo "0" > .git/prbuddy_run
    ;;
esac

exit 0
`

	err = os.WriteFile(preCommitPath, []byte(preCommitHookContent), 0755)
	if err != nil {
		return fmt.Errorf("failed to write pre-commit hook: %w", err)
	}
	fmt.Printf(colorutils.Cyan("[PRBuddy-Go] pre-commit hook installed at %s\n"), preCommitPath)

	// Install post-commit hook
	postCommitPath := filepath.Join(hooksDir, "post-commit")
	postCommitHookContent := `#!/bin/bash
echo "` + colorutils.Cyan("[PRBuddy-Go] Detected commit. Running post-commit hook...") + `"

# Check if the user opted to generate a PR
if [ -f .git/prbuddy_run ]; then
  RUN_PR_BUDDY=$(cat .git/prbuddy_run)
  rm -f .git/prbuddy_run

  if [ "$RUN_PR_BUDDY" = "1" ]; then
    echo "` + colorutils.Green("[PRBuddy-Go] Generating PR as requested...") + `"
    prbuddy-go post-commit
  else
    echo "` + colorutils.Yellow("[PRBuddy-Go] Skipping PR generation as requested.") + `"
  fi
else
  echo "` + colorutils.Yellow("[PRBuddy-Go] No PR generation preference found. Skipping.") + `"
fi
`

	err = os.WriteFile(postCommitPath, []byte(postCommitHookContent), 0755)
	if err != nil {
		return fmt.Errorf("failed to write post-commit hook: %w", err)
	}
	fmt.Printf(colorutils.Cyan("[PRBuddy-Go] post-commit hook installed at %s\n"), postCommitPath)

	return nil
}
