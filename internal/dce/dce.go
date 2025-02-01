package dce

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// DCE defines the interface for dynamic context engine functions.
type DCE interface {
	Activate(task string) error
	Deactivate(conversationID string) error
	BuildTaskList(input string) ([]contextpkg.Task, []string, error)
	FilterProjectData(tasks []contextpkg.Task) ([]FilteredData, []string, error)
	AugmentContext(ctx []contextpkg.Message, filteredData []FilteredData) []contextpkg.Message
}

// FilteredData represents extra project data discovered by the DCE.
type FilteredData struct {
	FileHierarchy string
	LinterResults string
}

// DefaultDCE is the default implementation of the DCE interface.
type DefaultDCE struct{}

// NewDCE creates a new instance of DefaultDCE.
func NewDCE() DCE {
	return &DefaultDCE{}
}

// Activate initializes the DCE with the given task.
func (d *DefaultDCE) Activate(task string) error {
	fmt.Printf("[DCE] Activated. User task: %q\n", task)
	return nil
}

// Deactivate cleans up the DCE for the given conversation.
func (d *DefaultDCE) Deactivate(conversationID string) error {
	fmt.Printf("[DCE] Deactivated for conversation ID: %s\n", conversationID)
	return nil
}

// BuildTaskList generates tasks based on user input.
// Task-building logic is offloaded to contextpkg/task_utils when possible.
func (d *DefaultDCE) BuildTaskList(input string) ([]contextpkg.Task, []string, error) {
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
	matchedFiles := d.matchFilesByKeywords(trackedFiles, input)
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
		funcs := d.extractFunctionsFromFile(f, fileFuncPattern)
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

	// (Optional) Use task_utils to further process the task.
	// For example, you could reformat the task string:
	// formatted := task_utils.FormatTaskMessage(task)
	// logs = append(logs, "Formatted task: "+formatted)

	return []contextpkg.Task{task}, logs, nil
}

// FilterProjectData uses git diff to discover changed functions and updates tasks.
func (d *DefaultDCE) FilterProjectData(tasks []contextpkg.Task) ([]FilteredData, []string, error) {
	var logs []string
	logs = append(logs, "Filtering project data based on tasks")

	diffOutput, err := utils.ExecGit("diff", "--unified=0")
	if err != nil {
		return nil, logs, fmt.Errorf("failed to get git diff: %w", err)
	}
	logs = append(logs, "Retrieved git diff output")

	// Parse changed functions using the centralized helper.
	changedFuncs := ParseFunctionNames(diffOutput)
	logs = append(logs, fmt.Sprintf("Found %d changed functions: %v", len(changedFuncs), changedFuncs))

	// Update tasks with dependencies.
	for i := range tasks {
		for _, cf := range changedFuncs {
			if stringSliceContains(tasks[i].Functions, cf) {
				tasks[i].Dependencies = append(tasks[i].Dependencies, cf)
				tasks[i].Notes = append(tasks[i].Notes, fmt.Sprintf("Function %s changed in diff.", cf))
				logs = append(logs, fmt.Sprintf("Added dependency %q to task %q", cf, tasks[i].Description))
			}
		}
	}

	fd := []FilteredData{
		{
			FileHierarchy: "N/A (adjust as needed)",
			LinterResults: fmt.Sprintf("Detected %d changed functions: %v", len(changedFuncs), changedFuncs),
		},
	}
	logs = append(logs, "Created filtered data summary")
	return fd, logs, nil
}

// AugmentContext adds a system-level summary message to the conversation context.
func (d *DefaultDCE) AugmentContext(ctx []contextpkg.Message, filteredData []FilteredData) []contextpkg.Message {
	var builder strings.Builder
	builder.WriteString("**Dynamic Context Engine Summary**\n\n")
	for _, fd := range filteredData {
		builder.WriteString(fmt.Sprintf("- File Hierarchy: %s\n", fd.FileHierarchy))
		builder.WriteString(fmt.Sprintf("- Linter/Change Results: %s\n", fd.LinterResults))
	}
	augmented := append(ctx, contextpkg.Message{
		Role:    "system",
		Content: builder.String(),
	})
	return augmented
}

// matchFilesByKeywords returns files from allFiles that contain any keyword from userInput.
func (d *DefaultDCE) matchFilesByKeywords(allFiles []string, userInput string) []string {
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

// extractFunctionsFromFile reads file content and extracts function names using the provided regex.
func (d *DefaultDCE) extractFunctionsFromFile(filePath, pattern string) []string {
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

// stringSliceContains returns true if the slice contains the value.
func stringSliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
