// internal/coreutils/logutils.go

package coreutils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogLittleGuyContext writes the given data to a file named
// littleguy-<conversationID>.txt in an optional logs directory
func LogLittleGuyContext(conversationID, data string) error {
	// (Optionally) use a dedicated logs directory
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	filename := filepath.Join(logsDir, fmt.Sprintf("littleguy-%s.txt", conversationID))

	// You can prepend timestamps or other contextual info if desired
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
