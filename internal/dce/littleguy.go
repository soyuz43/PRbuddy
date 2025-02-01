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
	return &LittleGuy{
		conversationID: conversationID,
		tasks:          initialTasks,
		completed:      []contextpkg.Task{},
		codeSnapshots:  make(map[string]string),
		pollInterval:   10 * time.Second,
	}
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
		// Use centralized FuncPattern from dce_helper.go.
		if matches := FuncPattern.FindStringSubmatch(line); len(matches) >= 3 {
			funcName := matches[2]
			lg.tasks = append(lg.tasks, contextpkg.Task{
				Description: fmt.Sprintf("Detected function: %s", funcName),
				Functions:   []string{funcName},
				Notes:       []string{"Consider testing and documenting this function."},
			})
		}
		// Simple heuristic for file references.
		if strings.Contains(line, ".go") || strings.Contains(line, ".js") ||
			strings.Contains(line, ".py") || strings.Contains(line, ".ts") {
			words := strings.Fields(line)
			for _, word := range words {
				if strings.Contains(word, ".go") || strings.Contains(word, ".js") ||
					strings.Contains(word, ".py") || strings.Contains(word, ".ts") {
					lg.tasks = append(lg.tasks, contextpkg.Task{
						Description: fmt.Sprintf("Detected file reference: %s", word),
						Files:       []string{word},
						Notes:       []string{"Consider adding to code snapshots or tasks."},
					})
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
			// Process added lines.
			content := trimmed[1:]
			funcs := ParseFunctionNames(content)
			for _, fn := range funcs {
				lg.tasks = append(lg.tasks, contextpkg.Task{
					Description: fmt.Sprintf("New function added: %s", fn),
					Functions:   []string{fn},
					Notes:       []string{"Update tests and documentation accordingly."},
				})
			}
		} else if strings.HasPrefix(trimmed, "-") {
			// Process removed lines.
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
	// System introduction.
	messages = append(messages, contextpkg.Message{
		Role:    "system",
		Content: "You are a helpful developer assistant. Below is the current task list and code snapshots.",
	})
	// Summarize uncompleted tasks.
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
	// Include code snapshots.
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
	// Add user query.
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

// UpdateTaskList appends new tasks to the current in-memory task list.
func (lg *LittleGuy) UpdateTaskList(newTasks []contextpkg.Task) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.tasks = append(lg.tasks, newTasks...)
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
