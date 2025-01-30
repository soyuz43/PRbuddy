// internal/dce/command_menu.go

package dce

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// HandleDCECommandMenu checks if the user input is a recognized command
// and executes the appropriate function.
//
// Returns 'true' if the input matched a command and was handled internally.
// Returns 'false' if the input did not match a command, so it can be passed to the LLM.
func HandleDCECommandMenu(input string, littleguy *LittleGuy) bool {
	trimmedInput := strings.TrimSpace(input)

	switch {
	case trimmedInput == "/tasks":
		displayTaskList(littleguy, false)
		return true

	case trimmedInput == "/tasks -v":
		displayTaskList(littleguy, true)
		return true

	case trimmedInput == "/commands", trimmedInput == "/cmds":
		displayCommandMenu()
		return true

	default:
		// Not a recognized command
		return false
	}
}

// displayTaskList prints the current task list.
// If verbose=true, it includes additional details like files, functions, notes, etc.
func displayTaskList(littleguy *LittleGuy, verbose bool) {
	color.Cyan("\n📌 Current Task List:\n")

	littleguy.mutex.RLock()
	tasks := littleguy.tasks
	littleguy.mutex.RUnlock()

	if len(tasks) == 0 {
		color.Yellow("  (No active tasks)\n")
		return
	}

	for i, task := range tasks {
		fmt.Printf("  %d) %s\n", i+1, task.Description)
		if verbose {
			// Print all task details
			if len(task.Files) > 0 {
				fmt.Printf("     📂 Files: %s\n", strings.Join(task.Files, ", "))
			}
			if len(task.Functions) > 0 {
				fmt.Printf("     🔧 Functions: %s\n", strings.Join(task.Functions, ", "))
			}
			if len(task.Notes) > 0 {
				fmt.Printf("     📝 Notes: %s\n", strings.Join(task.Notes, "; "))
			}
		}
	}
}

// displayCommandMenu shows available special commands for DCE
func displayCommandMenu() {
	color.Green("\n🛠 Available DCE Commands:\n")
	fmt.Println("  /tasks       - Show the current task list (concise)")
	fmt.Println("  /tasks -v    - Show the task list with additional details")
	fmt.Println("  /commands    - Show this command menu")
	fmt.Println("  /cmd1        - [Placeholder] Future feature")
	fmt.Println("  /cmd2        - [Placeholder] Future feature")
}
