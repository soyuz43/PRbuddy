package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
)

// LogLittleGuyContext writes the given data to a file named "littleguy-<conversationID>.txt"
// in a dedicated "logs" directory. A timestamp is prepended to each log entry.
func LogLittleGuyContext(conversationID, data string) error {
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	filename := filepath.Join(logsDir, fmt.Sprintf("littleguy-%s.txt", conversationID))
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] %s\n", timestamp, data)

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}
	return nil
}

// SaveContextToFile marshals a slice of context messages (from contextpkg.Message) to JSON and writes
// the result to a timestamped file in the repository's .git/pr_buddy_db/context_logs directory.
func SaveContextToFile(conversationID string, messages []contextpkg.Message) error {
	repoPath, err := GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	logDir := filepath.Join(repoPath, ".git", "pr_buddy_db", "context_logs")
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return fmt.Errorf("failed to create context_logs directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("conversation-%s-%s.json", conversationID, timestamp)
	filePath := filepath.Join(logDir, filename)

	jsonData, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context messages to JSON: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0640); err != nil {
		return fmt.Errorf("failed to write context to file: %w", err)
	}

	fmt.Printf("[Context Logger] Structured context successfully saved to %s\n", filePath)
	return nil
}

// SaveConcatenatedContextToFile concatenates a slice of context messages into a readable string and
// writes it to a timestamped text file in the repository's .git/pr_buddy_db/context_logs directory.
func SaveConcatenatedContextToFile(conversationID string, messages []contextpkg.Message) error {
	repoPath, err := GetRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get repository path: %w", err)
	}

	logDir := filepath.Join(repoPath, ".git", "pr_buddy_db", "context_logs")
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return fmt.Errorf("failed to create context_logs directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("conversation-%s-%s.txt", conversationID, timestamp)
	filePath := filepath.Join(logDir, filename)

	capitalizer := cases.Title(language.English)
	var builder strings.Builder
	for _, msg := range messages {
		builder.WriteString(fmt.Sprintf("%s: %s\n", capitalizer.String(msg.Role), msg.Content))
	}
	concatenatedContext := builder.String()

	if err := os.WriteFile(filePath, []byte(concatenatedContext), 0640); err != nil {
		return fmt.Errorf("failed to write concatenated context to file: %w", err)
	}

	fmt.Printf("[Context Logger] Concatenated context successfully saved to %s\n", filePath)
	return nil
}
