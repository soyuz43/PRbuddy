package utils

import (
	"errors"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
)

// ParseTasks converts raw task list input into a slice of Task objects.
// Expected task format per line: "Description | Files | Functions | Dependencies | Notes"
func ParseTasks(input string) ([]contextpkg.Task, error) {
	if input == "" {
		return nil, errors.New("empty task list input")
	}

	lines := strings.Split(input, "\n")
	var tasks []contextpkg.Task
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 1 {
			return nil, errors.New("invalid task format")
		}

		task := contextpkg.Task{
			Description:  strings.TrimSpace(parts[0]),
			Files:        parseList(parts, 1),
			Functions:    parseList(parts, 2),
			Dependencies: parseList(parts, 3),
			Notes:        parseList(parts, 4),
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// parseList safely splits and trims comma‐separated list items from a task part.
func parseList(parts []string, index int) []string {
	if index >= len(parts) {
		return nil
	}
	items := strings.Split(parts[index], ",")
	var trimmed []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			trimmed = append(trimmed, item)
		}
	}
	return trimmed
}
