package dce

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// BuildTaskList creates tasks based on user input, file matching, and function extraction.
func BuildTaskList(input string) ([]contextpkg.Task, []string, error) {
	var logs []string
	logs = append(logs, fmt.Sprintf("Building task list from input: %q", input))

	// 1. Retrieve all tracked files.
	out, err := utils.ExecGit("ls-files")
	if err != nil {
		return nil, logs, fmt.Errorf("failed to execute git ls-files: %w", err)
	}
	trackedFiles := strings.Split(strings.TrimSpace(out), "\n")
	logs = append(logs, fmt.Sprintf("Found %d tracked files", len(trackedFiles)))

	// 2. Match files based on keywords.
	matchedFiles := matchFilesByKeywords(trackedFiles, input)
	logs = append(logs, fmt.Sprintf("Matched %d files: %v", len(matchedFiles), matchedFiles))

	// 3. If no files matched, create a catch-all task.
	if len(matchedFiles) == 0 {
		task := contextpkg.Task{
			Description: input,
			Notes:       []string{"No direct file matches found. Add manually."},
		}
		logs = append(logs, "No file matches found - created catch-all task")
		return []contextpkg.Task{task}, logs, nil
	}

	// 4. Extract functions from each matched file.
	var allFunctions []string
	fileFuncPattern := `(?m)^\s*(def|func|function|public|private|static|void)\s+(\w+)\s*\(`
	for _, f := range matchedFiles {
		funcs := extractFunctionsFromFile(f, fileFuncPattern)
		if len(funcs) > 0 {
			logs = append(logs, fmt.Sprintf("Extracted %d functions from %s: %v", len(funcs), f, funcs))
			allFunctions = append(allFunctions, funcs...)
		} else {
			logs = append(logs, fmt.Sprintf("No functions found in %s", f))
		}
	}

	// 5. Create a consolidated task.
	task := contextpkg.Task{
		Description:  input,
		Files:        matchedFiles,
		Functions:    allFunctions,
		Dependencies: nil,
		Notes:        []string{"Matched via input and file heuristics."},
	}
	logs = append(logs, fmt.Sprintf("Created task with %d files and %d functions", len(matchedFiles), len(allFunctions)))

	return []contextpkg.Task{task}, logs, nil
}

// matchFilesByKeywords returns files from allFiles that contain any keyword from userInput.
func matchFilesByKeywords(allFiles []string, userInput string) []string {
	var matched []string
	words := strings.Fields(strings.ToLower(userInput))
	for _, file := range allFiles {
		lowerFile := strings.ToLower(file)
		for _, w := range words {
			if len(w) >= 3 && strings.Contains(lowerFile, w) {
				matched = append(matched, file)
				break
			}
		}
	}
	return matched
}

// extractFunctionsFromFile reads file content and extracts function names using the provided regex pattern.
func extractFunctionsFromFile(filePath, pattern string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	matches := re.FindAllStringSubmatch(string(data), -1)
	var funcs []string
	for _, m := range matches {
		if len(m) >= 3 {
			funcs = append(funcs, m[2])
		}
	}
	return funcs
}
