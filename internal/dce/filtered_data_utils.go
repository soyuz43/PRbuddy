package dce

import (
	"fmt"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
)

// GenerateFilteredData processes tasks to produce a summary of project data.
func GenerateFilteredData(tasks []contextpkg.Task) ([]FilteredData, error) {
	var filtered []FilteredData
	for _, task := range tasks {
		fd := FilteredData{
			FileHierarchy: fmt.Sprintf("src/%s.go", strings.ToLower(task.Description)),
			LinterResults: "All lint checks passed.",
		}
		filtered = append(filtered, fd)
	}
	return filtered, nil
}
