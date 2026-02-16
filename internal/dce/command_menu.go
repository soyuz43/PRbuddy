// internal/dce/command_menu.go

package dce

import (
	"fmt"
	"strconv"
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
	lowerInput := strings.ToLower(trimmedInput)

	switch {
	case lowerInput == "/tasks":
		displayTaskList(littleguy, false)
		return true

	case lowerInput == "/tasks -v" || lowerInput == "/tasks verbose":
		displayTaskList(littleguy, true)
		return true

	case strings.HasPrefix(lowerInput, "/dce "):
		handleDCEControlCommand(trimmedInput[5:], littleguy)
		return true

	case lowerInput == "/commands" || lowerInput == "/cmds":
		displayCommandMenu()
		return true

	case lowerInput == "/priority" || strings.HasPrefix(lowerInput, "/priority "):
		handlePriorityCommand(trimmedInput, littleguy)
		return true

	case lowerInput == "/complete" || strings.HasPrefix(lowerInput, "/complete "):
		handleCompleteCommand(trimmedInput, littleguy)
		return true

	case lowerInput == "/refresh":
		refreshTaskList(littleguy)
		return true

	case lowerInput == "/status":
		displayDCEStatus(littleguy)
		return true

	default:
		return false
	}
}

// displayTaskList prints the current task list.
// If verbose=true, it includes additional details like files, functions, notes, etc.
func displayTaskList(littleguy *LittleGuy, verbose bool) {
	color.Cyan("\n[Task List] Current Tasks:")

	littleguy.mutex.RLock()
	tasks := littleguy.tasks
	littleguy.mutex.RUnlock()

	if len(tasks) == 0 {
		color.Yellow("  [!] No active tasks")
		return
	}

	for i, task := range tasks {
		fmt.Printf("  %d) %s\n", i+1, task.Description)

		if verbose {
			if len(task.Files) > 0 {
				fmt.Printf("     Files: %s\n", strings.Join(task.Files, ", "))
			}
			if len(task.Functions) > 0 {
				fmt.Printf("     Functions: %s\n", strings.Join(task.Functions, ", "))
			}
			if len(task.Notes) > 0 {
				fmt.Printf("     Notes: %s\n", strings.Join(task.Notes, "; "))
			}
		}
	}
}

// handleDCEControlCommand processes DCE control commands like "on" and "off"
func handleDCEControlCommand(command string, littleguy *LittleGuy) {
	lowerCmd := strings.ToLower(strings.TrimSpace(command))

	switch lowerCmd {
	case "on", "activate", "start":
		littleguy.mutex.Lock()
		wasActive := littleguy.monitorStarted
		littleguy.mutex.Unlock()

		if !wasActive {
			littleguy.StartMonitoring()
			color.Green("[DCE] Dynamic Context Engine activated")
			color.Green("[DCE] Use '/tasks' to view current development tasks")
		} else {
			color.Yellow("[DCE] DCE is already active")
		}

	case "off", "deactivate", "stop":
		littleguy.mutex.Lock()
		wasActive := littleguy.monitorStarted
		littleguy.mutex.Unlock()

		if wasActive {
			// Instead of StopMonitoring (which doesn't exist yet), we'll set a flag
			littleguy.mutex.Lock()
			littleguy.monitorStarted = false
			littleguy.mutex.Unlock()
			color.Green("[DCE] Dynamic Context Engine deactivated")
		} else {
			color.Yellow("[DCE] DCE is already inactive")
		}

	case "status", "info":
		displayDCEStatus(littleguy)

	default:
		color.Red("[X] Unknown DCE command. Use '/dce on', '/dce off', or '/dce status'")
	}
}

// displayDCEStatus shows detailed DCE status information
func displayDCEStatus(littleguy *LittleGuy) {
	color.Cyan("\n[DCE Status] Engine Status:")

	littleguy.mutex.RLock()
	status := "ACTIVE"
	if !littleguy.monitorStarted {
		status = "INACTIVE"
	}
	taskCount := len(littleguy.tasks)
	littleguy.mutex.RUnlock()

	fmt.Printf("  Status: %s\n", status)
	fmt.Printf("  Active Tasks: %d\n", taskCount)
	fmt.Printf("  Monitoring Interval: %v\n", littleguy.pollInterval)
	fmt.Println("  Features: Dynamic task tracking, Git change monitoring")
}

// handlePriorityCommand allows users to set task priorities
func handlePriorityCommand(input string, littleguy *LittleGuy) {
	parts := strings.Fields(input)

	if len(parts) < 2 {
		color.Red("[X] Usage: /priority <task-number> <low|medium|high>")
		return
	}

	if len(parts) == 2 {
		// Display current priorities
		color.Cyan("\n[Priority] Current task priorities:")
		littleguy.mutex.RLock()
		defer littleguy.mutex.RUnlock()

		for i, task := range littleguy.tasks {
			priority := "Low"
			for _, note := range task.Notes {
				if strings.Contains(strings.ToLower(note), "high priority") {
					priority = "High"
					break
				} else if strings.Contains(strings.ToLower(note), "medium priority") {
					priority = "Medium"
				}
			}
			fmt.Printf("  %d) [%s] %s\n", i+1, priority, task.Description)
		}
		return
	}

	// Set priority for a specific task
	taskNumStr := parts[1]
	var priorityLevel string

	if len(parts) > 2 {
		priorityLevel = strings.ToLower(parts[2])
	}

	// Convert task number
	taskNum, err := strconv.Atoi(taskNumStr)
	if err != nil || taskNum < 1 {
		color.Red("[X] Invalid task number")
		return
	}

	// Update task priority
	littleguy.mutex.Lock()
	defer littleguy.mutex.Unlock()

	if taskNum > len(littleguy.tasks) {
		color.Red("[X] Task number out of range")
		return
	}

	// Remove existing priority notes
	task := &littleguy.tasks[taskNum-1]
	newNotes := []string{}
	for _, note := range task.Notes {
		lowerNote := strings.ToLower(note)
		if !strings.Contains(lowerNote, "priority") {
			newNotes = append(newNotes, note)
		}
	}

	// Add new priority note
	switch priorityLevel {
	case "high", "urgent", "critical":
		newNotes = append(newNotes, "High Priority: Critical task requiring immediate attention")
		color.Green("[Priority] Task %d set to HIGH priority", taskNum)
	case "medium", "normal":
		newNotes = append(newNotes, "Medium Priority: Important but not time-critical")
		color.Green("[Priority] Task %d set to MEDIUM priority", taskNum)
	case "low", "optional":
		newNotes = append(newNotes, "Low Priority: Can be addressed later")
		color.Green("[Priority] Task %d set to LOW priority", taskNum)
	default:
		color.Red("[X] Invalid priority level. Use: low, medium, or high")
		return
	}

	task.Notes = newNotes
}

// handleCompleteCommand marks tasks as completed
func handleCompleteCommand(input string, littleguy *LittleGuy) {
	parts := strings.Fields(input)

	if len(parts) < 2 {
		color.Red("[X] Usage: /complete <task-number>")
		return
	}

	// Convert task number
	taskNum, err := strconv.Atoi(parts[1])
	if err != nil || taskNum < 1 {
		color.Red("[X] Invalid task number")
		return
	}

	// Mark task as completed
	littleguy.mutex.Lock()
	defer littleguy.mutex.Unlock()

	if taskNum > len(littleguy.tasks) {
		color.Red("[X] Task number out of range")
		return
	}

	task := littleguy.tasks[taskNum-1]
	littleguy.tasks = append(littleguy.tasks[:taskNum-1], littleguy.tasks[taskNum:]...)

	color.Green("[Complete] Task %d marked as completed: %s", taskNum, task.Description)
}

// refreshTaskList manually triggers a task list refresh
func refreshTaskList(littleguy *LittleGuy) {
	color.Cyan("\n[Refresh] Refreshing task list from git changes...")

	// Directly call RefreshTaskListFromGitChanges with the conversation ID
	err := RefreshTaskListFromGitChanges(littleguy.conversationID)
	if err != nil {
		color.Red("[X] Failed to refresh task list: %v", err)
		return
	}

	color.Green("[Refresh] Task list updated with latest changes")
}

// displayCommandMenu shows available special commands for DCE
func displayCommandMenu() {
	color.Green("\n[Commands] Available DCE Commands:")
	fmt.Println("  /tasks                - Show the current task list (concise)")
	fmt.Println("  /tasks verbose        - Show the task list with additional details")
	fmt.Println("  /dce on               - Activate the Dynamic Context Engine")
	fmt.Println("  /dce off              - Deactivate the Dynamic Context Engine")
	fmt.Println("  /dce status           - Show DCE status and statistics")
	fmt.Println("  /priority             - Show current task priorities")
	fmt.Println("  /priority <num> <level> - Set task priority (low/medium/high)")
	fmt.Println("  /complete <num>       - Mark a task as completed")
	fmt.Println("  /refresh              - Manually refresh task list from git")
	fmt.Println("  /status               - Show detailed DCE status")
	fmt.Println("  /commands             - Show this command menu")
}
