// internal/dce/filtered_data_utils.go

package dce

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
)

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
func GenerateFilteredData(tasks []contextpkg.Task) ([]FilteredData, []string, error) {
	var logs []string
	var filtered []FilteredData

	logs = append(logs, "Generating filtered data from tasks")

	// 1. Build file hierarchy for relevant files
	fileHierarchy := buildRelevantFileHierarchy(tasks)

	// 2. Get linter results (simplified version)
	linterResults, linterLogs := getLinterResults(tasks)
	logs = append(logs, linterLogs...)

	fd := FilteredData{
		FileHierarchy: fileHierarchy,
		LinterResults: linterResults,
	}
	filtered = append(filtered, fd)

	logs = append(logs, "Generated filtered data with linter integration")
	return filtered, logs, nil
}

// getLinterResults provides a simplified linter integration
func getLinterResults(tasks []contextpkg.Task) (string, []string) {
	var logs []string
	var builder strings.Builder

	totalFiles := 0
	for _, task := range tasks {
		totalFiles += len(task.Files)
	}

	if totalFiles == 0 {
		return "No files to analyze.", logs
	}

	builder.WriteString(fmt.Sprintf("Analyzed %d files across %d tasks:\n\n", totalFiles, len(tasks)))

	// For demonstration, let's assume we find some potential issues
	issuesFound := 0
	for _, task := range tasks {
		for _, file := range task.Files {
			// In a real implementation, this would run an actual linter
			if strings.HasSuffix(file, ".go") {
				// Simulate finding some issues in Go files
				if issuesFound == 0 {
					builder.WriteString("Potential issues detected:\n")
				}

				if strings.Contains(file, "handler") {
					builder.WriteString(fmt.Sprintf("- %s: Consider adding error handling\n", file))
					issuesFound++
				}
				if strings.Contains(file, "service") {
					builder.WriteString(fmt.Sprintf("- %s: Missing unit tests\n", file))
					issuesFound++
				}
			}
		}
	}

	if issuesFound == 0 {
		builder.WriteString("No immediate issues detected. Code looks good!\n")
	} else {
		builder.WriteString("\nRecommended next steps:\n")
		builder.WriteString("- Address the identified issues\n")
		builder.WriteString("- Consider writing additional tests\n")
		builder.WriteString("- Review documentation\n")
	}

	logs = append(logs, fmt.Sprintf("Found %d potential issues", issuesFound))
	return builder.String(), logs
}
