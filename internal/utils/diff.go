// internal/utils/diff.go

package utils

import (
	"fmt"
	"strings"
)

type DiffMode int

const (
	DiffSinceLastCommit DiffMode = iota
	DiffAllLocalChanges
)

func GetDiffs(mode DiffMode) (string, error) {
	switch mode {
	case DiffSinceLastCommit:
		return ExecuteGitCommand("diff", "HEAD~1", "HEAD")

	case DiffAllLocalChanges:
		staged, err := ExecuteGitCommand("diff", "--cached", "HEAD")
		if err != nil {
			return "", fmt.Errorf("error getting staged diff: %w", err)
		}

		unstaged, err := ExecuteGitCommand("diff", "HEAD")
		if err != nil {
			return "", fmt.Errorf("error getting unstaged diff: %w", err)
		}

		untracked, err := ExecuteGitCommand("ls-files", "--others", "--exclude-standard")
		if err != nil {
			return "", fmt.Errorf("error getting untracked files: %w", err)
		}

		var builder strings.Builder
		if staged != "" {
			builder.WriteString(fmt.Sprintf("--- Staged Changes ---\n%s\n\n", staged))
		}
		if unstaged != "" {
			builder.WriteString(fmt.Sprintf("--- Unstaged Changes ---\n%s\n\n", unstaged))
		}
		if untracked != "" {
			builder.WriteString(fmt.Sprintf("--- Untracked Files ---\n%s\n\n", untracked))
		}

		return builder.String(), nil

	default:
		return "", fmt.Errorf("unknown diff mode: %d", mode)
	}
}
