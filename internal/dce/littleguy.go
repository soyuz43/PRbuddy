// internal/dce/littleguy.go

package dce

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
)

// LittleGuy tracks the ephemeral codebase snapshot and tasks for a single DCE session.
type LittleGuy struct {
	mutex         sync.RWMutex
	tasks         []contextpkg.Task // Ongoing tasks
	completed     []contextpkg.Task // Completed tasks
	codeSnapshots map[string]string // filePath -> file content
	pollInterval  time.Duration     // how often to check diffs

	// optionally track whether we've started the background goroutine
	monitorStarted bool
}

// NewLittleGuy initializes an in-memory ephemeral “LittleGuy” object.
func NewLittleGuy(initialTasks []contextpkg.Task) *LittleGuy {
	return &LittleGuy{
		tasks:         initialTasks,
		completed:     make([]contextpkg.Task, 0),
		codeSnapshots: make(map[string]string),
		pollInterval:  10 * time.Second, // default poll interval
	}
}

// StartMonitoring launches a background goroutine that periodically calls UpdateFromDiff.
func (lg *LittleGuy) StartMonitoring() {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	if lg.monitorStarted {
		return
	}
	lg.monitorStarted = true

	go func() {
		for {
			time.Sleep(lg.pollInterval)
			diffOutput, err := runGitDiff()
			if err != nil {
				// optional logging
				color.Red("[LittleGuy] Failed to run git diff: %v\n", err)
				continue
			}
			if diffOutput != "" {
				lg.UpdateFromDiff(diffOutput)
			}
		}
	}()
}

// UpdateFromDiff silently updates tasks based on the current git diff.
//   - If new methods are added, it creates new subtasks
//   - If tasks appear completed, it moves them to completed.
func (lg *LittleGuy) UpdateFromDiff(diff string) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	newFuncs, importExports := parseNewMethods(diff) // ✅ Corrected assignment

	// Process new functions
	for _, nf := range newFuncs {
		if nf.Action == "added" {
			// Create a subtask for the new function
			subtask := contextpkg.Task{
				Description: fmt.Sprintf("New method %s was added.", nf.FunctionName),
				Files:       []string{nf.FilePath},
				Functions:   []string{nf.FunctionName},
				Notes: []string{
					"Augment the test suite for " + nf.FunctionName,
					"Update API documentation if public",
				},
			}
			lg.tasks = append(lg.tasks, subtask)
		} else if nf.Action == "removed" {
			// Handle removed functions (if needed)
			lg.markTaskAsCompleted(nf.FunctionName)
		}
	}

	// Process import/export changes
	for _, impExp := range importExports {
		if impExp.Action == "added" {
			// Suggest reviewing new dependencies
			lg.tasks = append(lg.tasks, contextpkg.Task{
				Description: fmt.Sprintf("New import detected: %s", impExp.Statement),
				Files:       []string{impExp.FilePath},
				Notes:       []string{"Review dependency impact and update documentation"},
			})
		} else if impExp.Action == "removed" {
			// Suggest cleaning up unused dependencies
			lg.tasks = append(lg.tasks, contextpkg.Task{
				Description: fmt.Sprintf("Import removed: %s", impExp.Statement),
				Files:       []string{impExp.FilePath},
				Notes:       []string{"Check for orphaned references and clean up"},
			})
		}
	}
}

// markTaskAsCompleted moves completed tasks to a separate list
func (lg *LittleGuy) markTaskAsCompleted(funcName string) {
	for i, task := range lg.tasks {
		if contains(task.Functions, funcName) {
			lg.completed = append(lg.completed, task)
			lg.tasks = append(lg.tasks[:i], lg.tasks[i+1:]...) // Remove from active tasks
			break
		}
	}
}

// contains checks if a slice contains a specific value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// UpdateTaskList appends new tasks to the existing in-memory list.
func (lg *LittleGuy) UpdateTaskList(newTasks []contextpkg.Task) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.tasks = append(lg.tasks, newTasks...)
}

// BuildEphemeralContext aggregates tasks, code snapshots, and user input into a final LLM context.
func (lg *LittleGuy) BuildEphemeralContext(userQuery string) []contextpkg.Message {
	lg.mutex.RLock()
	defer lg.mutex.RUnlock()

	var messages []contextpkg.Message

	// 1. System introduction
	messages = append(messages, contextpkg.Message{
		Role:    "system",
		Content: "You are a helpful developer assistant. We maintain a dynamic list of tasks and code snapshots in memory.",
	})

	// 2. Show uncompleted tasks
	if len(lg.tasks) > 0 {
		var builder strings.Builder
		builder.WriteString("Current Tasks:\n")
		for i, t := range lg.tasks {
			builder.WriteString(fmt.Sprintf("  %d) %s\n", i+1, t.Description))
			if len(t.Notes) > 0 {
				builder.WriteString(fmt.Sprintf("     Notes: %v\n", t.Notes))
			}
			if len(t.Files) > 0 {
				builder.WriteString(fmt.Sprintf("     Files: %v\n", t.Files))
			}
			if len(t.Functions) > 0 {
				builder.WriteString(fmt.Sprintf("     Functions: %v\n", t.Functions))
			}
		}
		messages = append(messages, contextpkg.Message{
			Role:    "system",
			Content: builder.String(),
		})
	}

	// 3. Show completed tasks (optional)
	if len(lg.completed) > 0 {
		var builder strings.Builder
		builder.WriteString("Completed Tasks:\n")
		for i, t := range lg.completed {
			builder.WriteString(fmt.Sprintf("  %d) %s\n", i+1, t.Description))
		}
		messages = append(messages, contextpkg.Message{
			Role:    "system",
			Content: builder.String(),
		})
	}

	// 4. Code snapshots (optional)
	if len(lg.codeSnapshots) > 0 {
		builder := strings.Builder{}
		builder.WriteString("Code Snippets:\n\n")
		for path, content := range lg.codeSnapshots {
			builder.WriteString(fmt.Sprintf("File: %s\n---\n%s\n---\n", path, content))
		}
		messages = append(messages, contextpkg.Message{
			Role:    "system",
			Content: builder.String(),
		})
	}

	// 5. Finally, the user’s query
	messages = append(messages, contextpkg.Message{
		Role:    "user",
		Content: userQuery,
	})

	return messages
}

// AddCodeSnippet stores a snippet of file content in memory
func (lg *LittleGuy) AddCodeSnippet(filePath, content string) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.codeSnapshots[filePath] = content
}

// newMethod represents a detected function in the Git diff.
type newMethod struct {
	FilePath     string
	FunctionName string
	Action       string // "added" or "removed"
}

// importExport represents detected imports or exports in the Git diff.
type importExport struct {
	FilePath  string
	Statement string
	Action    string // "added" or "removed"
}

// parseNewMethods extracts function definitions, imports, and exports from a Git diff.
func parseNewMethods(diff string) ([]newMethod, []importExport) {
	var addedMethods, removedMethods []newMethod
	var addedImportsExports, removedImportsExports []importExport
	lines := strings.Split(diff, "\n")

	// Regex patterns
	funcRegex := regexp.MustCompile(`^\s*(func|def|function|public|private|static|void)\s+(\w+)\s*\(`)
	importRegex := regexp.MustCompile(`^\s*(import|from|require\(|export\s+(const|function|default|class|var|let|async function))\s+([^ ]+)`)

	var currentFile string

	for _, line := range lines {
		// Detect file path from `diff --git`
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Split(line, " ")
			if len(parts) > 2 {
				currentFile = strings.TrimPrefix(parts[2], "b/")
			}
			continue
		}

		// Process additions separately
		if strings.HasPrefix(line, "+") {
			if matches := funcRegex.FindStringSubmatch(line[1:]); len(matches) >= 3 {
				addedMethods = append(addedMethods, newMethod{
					FilePath:     currentFile,
					FunctionName: matches[2],
					Action:       "added",
				})
			} else if matches := importRegex.FindStringSubmatch(line[1:]); len(matches) >= 4 {
				addedImportsExports = append(addedImportsExports, importExport{
					FilePath:  currentFile,
					Statement: matches[0],
					Action:    "added",
				})
			}
		}

		// Process deletions separately
		if strings.HasPrefix(line, "-") {
			if matches := funcRegex.FindStringSubmatch(line[1:]); len(matches) >= 3 {
				removedMethods = append(removedMethods, newMethod{
					FilePath:     currentFile,
					FunctionName: matches[2],
					Action:       "removed",
				})
			} else if matches := importRegex.FindStringSubmatch(line[1:]); len(matches) >= 4 {
				removedImportsExports = append(removedImportsExports, importExport{
					FilePath:  currentFile,
					Statement: matches[0],
					Action:    "removed",
				})
			}
		}
	}

	// Combine added and removed separately
	allMethods := append(addedMethods, removedMethods...)
	allImportsExports := append(addedImportsExports, removedImportsExports...)

	return allMethods, allImportsExports
}

// runGitDiff executes `git diff --unified=0` and returns the output
func runGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--unified=0")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
