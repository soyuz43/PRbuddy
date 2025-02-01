package treesitter

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// -----------------------------------------------------------------------------
// Output Path Helpers
// -----------------------------------------------------------------------------

// getMetadataOutputPath returns the output path for the metadata file.
// For branch-specific files, you could modify this function.
func getMetadataOutputPath(branchName string) string {
	// For branch-specific naming, uncomment the next line:
	// return fmt.Sprintf(".git/pr_buddy_db/scaffold/project_metadata_%s.json", branchName)
	return ".git/pr_buddy_db/scaffold/project_metadata.json"
}

// getProjectMapOutputPath returns the output path for the project map file.
func getProjectMapOutputPath(branchName string) string {
	// For branch-specific naming, uncomment the next line:
	// return fmt.Sprintf(".git/pr_buddy_db/scaffold/project_map_%s.json", branchName)
	return ".git/pr_buddy_db/scaffold/project_map.json"
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
