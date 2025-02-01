// cmd/map.go

package cmd

import (
	"fmt"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/treesitter"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Generate project scaffolds using tree-sitter parsing",
	Long:  "Scans the repository using the Go parser, builds project metadata and a project map, and saves the results to scaffold files.",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Get repository root directory
		repoPath, err := utils.GetRepoPath()
		if err != nil {
			fmt.Printf("Error retrieving repository path: %v\n", err)
			return
		}

		// 2. Retrieve the current branch name
		branchName, err := utils.ExecGit("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			fmt.Printf("Error retrieving branch name: %v\n", err)
			return
		}
		branchName = strings.TrimSpace(branchName)

		// 3. Create a new Go parser (for now, we only support Go)
		parser := treesitter.NewGoParser()

		// 4. Build the project metadata
		metadata, err := parser.BuildProjectMetadata(repoPath)
		if err != nil {
			fmt.Printf("Error building project metadata: %v\n", err)
			return
		}

		// 5. Build the project map (function dependency map)
		projectMap, err := parser.BuildProjectMap(repoPath)
		if err != nil {
			fmt.Printf("Error building project map: %v\n", err)
			return
		}

		// 6. Save the metadata and project map using the saver functions
		if err := treesitter.SaveMetadata(metadata, branchName); err != nil {
			fmt.Printf("Error saving project metadata: %v\n", err)
			return
		}
		if err := treesitter.SaveProjectMap(projectMap, branchName); err != nil {
			fmt.Printf("Error saving project map: %v\n", err)
			return
		}

		fmt.Println("Project scaffolds generated successfully.")
	},
}

func init() {
	rootCmd.AddCommand(mapCmd)
}
