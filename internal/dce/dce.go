package dce

import (
	"fmt"
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

// BuildTaskList generates tasks based on user input by delegating to task_helper.
func (d *DefaultDCE) BuildTaskList(input string) ([]contextpkg.Task, []string, error) {
	return BuildTaskList(input)
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

// stringSliceContains returns true if the slice contains the value.
func stringSliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
