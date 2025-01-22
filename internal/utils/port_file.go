// internal/utils/port_file.go

package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// WritePortFile writes the port number to .git/.prbuddy_port
func WritePortFile(port int) error {
	repoPath, err := GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	portFilePath := filepath.Join(repoPath, ".git", ".prbuddy_port")
	file, err := os.Create(portFilePath)
	if err != nil {
		return fmt.Errorf("failed to create port file: %w", err)
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%d", port)
	if err != nil {
		return fmt.Errorf("failed to write port to file: %w", err)
	}

	return nil
}

// ReadPortFile reads the port number from .git/.prbuddy_port
func ReadPortFile() (int, error) {
	repoPath, err := GetRepoPath()
	if err != nil {
		return 0, fmt.Errorf("failed to get repository path: %w", err)
	}

	portFilePath := filepath.Join(repoPath, ".git", ".prbuddy_port")
	data, err := os.ReadFile(portFilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read port file: %w", err)
	}

	var port int
	_, err = fmt.Sscanf(string(data), "%d", &port)
	if err != nil {
		return 0, fmt.Errorf("failed to parse port number: %w", err)
	}

	return port, nil
}

// DeletePortFile removes the .git/.prbuddy_port file
func DeletePortFile() error {
	repoPath, err := GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	portFilePath := filepath.Join(repoPath, ".git", ".prbuddy_port")
	if err := os.Remove(portFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete port file: %w", err)
	}

	return nil
}
