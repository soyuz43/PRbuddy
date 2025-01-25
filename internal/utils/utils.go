// internal/utils/utils.go

package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ExecuteGitCommand runs a git command and returns its output
func ExecuteGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrapf(err, "git command failed: git %s", strings.Join(args, " "))
	}
	return strings.TrimSpace(out.String()), nil
}

// GetRepoPath retrieves the top-level directory of the current Git repository
func GetRepoPath() (string, error) {
	repoPath, err := ExecuteGitCommand("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return repoPath, nil
}

// SanitizeBranchName replaces slashes and spaces in branch names for safe usage
func SanitizeBranchName(branch string) string {
	return strings.ReplaceAll(strings.ReplaceAll(branch, "/", "_"), " ", "-")
}

// CheckExtensionInstalled verifies if the extension is installed
func CheckExtensionInstalled() (bool, error) {
	repoPath, err := GetRepoPath()
	if err != nil {
		return false, fmt.Errorf("failed to get repository path: %w", err)
	}

	indicatorPath := filepath.Join(repoPath, ".git", "prbuddy", ".extension-installed")
	if _, err := os.Stat(indicatorPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("error checking extension status: %w", err)
	}

	return true, nil
}

// CreateExtensionIndicator creates the extension installation marker
func CreateExtensionIndicator() error {
	repoPath, err := GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	prbuddyPath := filepath.Join(repoPath, ".git", "prbuddy")
	if err := os.MkdirAll(prbuddyPath, 0750); err != nil {
		return fmt.Errorf("failed to create prbuddy directory: %w", err)
	}

	indicatorPath := filepath.Join(prbuddyPath, ".extension-installed")
	if err := os.WriteFile(indicatorPath, []byte(""), 0640); err != nil {
		return fmt.Errorf("failed to create extension indicator: %w", err)
	}

	return nil
}

// RemoveExtensionIndicator removes the extension installation marker
func RemoveExtensionIndicator() error {
	repoPath, err := GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	indicatorPath := filepath.Join(repoPath, ".git", "prbuddy", ".extension-installed")
	if err := os.Remove(indicatorPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove extension indicator: %w", err)
	}

	return nil
}
