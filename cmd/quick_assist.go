// cmd/quick_assist.go

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/spf13/cobra"
)

var quickAssistCmd = &cobra.Command{
	Use:     "quickassist [query]",
	Aliases: []string{"qa"},
	Short:   "Get quick assistance from the LLM (interactive mode if no query provided)",
	Args:    cobra.ArbitraryArgs, // Allows zero or more arguments
	Run: func(cmd *cobra.Command, args []string) {
		// If user provides arguments, treat it as a one-time query
		if len(args) > 0 {
			query := strings.Join(args, " ")
			handleSingleQuickAssist(query)
			return
		}

		// Otherwise, start interactive chat session
		StartInteractiveQuickAssist()
	},
}

func handleSingleQuickAssist(query string) {
	if strings.TrimSpace(query) == "" {
		color.Red("Error: No question provided.\n")
		return
	}

	// Use a new conversation (empty ConversationID to generate a new one)
	response, err := llm.HandleQuickAssist("", query)
	if err != nil {
		color.Red("Error: %v\n", err)
		return
	}

	// Display assistant response
	fmt.Println("\nQuickAssist Response:")
	color.Cyan(response)
}

// StartInteractiveQuickAssist starts the interactive chat session.
// Exported so it can be called from root.go
func StartInteractiveQuickAssist() {
	color.Cyan("\n[PRBuddy-Go] Quick Assist - Interactive Mode")
	color.Yellow("Type 'exit' or 'q' to end the session.\n")

	reader := bufio.NewReader(os.Stdin)
	conversationID := "" // Start a new conversation

	for {
		// Prompt for user input
		color.Green("You:")
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			color.Red("Error reading input: %v\n", err)
			continue
		}

		// Trim spaces and check for exit condition
		query := strings.TrimSpace(input)
		if strings.EqualFold(query, "exit") || strings.EqualFold(query, "q") {
			color.Cyan("\n[PRBuddy-Go] Ending Quick Assist session.\n")
			break
		}

		if query == "" {
			color.Yellow("Please enter a valid question or type 'exit' to quit.")
			continue
		}

		// Get response from Quick Assist
		response, err := llm.HandleQuickAssist(conversationID, query)
		if err != nil {
			color.Red("Error: %v\n", err)
			continue
		}

		// Display assistant response
		color.Blue("Assistant:")
		color.Cyan(response)
	}
}

func init() {
	rootCmd.AddCommand(quickAssistCmd)
	// Removed unnecessary flags
}
