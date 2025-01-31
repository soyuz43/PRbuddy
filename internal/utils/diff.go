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

// GetDiffs returns diffs based on the given mode.
// It leverages the unified ExecGit (from gitutils.go) for all Git operations.
func GetDiffs(mode DiffMode) (string, error) {
	switch mode {
	case DiffSinceLastCommit:
		return ExecGit("diff", "HEAD~1", "HEAD")
	case DiffAllLocalChanges:
		staged, err := ExecGit("diff", "--cached", "HEAD")
		if err != nil {
			return "", fmt.Errorf("error getting staged diff: %w", err)
		}
		unstaged, err := ExecGit("diff", "HEAD")
		if err != nil {
			return "", fmt.Errorf("error getting unstaged diff: %w", err)
		}
		untracked, err := ExecGit("ls-files", "--others", "--exclude-standard")
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
