package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckExtensionInstalled verifies if the extension installation marker exists.
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

// CreateExtensionIndicator creates the extension installation marker.
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

// RemoveExtensionIndicator removes the extension installation marker.
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
