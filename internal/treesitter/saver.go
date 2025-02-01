package treesitter

import (
	"fmt"
	"time"

	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// -----------------------------------------------------------------------------
// Output Path Helpers
// -----------------------------------------------------------------------------

// getMetadataOutputPath returns the output path for the metadata file.
// If branchName is provided, it includes the branch name in the filename.
// Otherwise, it falls back to the format: project_metadata-<month>-<day>.json
func getMetadataOutputPath(branchName string) string {
	now := time.Now()
	if branchName != "" {
		return fmt.Sprintf(".git/pr_buddy_db/scaffold/project_metadata-%s-%02d-%02d.json", branchName, now.Month(), now.Day())
	}
	return fmt.Sprintf(".git/pr_buddy_db/scaffold/project_metadata-%02d-%02d.json", now.Month(), now.Day())
}

// getProjectMapOutputPath returns the output path for the project map file.
// If branchName is provided, it includes the branch name in the filename.
func getProjectMapOutputPath(branchName string) string {
	now := time.Now()
	if branchName != "" {
		return fmt.Sprintf(".git/pr_buddy_db/scaffold/project_map-%s-%02d-%02d.json", branchName, now.Month(), now.Day())
	}
	return fmt.Sprintf(".git/pr_buddy_db/scaffold/project_map-%02d-%02d.json", now.Month(), now.Day())
}

// -----------------------------------------------------------------------------
// Saving Functions (Using utils.WriteFile and utils.MarshalJSON)
// -----------------------------------------------------------------------------

// SaveMetadata writes the given metadata to a JSON file using atomic file writes.
func SaveMetadata(metadata *ProjectMetadata, branchName string) error {
	// Use the utils.MarshalJSON function to get a pretty-printed JSON string.
	jsonStr, err := utils.MarshalJSON(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal project metadata: %w", err)
	}
	data := []byte(jsonStr)
	outputPath := getMetadataOutputPath(branchName)
	return utils.WriteFile(outputPath, data)
}

// SaveProjectMap writes the given project map to a JSON file using atomic file writes.
func SaveProjectMap(projectMap *ProjectMap, branchName string) error {
	jsonStr, err := utils.MarshalJSON(projectMap)
	if err != nil {
		return fmt.Errorf("failed to marshal project map: %w", err)
	}
	data := []byte(jsonStr)
	outputPath := getProjectMapOutputPath(branchName)
	return utils.WriteFile(outputPath, data)
}
