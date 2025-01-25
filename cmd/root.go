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

// initialization
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

func handleServeCommand() {
	color.Cyan("\n[PRBuddy-Go] Starting API server...\n")
	llm.ServeCmd.Run(nil, nil) // This will use the ServeCmd from your llm package
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

func runInteractiveSession() {
	color.Green("\nPRBuddy-Go is initialized in this repository.\n")

	fmt.Println(bold("Available Commands:"))
	fmt.Printf("   %s    - %s\n", green("generate pr"), "Generate a draft pull request")
	fmt.Printf("   %s    - %s\n", green("what changed"), "Show changes since the last commit")
	fmt.Printf("   %s    - %s\n", green("quickassist"), "Get AI assistance for coding questions")
	fmt.Printf("   %s    - %s\n", green("serve"), "Start API server for extension integration")
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

		// Split the input into command and arguments
		parts := strings.Fields(strings.TrimSpace(input))
		if len(parts) == 0 {
			continue
		}

		command := strings.ToLower(parts[0])
		args := parts[1:] // Extract arguments from the input

		switch command {
		case "generate pr", "gen pr", "pr", "gen":
			handleGeneratePR()
		case "what changed", "what", "changes", "w":
			handleWhatChanged()
		case "quickassist", "qa":
			handleQuickAssist(args, reader) // Pass args and reader to the handler
		case "serve", "s":
			handleServeCommand()
		case "help", "h":
			printInteractiveHelp()
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
	color.Cyan("\n[PRBuddy-Go] Quick Assist - Ask me anything about your code!\n")

	var query string
	if len(args) > 0 {
		query = strings.Join(args, " ")
	} else {
		fmt.Print("Enter your question: ")
		input, _ := reader.ReadString('\n')
		query = strings.TrimSpace(input)
	}

	if query == "" {
		color.Red("No question provided\n")
		return
	}

	response, err := llm.HandleCLIQuickAssist(query)
	if err != nil {
		color.Red("Error: %v\n", err)
		return
	}

	fmt.Println("\n" + green("Assistant Response:"))
	fmt.Println(response)
}

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
	fmt.Printf("   %s    - %s\n", green("help"), "Show this help information")
	fmt.Printf("   %s    - %s\n", green("exit"), "Exit the tool")
}
