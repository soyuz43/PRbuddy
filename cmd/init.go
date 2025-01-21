// cmd/init.go

package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/database"
	"github.com/soyuz43/prbuddy-go/internal/github"
	"github.com/soyuz43/prbuddy-go/internal/hooks"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize PRBuddy in the current Git repository.",
	Long:  `Fetches PRs from GitHub, stores them in SQLite, embeds them in ChromaDB, and installs a post-commit hook.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Fetch the GitHub remote URL
		remoteURL, err := github.GetRemoteURL()
		if err != nil {
			fmt.Printf("[prbuddy-go] Error fetching remote URL: %v\n", err)
			return
		}
		fmt.Printf("[prbuddy-go] Found remote repository: %s\n", remoteURL)

		// Initialize the database
		db, err := database.NewDatabase("prbuddy.db")
		if err != nil {
			fmt.Printf("[prbuddy-go] Error initializing database: %v\n", err)
			return
		}
		defer db.Close()

		// Fetch pull requests from GitHub
		pulls, err := github.FetchPullRequests(remoteURL)
		if err != nil {
			fmt.Printf("[prbuddy-go] Error fetching pull requests: %v\n", err)
			return
		}
		if len(pulls) == 0 {
			fmt.Println("[prbuddy-go] No pull requests found.")
			return
		}

		// Process and store pull requests
		for _, pr := range pulls {
			dbPR := database.ConvertGitHubPRToDatabasePR(&pr) // Pass pointer
			err := db.InsertPullRequest(dbPR)
			if err != nil {
				fmt.Printf("[prbuddy-go] Error inserting PR #%d: %v\n", pr.Number, err)
				continue
			}
			// Additional processing like embedding can be added here
		}
		fmt.Println("[prbuddy-go] Processing complete.")

		// Install the post-commit hook
		err = hooks.InstallPostCommitHook()
		if err != nil {
			fmt.Printf("[prbuddy-go] Error installing post-commit hook: %v\n", err)
			return
		}
		fmt.Println("[prbuddy-go] post-commit hook installation complete.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
