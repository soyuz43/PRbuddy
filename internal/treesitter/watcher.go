package treesitter

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// WatchFiles sets up file watchers on files that are relevant to the active tasks.
// It uses fsnotify to monitor for changes and logs detected changes.
// This function does not update the task list.
func WatchFiles(conversationID string) {
	conversation, exists := contextpkg.ConversationManagerInstance.GetConversation(conversationID)
	if !exists {
		fmt.Println("No active conversation found.")
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Add each file from the current task list to the watcher.
	for _, task := range conversation.Tasks {
		for _, file := range task.Files {
			fmt.Printf("[Watcher] Verifying watcher for file: %s\n", file)
			err = watcher.Add(file)
			if err != nil {
				fmt.Printf("[Watcher] Error adding file %s to watcher: %v\n", file, err)
			}
		}
	}

	// Process file events.
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				fmt.Printf("[Watcher] Detected change in: %s\n", event.Name)
				// This function is observational only.
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("[Watcher] Error:", err)
		}
	}
}

// CheckForUnstagedChanges checks for unstaged changes using git diff and logs them.
func CheckForUnstagedChanges(conversationID string) {
	_, exists := contextpkg.ConversationManagerInstance.GetConversation(conversationID)
	if !exists {
		fmt.Println("No active conversation found.")
		return
	}

	diffOutput, err := utils.ExecGit("diff", "--name-only")
	if err != nil {
		fmt.Println("Failed to retrieve git diff:", err)
		return
	}

	changedFiles := utils.SplitLines(diffOutput)
	if len(changedFiles) == 0 {
		fmt.Println("No unstaged changes detected.")
		return
	}

	// Log the unstaged files.
	for _, file := range changedFiles {
		fmt.Printf("[Watcher] Unstaged change detected: %s\n", file)
	}
}

// CheckForUntrackedFiles detects new untracked files and logs them.
// It does not update the task list.
func CheckForUntrackedFiles() {
	diffOutput, err := utils.ExecGit("ls-files", "--others", "--exclude-standard")
	if err != nil {
		fmt.Println("Failed to retrieve untracked files:", err)
		return
	}

	untrackedFiles := utils.SplitLines(diffOutput)
	if len(untrackedFiles) == 0 {
		fmt.Println("No untracked files detected.")
		return
	}

	// Log untracked files.
	for _, file := range untrackedFiles {
		fmt.Printf("[Watcher] New untracked file detected: %s\n", file)
	}
}
