// internal/dce/filtered_data_utils.go

package dce

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
)

// GenerateFilteredData processes tasks to produce a summary of project data.
// This is a simplified implementation that uses standard Go libraries instead of
// non-existent utility functions.
func GenerateFilteredData(tasks []contextpkg.Task) ([]FilteredData, []string, error) {
	var logs []string
	var filtered []FilteredData

	logs = append(logs, "Generating filtered data from tasks")

	// 1. Build file hierarchy for relevant files
	fileHierarchy := buildRelevantFileHierarchy(tasks)

	// 2. Use a simplified approach for linter results
	// In a future implementation, we would integrate with an actual linter
	linterResults := buildSimplifiedLinterResults(tasks)

	fd := FilteredData{
		FileHierarchy: fileHierarchy,
		LinterResults: linterResults,
	}
	filtered = append(filtered, fd)

	logs = append(logs, "Generated filtered data with simplified results")
	return filtered, logs, nil
}

// buildRelevantFileHierarchy builds a file hierarchy for relevant files
// using standard filepath package instead of non-existent utils functions
func buildRelevantFileHierarchy(tasks []contextpkg.Task) string {
	var builder strings.Builder

	// Group files by directory
	dirs := make(map[string][]string)
	for _, task := range tasks {
		for _, file := range task.Files {
			// Use standard filepath functions instead of custom utils ones
			dir := filepath.Dir(file)
			filename := filepath.Base(file)
			dirs[dir] = append(dirs[dir], filename)
		}
	}

	// Format as a tree
	for dir, files := range dirs {
		builder.WriteString(fmt.Sprintf("%s/\n", dir))
		for _, file := range files {
			builder.WriteString(fmt.Sprintf("  ├── %s\n", file))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// buildSimplifiedLinterResults creates a simplified linter results string
// This is a placeholder until actual linter integration is implemented
func buildSimplifiedLinterResults(tasks []contextpkg.Task) string {
	var builder strings.Builder

	// Count total files and functions to provide some basic metrics
	totalFiles := 0
	totalFunctions := 0

	for _, task := range tasks {
		totalFiles += len(task.Files)
		totalFunctions += len(task.Functions)
	}

	if totalFiles == 0 {
		builder.WriteString("No relevant files found for analysis.")
		return builder.String()
	}

	builder.WriteString(fmt.Sprintf("Analyzed %d files with %d functions.\n", totalFiles, totalFunctions))
	builder.WriteString("Note: Full linter integration is not yet implemented.\n\n")

	// Add some basic information about the tasks
	builder.WriteString("Task Summary:\n")
	for i, task := range tasks {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, task.Description))

		if len(task.Files) > 0 {
			builder.WriteString(fmt.Sprintf("   • Files: %d\n", len(task.Files)))
		}
		if len(task.Functions) > 0 {
			builder.WriteString(fmt.Sprintf("   • Functions: %d\n", len(task.Functions)))
		}
		if len(task.Notes) > 0 {
			builder.WriteString("   • Notes: Available\n")
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
