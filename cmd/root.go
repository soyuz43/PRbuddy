// cmd/root.go

package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "prbuddy-go",
	Short: "PRBuddy-Go: Enhance your pull request workflow.",
	Long:  `PRBuddy-Go helps automate pull request generation, manage Git hooks, and provide insightful feedback predictions.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default action when no subcommand is provided
		cmd.Help()
	},
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("Error executing command: %v", err)
		os.Exit(1)
	}
}

func init() {
	// Initialization logic if needed
}
