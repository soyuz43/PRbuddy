// internal/utils/context_logger.go

package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
)

// SaveContextToFile saves the concatenated context messages to a JSON file.
// It creates the necessary directories if they do not exist.
// The file is named with a timestamp for easy identification.
func SaveContextToFile(conversationID string, messages []contextpkg.Message) error {
	// Retrieve the repository path
	repoPath, err := GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	// Define the log directory path
	logDir := filepath.Join(repoPath, ".git", "pr_buddy_db", "context_logs")

	// Create the log directory if it doesn't exist
	err = os.MkdirAll(logDir, 0750)
	if err != nil {
		return fmt.Errorf("failed to create context_logs directory: %w", err)
	}

	// Generate a timestamped filename
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("conversation-%s-%s.json", conversationID, timestamp)
	filePath := filepath.Join(logDir, filename)

	// Marshal the messages to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context messages to JSON: %w", err)
	}

	// Write the JSON data to the file
	err = os.WriteFile(filePath, jsonData, 0640)
	if err != nil {
		return fmt.Errorf("failed to write context to file: %w", err)
	}

	fmt.Printf("[Context Logger] Structured context successfully saved to %s\n", filePath)
	return nil
}

// SaveConcatenatedContextToFile concatenates all messages and saves them to a text file.
// This provides a readable format of what the LLM receives.
func SaveConcatenatedContextToFile(conversationID string, messages []contextpkg.Message) error {
	// Retrieve the repository path
	repoPath, err := GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	// Define the log directory path
	logDir := filepath.Join(repoPath, ".git", "pr_buddy_db", "context_logs")

	// Create the log directory if it doesn't exist
	err = os.MkdirAll(logDir, 0750)
	if err != nil {
		return fmt.Errorf("failed to create context_logs directory: %w", err)
	}

	// Generate a timestamped filename
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("conversation-%s-%s.txt", conversationID, timestamp)
	filePath := filepath.Join(logDir, filename)

	// Concatenate all messages into a single string
	var builder strings.Builder
	for _, msg := range messages {
		// Optionally, format each message with role labels
		builder.WriteString(fmt.Sprintf("%s: %s\n", strings.Title(msg.Role), msg.Content))
	}
	concatenatedContext := builder.String()

	// Write the concatenated context to the file
	err = os.WriteFile(filePath, []byte(concatenatedContext), 0640)
	if err != nil {
		return fmt.Errorf("failed to write concatenated context to file: %w", err)
	}

	fmt.Printf("[Context Logger] Concatenated context successfully saved to %s\n", filePath)
	return nil
}
