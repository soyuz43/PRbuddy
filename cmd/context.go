// cmd/context.go
package cmd

import (
	"fmt"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage PRBuddy-Go conversation context",
}

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save current conversation context to disk",
	Run: func(cmd *cobra.Command, args []string) {
		branch, _ := utils.GetCurrentBranch()
		commit, _ := utils.GetLatestCommit()
		conv, exists := contextpkg.ConversationManagerInstance.GetConversation("current")
		if !exists {
			fmt.Println("No active conversation to save.")
			return
		}
		err := llm.SaveDraftContext(branch, commit, conv.BuildContext())
		if err != nil {
			fmt.Println("Error saving context:", err)
		} else {
			fmt.Printf("✅ Context saved for %s@%s\n", branch, commit[:7])
		}
	},
}

var loadCmd = &cobra.Command{
	Use:   "load [branch] [commit]",
	Short: "Load a saved context into memory",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		branch := args[0]
		commit := args[1]
		ctx, err := llm.LoadDraftContext(branch, commit)
		if err != nil {
			fmt.Println("❌ Failed to load context:", err)
			return
		}
		conv := contextpkg.ConversationManagerInstance.StartConversation("current", "", false)
		conv.SetMessages(ctx)
		fmt.Printf("✅ Loaded context for %s@%s into memory\n", branch, commit[:7])
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(saveCmd)
	contextCmd.AddCommand(loadCmd)
}
