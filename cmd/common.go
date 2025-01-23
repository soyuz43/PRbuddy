package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

// Color definitions
var (
	cyan  = color.New(color.FgCyan).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	bold  = color.New(color.Bold).SprintFunc()
)

// Root command
var rootCmd = &cobra.Command{
	Use:   "prbuddy-go",
	Short: "PRBuddy-Go: Enhance your pull request workflow.",
	Long:  `PRBuddy-Go helps automate pull request generation, manage Git hooks, and provide insightful feedback predictions.`,
	Run:   runRootCommand,
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		color.Red("Error executing command: %v\n", err)
		os.Exit(1)
	}
}

// isInitialized checks if PRBuddy is initialized in the current repository
func isInitialized() (bool, error) {
	repoPath, err := utils.GetRepoPath()
	if err != nil {
		return false, fmt.Errorf("failed to get repository path: %w", err)
	}

	// Check for the existence of the pr_buddy_db directory
	prBuddyDBPath := filepath.Join(repoPath, ".git", "pr_buddy_db")
	if _, err := os.Stat(prBuddyDBPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil // Not initialized
		}
		return false, fmt.Errorf("error checking pr_buddy_db: %w", err)
	}

	return true, nil // Initialized
}
