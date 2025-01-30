// internal/dce/littleguy.go

package dce

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/coreutils"
)

// LittleGuy tracks the ephemeral codebase snapshot and tasks for a single DCE session.
type LittleGuy struct {
	mutex          sync.RWMutex
	conversationID string
	tasks          []contextpkg.Task // Ongoing tasks
	completed      []contextpkg.Task // Completed tasks
	codeSnapshots  map[string]string // filePath -> file content
	pollInterval   time.Duration     // how often to check diffs
	monitorStarted bool              // track background goroutine status
}

// NewLittleGuy initializes an in-memory ephemeral “LittleGuy” object.
func NewLittleGuy(conversationID string, initialTasks []contextpkg.Task) *LittleGuy {
	return &LittleGuy{
		conversationID: conversationID,
		tasks:          initialTasks,
		completed:      make([]contextpkg.Task, 0),
		codeSnapshots:  make(map[string]string),
		pollInterval:   10 * time.Second, // default poll interval
	}
}

// StartMonitoring launches a background goroutine that periodically calls UpdateFromDiff.
// This keeps LittleGuy aware of new or removed functions, imports, etc.
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
				color.Red("[LittleGuy] Failed to run git diff: %v\n", err)
				continue
			}
			if diffOutput != "" {
				lg.UpdateFromDiff(diffOutput)
			}
		}
	}()
}

// MonitorInput is an example method that analyzes arbitrary user input
// for references to function names, files, etc. Then it silently updates tasks.
func (lg *LittleGuy) MonitorInput(input string) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	lines := strings.Split(input, "\n")

	// Simple patterns for demonstration:
	funcRegex := regexp.MustCompile(`(?i)(func|def|function|public|private|static|void)\s+([A-Za-z0-9_]+)\s*\(`)
	fileRegex := regexp.MustCompile(`[A-Za-z0-9_\-/]+\.(go|js|ts|py|rb|java|cs)`)

	for _, line := range lines {
		if match := funcRegex.FindStringSubmatch(line); len(match) >= 3 {
			funcName := match[2]
			// Add a new "review" task for discovered function
			lg.tasks = append(lg.tasks, contextpkg.Task{
				Description: fmt.Sprintf("Detected function: %s", funcName),
				Functions:   []string{funcName},
				Notes: []string{
					"Consider testing and documenting this function",
				},
			})
		}
		if fileMatch := fileRegex.FindAllString(line, -1); len(fileMatch) > 0 {
			for _, fileRef := range fileMatch {
				// Add a new "review" task for discovered file reference
				lg.tasks = append(lg.tasks, contextpkg.Task{
					Description: fmt.Sprintf("Detected file reference: %s", fileRef),
					Files:       []string{fileRef},
					Notes: []string{
						"Consider adding to relevant code snapshots or tasks",
					},
				})
			}
		}
	}

	messages := lg.BuildEphemeralContext("") // Generate the LLM context
	lg.logLLMContext(messages)               // Log the exact LLM context
}

// UpdateFromDiff parses the current Git diff and silently updates tasks
// based on newly added or removed methods and imports.
func (lg *LittleGuy) UpdateFromDiff(diff string) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	newFuncs, importExports := parseNewMethods(diff)

	// Process new or removed functions
	for _, nf := range newFuncs {
		if nf.Action == "added" {
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
			lg.markTaskAsCompleted(nf.FunctionName)
		}
	}

	// Process import/export changes
	for _, impExp := range importExports {
		if impExp.Action == "added" {
			lg.tasks = append(lg.tasks, contextpkg.Task{
				Description: fmt.Sprintf("New import detected: %s", impExp.Statement),
				Files:       []string{impExp.FilePath},
				Notes:       []string{"Review dependency impact and update documentation"},
			})
		} else if impExp.Action == "removed" {
			lg.tasks = append(lg.tasks, contextpkg.Task{
				Description: fmt.Sprintf("Import removed: %s", impExp.Statement),
				Files:       []string{impExp.FilePath},
				Notes:       []string{"Check for orphaned references and clean up"},
			})
		}
	}

	messages := lg.BuildEphemeralContext("") // Generate the updated LLM context
	lg.logLLMContext(messages)               // Log the exact LLM context

}

// markTaskAsCompleted moves tasks referencing funcName to the completed list.
func (lg *LittleGuy) markTaskAsCompleted(funcName string) {
	for i, task := range lg.tasks {
		if contains(task.Functions, funcName) {
			lg.completed = append(lg.completed, task)
			lg.tasks = append(lg.tasks[:i], lg.tasks[i+1:]...)
			break
		}
	}
}

// BuildEphemeralContext aggregates tasks, code snapshots, and user input
// into a final LLM context. Typically invoked before calling the LLM.
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
		for i, t := range lg.tasks {
			builder.WriteString(fmt.Sprintf("Task %d: %s\n", i+1, t.Description))
			if len(t.Notes) > 0 {
				builder.WriteString(fmt.Sprintf("Notes: %v\n", t.Notes))
			}
			if len(t.Files) > 0 {
				builder.WriteString(fmt.Sprintf("Files: %v\n", t.Files))
			}
			if len(t.Functions) > 0 {
				builder.WriteString(fmt.Sprintf("Functions: %v\n", t.Functions))
			}
			builder.WriteString("\n")
		}
		messages = append(messages, contextpkg.Message{
			Role:    "system",
			Content: builder.String(),
		})
	}

	// 3. Show completed tasks (if applicable)
	if len(lg.completed) > 0 {
		var builder strings.Builder
		for i, t := range lg.completed {
			builder.WriteString(fmt.Sprintf("Completed Task %d: %s\n", i+1, t.Description))
		}
		messages = append(messages, contextpkg.Message{
			Role:    "system",
			Content: builder.String(),
		})
	}

	// 4. Code snapshots
	if len(lg.codeSnapshots) > 0 {
		builder := strings.Builder{}
		for path, content := range lg.codeSnapshots {
			builder.WriteString(fmt.Sprintf("File: %s\n---\n%s\n---\n\n", path, content))
		}
		messages = append(messages, contextpkg.Message{
			Role:    "system",
			Content: builder.String(),
		})
	}

	// 5. User's input
	messages = append(messages, contextpkg.Message{
		Role:    "user",
		Content: userQuery,
	})

	// **NEW: Log the exact context being passed to the LLM**
	lg.logLLMContext(messages)

	return messages
}

// AddCodeSnippet stores a snippet of file content in memory.
func (lg *LittleGuy) AddCodeSnippet(filePath, content string) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	lg.codeSnapshots[filePath] = content
}

// UpdateTaskList appends new tasks to the existing in-memory list.
func (lg *LittleGuy) UpdateTaskList(newTasks []contextpkg.Task) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	lg.tasks = append(lg.tasks, newTasks...)
}

// logLLMContext writes the exact raw LLM input to the log file (no formatting).
func (lg *LittleGuy) logLLMContext(messages []contextpkg.Message) {
	var rawContext strings.Builder
	for _, msg := range messages {
		rawContext.WriteString(fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content))
	}

	// Write the exact unformatted LLM context to littleguy-<conversationID>.txt
	if err := coreutils.LogLittleGuyContext(lg.conversationID, rawContext.String()); err != nil {
		color.Red("[LittleGuy] Failed to log LLM context: %v\n", err)
	}
}

// parseNewMethods extracts function definitions, imports, and exports from a Git diff.
func parseNewMethods(diff string) ([]newMethod, []importExport) {
	var addedMethods, removedMethods []newMethod
	var addedImportsExports, removedImportsExports []importExport

	lines := strings.Split(diff, "\n")

	// Regex patterns for demonstration
	funcRegex := regexp.MustCompile(`^\s*(func|def|function|public|private|static|void)\s+([A-Za-z0-9_]+)\s*\(`)
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

		if strings.HasPrefix(line, "+") {
			// Check for newly added functions
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

		if strings.HasPrefix(line, "-") {
			// Check for removed functions
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

	allMethods := append(addedMethods, removedMethods...)
	allImportsExports := append(addedImportsExports, removedImportsExports...)
	return allMethods, allImportsExports
}

// runGitDiff uses coreutils to run a unified diff command.
func runGitDiff() (string, error) {
	return coreutils.ExecGit("diff", "--unified=0")
}

// Helper types for parseNewMethods():
type newMethod struct {
	FilePath     string
	FunctionName string
	Action       string // "added" or "removed"
}

type importExport struct {
	FilePath  string
	Statement string
	Action    string // "added" or "removed"
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
