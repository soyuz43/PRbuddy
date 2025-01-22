// internal/utils/extension_utils.go

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	extensionIndicatorFile = ".git/prbuddy/.extension-installed"
)

// CheckExtensionInstalled checks if the VS Code extension is installed
func CheckExtensionInstalled() (bool, error) {
	repoPath, err := GetRepoPath()
	if err != nil {
		return false, fmt.Errorf("failed to get repository path: %w", err)
	}

	indicatorPath := filepath.Join(repoPath, extensionIndicatorFile)
	if _, err := os.Stat(indicatorPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check extension indicator file: %w", err)
	}

	return true, nil
}

// ActivateVSCodeExtension attempts to activate the VS Code extension
func ActivateVSCodeExtension() error {
	cmd := exec.Command("code", "--activate-extension", "prbuddy.extension")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to activate VS Code extension: %w", err)
	}
	return nil
}

// WaitForExtensionInitialization waits briefly for the extension to initialize
func WaitForExtensionInitialization() {
	time.Sleep(1 * time.Second)
}
