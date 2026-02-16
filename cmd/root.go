// cmd/root.go

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/dce"
	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func runRootCommand(cmd *cobra.Command, args []string) {
	color.Cyan("[PRBuddy-Go] Starting...\n")

	initialized, err := isInitialized()
	if err != nil {
		color.Red("Error checking initialization status: %v\n", err)
		os.Exit(1)
	}

	if initialized {
		runInteractiveSession()
	} else {
		showInitialMenu()
	}
}

func runInteractiveSession() {
	color.Green("\nPRBuddy-Go is initialized in this repository.\n")

	fmt.Println(bold("Available Commands:"))
	fmt.Printf("   %s    - %s\n", green("generate pr"), "Generate a draft pull request")
	fmt.Printf("   %s    - %s\n", green("what changed"), "Show changes since the last commit")
	fmt.Printf("   %s    - %s\n", green("quickassist"), "Open a persistent chat session with the assistant")
	fmt.Printf("   %s    - %s\n", green("dce"), "Dynamic Context Engine")
	fmt.Printf("   %s    - %s\n", green("context save"), "Save current conversation context")
	fmt.Printf("   %s    - %s\n", green("context load"), "Reload saved context for current branch/commit")
	fmt.Printf("   %s    - %s\n", green("serve"), "Start API server for extension integration")
	fmt.Printf("   %s    - %s\n", green("map"), "Generate project scaffolds")
	fmt.Printf("   %s    - %s\n", green("help"), "Show help information")
	fmt.Printf("   %s    - %s\n", red("remove"), "Uninstall PRBuddy-Go and delete all associated files")
	fmt.Printf("   %s    - %s\n", green("exit"), "Exit the tool")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n%s ", cyan(">"))
		input, err := reader.ReadString('\n')
		if err != nil {
			color.Red("Error reading input: %v\n", err)
			continue
		}

		parts := strings.Fields(strings.TrimSpace(input))
		if len(parts) == 0 {
			continue
		}

		command := strings.ToLower(parts[0])
		args := parts[1:]

		switch command {
		case "generate", "gen", "pr":
			handleGeneratePR()
		case "what", "w", "changes":
			handleWhatChanged()
		case "quickassist", "qa":
			handleQuickAssist(args, reader)
		case "dce":
			handleDCECommand()
		case "serve", "s":
			handleServeCommand()
		case "map":
			handleMapCommand()
		case "context":
			if len(args) < 1 {
				color.Red("Usage: context [save|load]")
				continue
			}
			switch args[0] {
			case "save":
				handleContextSave()
			case "load":
				handleContextLoad()
			default:
				color.Red("Unknown context subcommand. Use 'save' or 'load'.")
			}
		case "help", "h":
			printInteractiveHelp()
		case "remove", "uninstall":
			handleRemoveCommand()
		case "exit", "e", "quit", "q":
			color.Cyan("Exiting...\n")
			return
		default:
			color.Red("Unknown command. Type 'help' for available commands.\n")
		}
	}
}

func handleGeneratePR() {
	color.Cyan("\n[PRBuddy-Go] Generating draft PR...\n")
	runPostCommit(nil, nil)
}

func handleWhatChanged() {
	color.Cyan("\n[PRBuddy-Go] Checking changes...\n")
	whatCmd.Run(nil, nil)
}

func handleQuickAssist(args []string, reader *bufio.Reader) {
	if len(args) > 0 {
		singleQueryResponse(strings.Join(args, " "))
		return
	}
	startInteractiveQuickAssist(reader)
}

func singleQueryResponse(query string) {
	if query == "" {
		color.Red("No question provided.\n")
		return
	}

	resp, err := llm.HandleQuickAssist("", query)
	if err != nil {
		color.Red("Error: %v\n", err)
		return
	}

	color.Yellow("\nQuickAssist Response:\n")
	color.Cyan(resp)
	fmt.Println()
}

func startInteractiveQuickAssist(reader *bufio.Reader) {
	color.Cyan("\n[PRBuddy-Go] Quick Assist - Interactive Mode")
	color.Yellow("Type 'exit' or 'q' to end the session.\n")

	conversationID := ""

	for {
		color.Green("\nYou:")
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			color.Red("Error reading input: %v\n", err)
			continue
		}

		query := strings.TrimSpace(input)
		if shouldExit(query) {
			color.Cyan("\nEnding session.\n")
			return
		}

		if query == "" {
			color.Yellow("No question provided.\n")
			continue
		}

		resp, err := llm.HandleQuickAssist(conversationID, query)
		if err != nil {
			color.Red("Error: %v\n", err)
			continue
		}

		color.Blue("\nAssistant:\n")
		color.Cyan(resp)
		fmt.Println()

		if conv, exists := contextpkg.ConversationManagerInstance.GetConversation(conversationID); exists {
			conv.AddMessage("assistant", resp)
		}
	}
}

// cmd/what.go or wherever your command handlers are
func handleDCECommand() {
	color.Cyan("[PRBuddy-Go] Dynamic Context Engine - Interactive Mode")
	color.Yellow("Type 'exit' or 'q' to end the session.")

	// Initialize DCE
	dceInstance := dce.NewDCE()
	reader := bufio.NewReader(os.Stdin)

	color.Green("What are we working on today?")
	fmt.Print("> ")
	firstInput, err := reader.ReadString('\n')
	if err != nil {
		color.Red("Error reading input: %v", err)
		return
	}

	query := strings.TrimSpace(firstInput)
	if query == "" || query == "exit" || query == "q" {
		color.Red("No input provided. Exiting DCE.")
		return
	}

	// Activate DCE with the initial task
	if err := dceInstance.Activate(query); err != nil {
		color.Red("Error activating DCE: %v", err)
		return
	}

	// Get the conversation ID from the DCE context
	var conversationID string
	dce.GetDCEContextManager().ForEachContext(func(cid string, _ *dce.LittleGuy) {
		conversationID = cid
	})

	if conversationID == "" {
		color.Red("Failed to get conversation ID")
		return
	}

	// Interactive loop
	color.Green("DCE is active. Type your queries or DCE commands (/tasks, /status, etc.)")
	for {
		color.Green("You:")
		fmt.Print("> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			color.Red("Error reading input: %v", err)
			break
		}

		input = strings.TrimSpace(input)
		if input == "exit" || input == "q" {
			break
		}

		// Check if it's a DCE command
		littleguy, _ := dce.GetDCEContextManager().GetContext(conversationID)
		if littleguy != nil && dce.HandleDCECommandMenu(input, littleguy) {
			continue
		}

		// Process as regular query
		response, err := llm.HandleDCERequest(conversationID, input)
		if err != nil {
			color.Red("Error processing request: %v", err)
			continue
		}

		color.Cyan("Assistant:")
		fmt.Println(response)
	}

	// Deactivate DCE
	dceInstance.Deactivate(conversationID)
	color.Cyan("DCE deactivated. Exiting.")
}

func handleMapCommand() {
	mapCmd.Run(nil, nil)
}

func handleServeCommand() {
	color.Cyan("\n[PRBuddy-Go] Starting API server...\n")
	llm.ServeCmd.Run(nil, nil)
}

func handleRemoveCommand() {
	color.Red("\n⚠ WARNING: This will remove PRBuddy-Go from your repository! ⚠")
	color.Yellow("Are you sure? Type 'yes' to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(strings.ToLower(confirmation))

	if confirmation != "yes" {
		color.Cyan("Operation cancelled.")
		return
	}

	color.Red("\n[PRBuddy-Go] Removing PRBuddy-Go from the repository...\n")
	removeCmd.Run(nil, nil)
	color.Green("\n[PRBuddy-Go] Successfully uninstalled.\n")
}

func handleContextSave() {
	branch, err := utils.GetCurrentBranch()
	if err != nil {
		color.Red("Error getting branch: %v", err)
		return
	}
	commit, err := utils.GetLatestCommit()
	if err != nil {
		color.Red("Error getting commit hash: %v", err)
		return
	}

	conv, exists := contextpkg.ConversationManagerInstance.GetConversation("")
	if !exists {
		color.Yellow("No active conversation to save.\n")
		return
	}

	if err := llm.SaveDraftContext(branch, commit, conv.BuildContext()); err != nil {
		color.Red("Failed to save context: %v", err)
		return
	}
	color.Green("Conversation context saved for %s @ %s\n", branch, commit[:7])
}

func handleContextLoad() {
	branch, err := utils.GetCurrentBranch()
	if err != nil {
		color.Red("Error getting branch: %v", err)
		return
	}
	commit, err := utils.GetLatestCommit()
	if err != nil {
		color.Red("Error getting commit hash: %v", err)
		return
	}

	context, err := llm.LoadDraftContext(branch, commit)
	if err != nil {
		color.Red("Failed to load context: %v", err)
		return
	}

	conv := contextpkg.ConversationManagerInstance.StartConversation("", "", false)
	conv.SetMessages(context)
	color.Green("Context loaded for %s @ %s.\n", branch, commit[:7])
}

func joinMessages(msgs []contextpkg.Message) string {
	var sb strings.Builder
	caser := cases.Title(language.English)
	for _, m := range msgs {
		sb.WriteString(caser.String(m.Role))
		sb.WriteString(": ")
		sb.WriteString(m.Content)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func shouldExit(query string) bool {
	return strings.EqualFold(query, "exit") ||
		strings.EqualFold(query, "q") ||
		strings.EqualFold(query, "quit")
}

func printInitialHelp() {
	fmt.Println(bold("\nInitial Setup Commands:"))
	fmt.Printf("   %s    - %s\n", green("init"), "Initialize PRBuddy-Go in the current repository")
	fmt.Printf("   %s    - %s\n", green("help"), "Show this help information")
	fmt.Printf("   %s    - %s\n", green("exit"), "Exit the tool")
}

func printInteractiveHelp() {
	fmt.Println(bold("\nPull Request Workflow"))
	fmt.Printf("   %s    - %s\n", green("generate pr"), "Draft a new pull request with AI assistance")
	fmt.Printf("   %s    - %s\n", green("what changed"), "Show changes since your last commit")

	fmt.Println(bold("\nAssistant Tools"))
	fmt.Printf("   %s    - %s\n", green("quickassist"), "Chat live with the assistant (no memory)")
	fmt.Printf("   %s    - %s\n", green("dce"), "Enable Dynamic Context Engine (monitors task context)")
	fmt.Printf("   %s    - %s\n", green("context save"), "Save current conversation context")
	fmt.Printf("   %s    - %s\n", green("context load"), "Reload saved context for current branch/commit")

	fmt.Println(bold("\nProject Utilities"))
	fmt.Printf("   %s    - %s\n", green("map"), "Generate starter scaffolds for your project")
	fmt.Printf("   %s    - %s\n", green("serve"), "Start API server (for editor integration)")

	fmt.Println(bold("\nSystem"))
	fmt.Printf("   %s    - %s\n", green("help"), "Show this help information")
	fmt.Printf("   %s    - %s\n", red("remove"), "Uninstall PRBuddy-Go from this repository")
	fmt.Printf("   %s    - %s\n", green("exit"), "Exit the CLI")
}

func showInitialMenu() {
	color.Yellow("\nPRBuddy-Go is not initialized in this repository.\n")

	fmt.Println(bold("Available Commands:"))
	fmt.Printf("   %s    - %s\n", green("init"), "Initialize PRBuddy-Go in the current repository")
	fmt.Printf("   %s    - %s\n", green("help"), "Show help information")
	fmt.Printf("   %s    - %s\n", green("exit"), "Exit the tool")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n%s ", cyan(">"))
		input, err := reader.ReadString('\n')
		if err != nil {
			color.Red("Error reading input: %v\n", err)
			continue
		}

		command := strings.TrimSpace(strings.ToLower(input))

		switch command {
		case "init", "i":
			initCmd.Run(nil, nil)
			return
		case "help", "h":
			printInitialHelp()
		case "exit", "e", "quit", "q":
			color.Cyan("Exiting...\n")
			return
		default:
			color.Red("Unknown command. Type 'help' for available commands.\n")
		}
	}
}
