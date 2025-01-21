// internal/llm/what_llm.go

package llm

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/utils"
)

func GenerateWhatSummary() (string, error) {
	// Preserve original what command logic
	commitCount, err := utils.ExecuteGitCommand("rev-list", "--count", "HEAD")
	if err != nil {
		return "", fmt.Errorf("error checking commits: %w", err)
	}
	if commitCount == "0" {
		return "", fmt.Errorf("no commits found in the repository")
	}

	unstagedChanges, err := utils.ExecuteGitCommand("diff", "HEAD")
	if err != nil {
		return "", fmt.Errorf("error getting unstaged diff: %w", err)
	}

	stagedChanges, err := utils.ExecuteGitCommand("diff", "--cached", "HEAD")
	if err != nil {
		return "", fmt.Errorf("error getting staged diff: %w", err)
	}

	untrackedFiles, err := utils.ExecuteGitCommand("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return "", fmt.Errorf("error getting untracked files: %w", err)
	}

	fullDiff := ""
	if stagedChanges != "" {
		fullDiff += fmt.Sprintf("--- Staged Changes ---\n%s\n\n", stagedChanges)
	}
	if unstagedChanges != "" {
		fullDiff += fmt.Sprintf("--- Unstaged Changes ---\n%s\n\n", unstagedChanges)
	}
	if untrackedFiles != "" {
		fullDiff += fmt.Sprintf("--- Untracked Files ---\n%s\n\n", untrackedFiles)
	}

	if fullDiff == "" {
		return "No changes detected since the last commit.", nil
	}

	prompt := fmt.Sprintf(`
These are the git diffs for the repository, split into staged, unstaged, and untracked files. Each category may or may not contain changes:

# Staged Changes:

%s

# Unstaged Changes:

%s

# Untracked Files:

%s

---
!TASK::
1. Provide a meticulous natural language summary of each of the changes. Do so by file. Describe each change made in full.
2. List and separate changes for each file changed using numbered points, and using markdown standards in formatting.
3. Only describe the changes explicitly present in the diffs. Do not infer, speculate, or invent additional content.
4. Focus on helping the developer reorient themselves and where they left off.
`, stagedChanges, unstagedChanges, untrackedFiles)

	return GenerateSummary(prompt)
}
