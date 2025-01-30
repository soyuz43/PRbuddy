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
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// runRootCommand checks initialization and enters interactive menu
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
	fmt.Printf("   %s    - %s\n", green("serve"), "Start API server for extension integration")
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
		case "generate pr", "gen pr", "pr", "gen":
			handleGeneratePR()
		case "what changed", "what", "changes", "w":
			handleWhatChanged()
		case "quickassist", "qa":
			handleQuickAssist(args, reader)
		case "dce": // <-- New case for DCE
			handleDCECommand()
		case "serve", "s":
			handleServeCommand()
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

// 🟢 Quick Assist Handlers

// handleQuickAssist determines whether we're in single-query or interactive mode
func handleQuickAssist(args []string, reader *bufio.Reader) {
	if len(args) > 0 {
		// Single query mode (e.g. "quickassist how do I fix bug?")
		query := strings.Join(args, " ")
		singleQueryResponse(query)
		return
	}
	// Otherwise, interactive loop
	startInteractiveQuickAssist(reader)
}

// Single-shot query (e.g., "quickassist why is the sky blue")
func singleQueryResponse(query string) {
	if query == "" {
		color.Red("No question provided.\n")
		return
	}

	streamChan, err := llm.HandleQuickAssist("", query)
	if err != nil {
		color.Red("Error: %v\n", err)
		return
	}

	color.Yellow("\nQuickAssist Response:\n")

	for chunk := range streamChan {
		chunkStr := fmt.Sprintf("%v", chunk)
		fmt.Print(color.CyanString(chunkStr))
		os.Stdout.Sync()
	}
	fmt.Println()
}

// Persistent chat session (Interactive)
func startInteractiveQuickAssist(reader *bufio.Reader) {
	color.Cyan("\n[PRBuddy-Go] Quick Assist - Interactive Mode")
	color.Yellow("Type 'exit' or 'q' to end the session.\n")

	// If you want a new conversation ID each time, set it to ""
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
		if strings.EqualFold(query, "exit") || strings.EqualFold(query, "q") {
			color.Cyan("\n[PRBuddy-Go] Ending Quick Assist session.\n")
			return
		}

		if query == "" {
			color.Yellow("No question provided.\n")
			continue
		}

		streamChan, err := llm.HandleQuickAssist(conversationID, query)
		if err != nil {
			color.Red("Error: %v\n", err)
			continue
		}

		color.Blue("\nAssistant:\n")

		// Stream the response in real-time
		for chunk := range streamChan {
			chunkStr := fmt.Sprintf("%v", chunk)
			fmt.Print(color.CyanString(chunkStr))
			os.Stdout.Sync() // Flush immediately
		}
		fmt.Println() // New line after finishing
	}
}

// 🔵 DCE
func handleDCECommand() {
	color.Cyan("\n[PRBuddy-Go] Dynamic Context Engine - Interactive Mode")
	color.Yellow("Type 'exit' or 'q' to end the session.\n")

	dceInstance := dce.NewDCE()
	reader := bufio.NewReader(os.Stdin)

	// 1) Gather first user input to build initial tasks
	color.Green("\nYou:")
	fmt.Print("> ")
	firstInput, err := reader.ReadString('\n')
	if err != nil {
		color.Red("Error reading input: %v\n", err)
		return
	}
	query := strings.TrimSpace(firstInput)

	if query == "" {
		color.Red("No input provided. Exiting DCE.\n")
		return
	}

	// Build the initial task list
	tasks, logs, err := dceInstance.BuildTaskList(query)
	if err != nil {
		color.Red("Error building initial task list: %v\n", err)
		return
	}

	color.Yellow("\n[Initial DCE Logs]")
	for _, lg := range logs {
		color.White("  • %s", lg)
	}

	// Create a "LittleGuy" to track tasks & code snapshots
	littleGuy := dce.NewLittleGuy("", tasks)
	littleGuy.StartMonitoring() // Optional background monitoring for diffs

	// 2) Enter multi-turn loop
	for {
		color.Green("\nYou:")
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			color.Red("Error reading input: %v\n", err)
			continue
		}

		query = strings.TrimSpace(input)
		if strings.EqualFold(query, "exit") || strings.EqualFold(query, "q") {
			color.Cyan("\n[PRBuddy-Go] Exiting Dynamic Context Engine session.\n")
			return
		}

		// 3) Check if it's a recognized DCE command (like "/tasks")
		handled := dce.HandleDCECommandMenu(query, littleGuy)
		if handled {
			// The input was consumed by a command, so skip LLM
			continue
		}

		// 4) Otherwise, proceed with ephemeral LLM usage
		if query == "" {
			color.Yellow("No input provided.\n")
			continue
		}

		// Build ephemeral context from the "LittleGuy"
		messages := littleGuy.BuildEphemeralContext(query)

		// We pass the entire context as a single string to QuickAssist
		llmInput := joinMessages(messages)

		streamChan, err := llm.HandleQuickAssist("", llmInput)
		if err != nil {
			color.Red("LLM Error: %v\n", err)
			continue
		}

		color.Blue("\nAssistant:\n")
		// Stream the ephemeral LLM response in real-time
		for chunk := range streamChan {
			chunkStr := fmt.Sprintf("%v", chunk)
			fmt.Print(color.CyanString(chunkStr))
			os.Stdout.Sync()
		}
		fmt.Println()
	}
}

// Helper function to join multiple context messages into a single string
// for passing to LLM.HandleQuickAssist as one prompt.
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

// A helper to remove PRBuddy-Go
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
	removeCmd.Run(nil, nil) // Call the remove command
	color.Green("\n[PRBuddy-Go] Successfully uninstalled.\n")
}

// 🟢 Additional handlers for commands from the interactive menu
func handleGeneratePR() {
	color.Cyan("\n[PRBuddy-Go] Generating draft PR...\n")
	runPostCommit(nil, nil)
}

func handleWhatChanged() {
	color.Cyan("\n[PRBuddy-Go] Checking changes...\n")
	whatCmd.Run(nil, nil)
}

func handleServeCommand() {
	color.Cyan("\n[PRBuddy-Go] Starting API server...\n")
	llm.ServeCmd.Run(nil, nil)
}

// 🟢 Help and Formatting Functions
func printInitialHelp() {
	fmt.Println(bold("\nInitial Setup Commands:"))
	fmt.Printf("   %s    - %s\n", green("init"), "Initialize PRBuddy-Go in the current repository")
	fmt.Printf("   %s    - %s\n", green("help"), "Show this help information")
	fmt.Printf("   %s    - %s\n", green("exit"), "Exit the tool")
}

func printInteractiveHelp() {
	fmt.Println(bold("\nAvailable Commands:"))
	fmt.Printf("   %s    - %s\n", green("generate pr"), "Generate a draft pull request")
	fmt.Printf("   %s    - %s\n", green("what changed"), "Show changes since the last commit")
	fmt.Printf("   %s    - %s\n", green("quickassist"), "Open a persistent chat session with the assistant")
	fmt.Printf("   %s    - %s\n", green("dce"), "Dynamic Context Engine")
	fmt.Printf("   %s    - %s\n", green("serve"), "Start API server for extension integration")
	fmt.Printf("   %s    - %s\n", green("help"), "Show this help information")
	fmt.Printf("   %s    - %s\n", red("remove"), "Uninstall PRBuddy-Go and delete all associated files")
	fmt.Printf("   %s    - %s\n", green("exit"), "Exit the tool")
}
