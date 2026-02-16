// internal/dce/littleguy.go

package dce

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// LittleGuy tracks an ephemeral code snapshot and tasks for a single DCE session.
type LittleGuy struct {
	mutex          sync.RWMutex
	conversationID string
	tasks          []contextpkg.Task // Ongoing tasks
	completed      []contextpkg.Task // Completed tasks
	codeSnapshots  map[string]string // filePath -> file content
	pollInterval   time.Duration     // How often to check for diffs
	monitorStarted bool              // Tracks background monitoring status
}

// NewLittleGuy initializes a new LittleGuy instance.
func NewLittleGuy(conversationID string, initialTasks []contextpkg.Task) *LittleGuy {
	lg := &LittleGuy{
		conversationID: conversationID,
		tasks:          initialTasks,
		completed:      []contextpkg.Task{},
		codeSnapshots:  make(map[string]string),
		pollInterval:   10 * time.Second,
	}

	// Add to context manager
	GetDCEContextManager().AddContext(conversationID, lg)
	return lg
}

// IsActive returns whether the DCE monitoring is active
func (lg *LittleGuy) IsActive() bool {
	lg.mutex.RLock()
	defer lg.mutex.RUnlock()
	return lg.monitorStarted
}

// StopMonitoring stops the background monitoring
func (lg *LittleGuy) StopMonitoring() {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.monitorStarted = false
}

// GetPollInterval returns the current polling interval
func (lg *LittleGuy) GetPollInterval() time.Duration {
	lg.mutex.RLock()
	defer lg.mutex.RUnlock()
	return lg.pollInterval
}

// GetConversationID returns the associated conversation ID
func (lg *LittleGuy) GetConversationID() string {
	return lg.conversationID
}

// StartMonitoring launches a background goroutine that periodically checks Git diffs.
func (lg *LittleGuy) StartMonitoring() {
	lg.mutex.Lock()
	if lg.monitorStarted {
		lg.mutex.Unlock()
		return
	}
	lg.monitorStarted = true
	lg.mutex.Unlock()

	go func() {
		for {
			lg.mutex.RLock()
			monitoring := lg.monitorStarted
			lg.mutex.RUnlock()

			if !monitoring {
				return
			}

			time.Sleep(lg.pollInterval)
			diffOutput, err := utils.ExecGit("diff", "--unified=0")
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

// MonitorInput analyzes user input for function names or file references and updates tasks.
func (lg *LittleGuy) MonitorInput(input string) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	lines := strings.Split(input, "\n")
	for _, line := range lines {
		if matches := FuncPattern.FindStringSubmatch(line); len(matches) >= 3 {
			funcName := matches[2]
			if !lg.hasTaskForFunction(funcName) {
				lg.tasks = append(lg.tasks, contextpkg.Task{
					Description: fmt.Sprintf("Detected function: %s", funcName),
					Functions:   []string{funcName},
					Notes:       []string{"Consider testing and documenting this function."},
				})
			}
		}

		if strings.Contains(line, ".go") || strings.Contains(line, ".js") ||
			strings.Contains(line, ".py") || strings.Contains(line, ".ts") {
			words := strings.Fields(line)
			for _, word := range words {
				if strings.Contains(word, ".go") || strings.Contains(word, ".js") ||
					strings.Contains(word, ".py") || strings.Contains(word, ".ts") {
					if !lg.hasTaskForFile(word) {
						lg.tasks = append(lg.tasks, contextpkg.Task{
							Description: fmt.Sprintf("Detected file reference: %s", word),
							Files:       []string{word},
							Notes:       []string{"Consider adding to code snapshots or tasks."},
						})
					}
				}
			}
		}
	}
	messages := lg.BuildEphemeralContext("")
	lg.logLLMContext(messages)
}

// UpdateFromDiff parses Git diff output and updates tasks accordingly.
func (lg *LittleGuy) UpdateFromDiff(diff string) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if strings.HasPrefix(trimmed, "+") {
			content := trimmed[1:]
			funcs := ParseFunctionNames(content)
			for _, fn := range funcs {
				if !lg.hasTaskForFunction(fn) {
					lg.tasks = append(lg.tasks, contextpkg.Task{
						Description: fmt.Sprintf("New function added: %s", fn),
						Functions:   []string{fn},
						Notes:       []string{"Update tests and documentation accordingly."},
					})
				}
			}
		} else if strings.HasPrefix(trimmed, "-") {
			content := trimmed[1:]
			funcs := ParseFunctionNames(content)
			for _, fn := range funcs {
				lg.markTaskAsCompleted(fn)
			}
		}
	}
	messages := lg.BuildEphemeralContext("")
	lg.logLLMContext(messages)
}

// markTaskAsCompleted moves tasks referencing a given function to the completed list.
func (lg *LittleGuy) markTaskAsCompleted(funcName string) {
	for i, task := range lg.tasks {
		for _, f := range task.Functions {
			if f == funcName {
				lg.completed = append(lg.completed, task)
				lg.tasks = append(lg.tasks[:i], lg.tasks[i+1:]...)
				return
			}
		}
	}
}

// BuildEphemeralContext aggregates tasks, code snapshots, and user input into the LLM context.
func (lg *LittleGuy) BuildEphemeralContext(userQuery string) []contextpkg.Message {
	lg.mutex.RLock()
	defer lg.mutex.RUnlock()

	var messages []contextpkg.Message
	messages = append(messages, contextpkg.Message{
		Role:    "system",
		Content: "You are a helpful developer assistant. Below is the current task list and code snapshots.",
	})

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

	if len(lg.codeSnapshots) > 0 {
		var builder strings.Builder
		for path, content := range lg.codeSnapshots {
			builder.WriteString(fmt.Sprintf("File: %s\n---\n%s\n---\n\n", path, content))
		}
		messages = append(messages, contextpkg.Message{
			Role:    "system",
			Content: builder.String(),
		})
	}

	messages = append(messages, contextpkg.Message{
		Role:    "user",
		Content: userQuery,
	})
	return messages
}

// AddCodeSnippet stores a snippet of file content.
func (lg *LittleGuy) AddCodeSnippet(filePath, content string) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.codeSnapshots[filePath] = content
}

// UpdateTaskList appends new tasks if they're not already represented.
func (lg *LittleGuy) UpdateTaskList(newTasks []contextpkg.Task) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	for _, t := range newTasks {
		duplicate := false
		for _, existing := range lg.tasks {
			if t.Description == existing.Description {
				duplicate = true
				break
			}
		}
		if !duplicate {
			lg.tasks = append(lg.tasks, t)
		}
	}
}

// logLLMContext writes the raw LLM input to a log file using utils.LogLittleGuyContext.
func (lg *LittleGuy) logLLMContext(messages []contextpkg.Message) {
	var rawContext strings.Builder
	for _, msg := range messages {
		rawContext.WriteString(fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content))
	}
	if err := utils.LogLittleGuyContext(lg.conversationID, rawContext.String()); err != nil {
		color.Red("[LittleGuy] Failed to log LLM context: %v\n", err)
	}
}

// hasTaskForFile returns true if any task already includes the file.
func (lg *LittleGuy) hasTaskForFile(file string) bool {
	for _, task := range lg.tasks {
		for _, f := range task.Files {
			if f == file {
				return true
			}
		}
	}
	return false
}

// hasTaskForFunction returns true if any task already includes the function.
func (lg *LittleGuy) hasTaskForFunction(fn string) bool {
	for _, task := range lg.tasks {
		for _, f := range task.Functions {
			if f == fn {
				return true
			}
		}
	}
	return false
}
