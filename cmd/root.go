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
			handleDCECommand(args)
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
func handleQuickAssist(args []string, reader *bufio.Reader) {
	if len(args) > 0 {
		// Single query mode
		query := strings.Join(args, " ")
		singleQueryResponse(query)
		return
	}
	// Interactive loop
	startInteractiveQuickAssist(reader)
}

// Single-shot query (e.g., "quickassist how do I fix bug?")
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

	fmt.Println("\nQuickAssist Response:")
	color.Cyan(resp)
}

// Persistent chat session
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
		if strings.EqualFold(query, "exit") || strings.EqualFold(query, "q") {
			color.Cyan("\n[PRBuddy-Go] Ending Quick Assist session.\n")
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

		color.Blue("\nAssistant:")
		color.Cyan(resp)
	}
}

func handleDCECommand(args []string) {
	color.Cyan("\n[PRBuddy-Go] Dynamic Context Engine - Interactive Mode")
	color.Yellow("Type 'exit' or 'q' to end the session.\n")

	dceInstance := dce.NewDCE() // from your existing internal/dce/dce.go
	reader := bufio.NewReader(os.Stdin)

	// Step 1: Gather first user input to build initial tasks
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
	for _, line := range logs {
		color.White("  • %s", line)
	}

	// Create a "LittleGuy" to track tasks & code snapshots
	lg := dce.NewLittleGuy(tasks)

	// (Optional) If you want to pre-extract code for matched files:
	// For example, define ExtractCodeForTasks in dce.go:
	/*
	   codeMap, err := dceInstance.(*dce.DefaultDCE).ExtractCodeForTasks(tasks)
	   if err != nil {
	       color.Red("Error extracting code: %v\n", err)
	   } else {
	       for path, content := range codeMap {
	           lg.AddCodeSnippet(path, content)
	       }
	   }
	*/

	// Step 2: Start background monitoring for new changes
	lg.StartMonitoring()

	// Step 3: Enter multi-turn loop
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

		if query == "" {
			color.Yellow("No input provided.\n")
			continue
		}

		// Build ephemeral context from the "LittleGuy"
		messages := lg.BuildEphemeralContext(query)

		// Send to the LLM
		// (We can pass "" as conversationID to do ephemeral queries,
		// or make a new conversationID if you'd prefer persistent conversation.)
		llmResponse, err := llm.HandleQuickAssist("", joinMessages(messages))
		if err != nil {
			color.Red("LLM Error: %v\n", err)
			continue
		}

		color.Blue("\nAssistant:")
		color.Cyan(llmResponse)

		// Optionally: after each user message, you might do an immediate
		// diff check in the foreground. But we've already started
		// background monitoring with lg.StartMonitoring().
	}
}

// A helper to join multiple context messages into a single string for
// passing to HandleQuickAssist as if it’s one user query.
func joinMessages(msgs []contextpkg.Message) string {
	var sb strings.Builder
	for _, m := range msgs {
		sb.WriteString(fmt.Sprintf("%s: %s\n\n", strings.Title(m.Role), m.Content))
	}
	return sb.String()
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
	removeCmd.Run(nil, nil) // Call the remove command
	color.Green("\n[PRBuddy-Go] Successfully uninstalled.\n")
}

// Other Handlers
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
	fmt.Printf("   %s    - %s\n", red("remove"), "Uninstall PRBuddy-Go and delete all associated files") // 🔴 Highlighted in red
	fmt.Printf("   %s    - %s\n", green("exit"), "Exit the tool")
}
