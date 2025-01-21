// cmd/update.go

package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/database"
	"github.com/soyuz43/prbuddy-go/internal/github"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Fetch and process new merged pull requests from the remote repository.",
	Long:  `Fetches new merged pull requests since the tool was first initialized and stores their details in the SQLite database.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[prbuddy-go] Running update command...")

		// 1. Fetch GitHub remote URL
		remoteURL, err := github.GetRemoteURL()
		if err != nil {
			fmt.Printf("[prbuddy-go] Error fetching remote URL: %v\n", err)
			return
		}

		// 2. Initialize GitHub client
		pulls, err := github.FetchPullRequests(remoteURL)
		if err != nil {
			fmt.Printf("[prbuddy-go] Error fetching pull requests: %v\n", err)
			return
		}
		if len(pulls) == 0 {
			fmt.Println("[prbuddy-go] No pull requests found in the repository.")
			return
		}

		// 3. Initialize the database
		db, err := database.NewDatabase("prbuddy.db")
		if err != nil {
			fmt.Printf("[prbuddy-go] Error initializing database: %v\n", err)
			return
		}
		defer db.Close()

		// 4. Process and store new merged PRs
		for _, pr := range pulls {
			// Convert to database.PullRequest
			dbPR := database.ConvertGitHubPRToDatabasePR(&pr) // Pass pointer

			// Insert PR into the database
			err := db.InsertPullRequest(dbPR)
			if err != nil {
				fmt.Printf("[prbuddy-go] Error inserting PR #%d: %v\n", pr.Number, err)
				continue
			}

			// Note: Embedding and ChromaDB storage steps have been removed.
			// If you have additional processing, add it here.
		}

		fmt.Println("[prbuddy-go] Successfully updated with new merged pull requests.")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
