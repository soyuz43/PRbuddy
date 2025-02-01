package treesitter

import (
	"fmt"
)

// -----------------------------------------------------------------------------
// Project Knowledge Update Functions & Triggers
// -----------------------------------------------------------------------------

// RefreshProjectKnowledge rebuilds the project metadata and map and saves them.
// The branchName parameter allows for branch-specific storage if desired.
func RefreshProjectKnowledge(rootDir, branchName string) error {
	parser := NewGoParser()

	// Build metadata.
	metadata, err := parser.BuildProjectMetadata(rootDir)
	if err != nil {
		return fmt.Errorf("failed to build project metadata: %w", err)
	}
	if err := SaveMetadata(metadata, branchName); err != nil {
		return fmt.Errorf("failed to save project metadata: %w", err)
	}

	// Build project map.
	projectMap, err := parser.BuildProjectMap(rootDir)
	if err != nil {
		return fmt.Errorf("failed to build project map: %w", err)
	}
	if err := SaveProjectMap(projectMap, branchName); err != nil {
		return fmt.Errorf("failed to save project map: %w", err)
	}

	fmt.Println("Project knowledge refreshed successfully.")
	return nil
}

// OnCommit is called on git commit. It triggers a refresh of the project map.
func OnCommit(rootDir, branchName string) error {
	fmt.Println("Trigger: OnCommit - Refreshing project map.")
	return RefreshProjectKnowledge(rootDir, branchName)
}

// OnPull is called on git pull. It triggers a refresh to sync remote changes.
func OnPull(rootDir, branchName string) error {
	fmt.Println("Trigger: OnPull - Refreshing project map.")
	return RefreshProjectKnowledge(rootDir, branchName)
}

// OnMerge is called on git merge. It refreshes the project map to capture new changes.
func OnMerge(rootDir, branchName string) error {
	fmt.Println("Trigger: OnMerge - Refreshing project map.")
	return RefreshProjectKnowledge(rootDir, branchName)
}

// OnCheckout is called on git checkout. The map should reflect the new branch state.
func OnCheckout(rootDir, branchName string) error {
	fmt.Println("Trigger: OnCheckout - Refreshing project map.")
	return RefreshProjectKnowledge(rootDir, branchName)
}

// ManualRefresh allows a manual request (e.g., via a /refresh-map command) to update the map.
func ManualRefresh(rootDir, branchName string) error {
	fmt.Println("Trigger: ManualRefresh - Refreshing project map.")
	return RefreshProjectKnowledge(rootDir, branchName)
}
