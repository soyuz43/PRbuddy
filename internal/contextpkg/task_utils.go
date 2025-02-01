package contextpkg

import (
	"fmt"
	"strings"
)

// ParseTaskMessage converts a raw message content into a Task.
// This function is now the single source for task parsing logic.
func ParseTaskMessage(content string) (Task, error) {
	var task Task
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Task: ") {
			task.Description = strings.TrimPrefix(line, "Task: ")
		} else if strings.HasPrefix(line, "Files: ") {
			files := strings.TrimPrefix(line, "Files: ")
			task.Files = strings.Split(files, ", ")
		} else if strings.HasPrefix(line, "Functions: ") {
			funcs := strings.TrimPrefix(line, "Functions: ")
			task.Functions = strings.Split(funcs, ", ")
		} else if strings.HasPrefix(line, "Dependencies: ") {
			deps := strings.TrimPrefix(line, "Dependencies: ")
			task.Dependencies = strings.Split(deps, ", ")
		} else if strings.HasPrefix(line, "Notes: ") {
			notes := strings.TrimPrefix(line, "Notes: ")
			task.Notes = strings.Split(notes, "; ")
		}
	}
	if task.Description == "" {
		return task, fmt.Errorf("empty task description")
	}
	return task, nil
}

// FormatTaskMessage returns a formatted string representation of a Task.
func FormatTaskMessage(task Task) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Task: %s\n", task.Description))
	if len(task.Files) > 0 {
		builder.WriteString(fmt.Sprintf("Files: %s\n", strings.Join(task.Files, ", ")))
	}
	if len(task.Functions) > 0 {
		builder.WriteString(fmt.Sprintf("Functions: %s\n", strings.Join(task.Functions, ", ")))
	}
	if len(task.Dependencies) > 0 {
		builder.WriteString(fmt.Sprintf("Dependencies: %s\n", strings.Join(task.Dependencies, ", ")))
	}
	if len(task.Notes) > 0 {
		builder.WriteString(fmt.Sprintf("Notes: %s\n", strings.Join(task.Notes, "; ")))
	}
	return builder.String()
}
