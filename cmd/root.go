package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
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
		case "init":
			initCmd.Run(nil, nil)
			return
		case "help":
			printInitialHelp()
		case "exit":
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
		case "generate pr", "gen pr", "pr":
			handleGeneratePR()
		case "what changed", "what", "changes":
			handleWhatChanged()
		case "help":
			printInteractiveHelp()
		case "exit", "quit", "q":
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
