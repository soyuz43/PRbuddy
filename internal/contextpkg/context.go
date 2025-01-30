package contextpkg

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/soyuz43/prbuddy-go/internal/coreutils"
)

// Message represents a chat message for LLM interactions
type Message struct {
	Role      string        `json:"role"`                 // "user", "assistant", "system"
	Content   string        `json:"content,omitempty"`    // The main text content
	Images    []string      `json:"images,omitempty"`     // Optional: image paths for multimodal models
	ToolCalls []interface{} `json:"tool_calls,omitempty"` // Optional: tool calls (if applicable)
}

// Task represents a unit of work detected by the DCE
type Task struct {
	Description  string   `json:"description"`
	Files        []string `json:"files"`
	Functions    []string `json:"functions"`
	Dependencies []string `json:"dependencies"`
	Notes        []string `json:"notes"`
}

// Conversation represents a single developer's conversation thread
type Conversation struct {
	ID             string
	Ephemeral      bool
	InitialDiff    string
	Messages       []Message
	LastActivity   time.Time
	DiffTruncation bool
	mutex          sync.RWMutex
}

// ConversationManager manages all conversations
type ConversationManager struct {
	conversations map[string]*Conversation
	mutex         sync.RWMutex
}

// NewConversationManager creates a new ConversationManager
func NewConversationManager() *ConversationManager {
	return &ConversationManager{
		conversations: make(map[string]*Conversation),
	}
}

// StartConversation initializes a new conversation
func (cm *ConversationManager) StartConversation(id, initialDiff string, ephemeral bool) *Conversation {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	conv := &Conversation{
		ID:           id,
		Ephemeral:    ephemeral,
		InitialDiff:  initialDiff,
		Messages:     make([]Message, 0),
		LastActivity: time.Now(),
	}

	cm.conversations[id] = conv
	return conv
}

// GetConversation retrieves an existing conversation
func (cm *ConversationManager) GetConversation(id string) (*Conversation, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	conv, exists := cm.conversations[id]
	return conv, exists
}

// RemoveConversation removes a conversation from memory
func (cm *ConversationManager) RemoveConversation(id string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.conversations, id)
}

// Cleanup removes stale conversations
func (cm *ConversationManager) Cleanup(maxAge time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	for id, conv := range cm.conversations {
		if now.Sub(conv.LastActivity) > maxAge {
			delete(cm.conversations, id)
		}
	}
}

// AddMessage adds a new message to the conversation (thread-safe)
func (c *Conversation) AddMessage(role, content string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Messages = append(c.Messages, Message{
		Role:    role,
		Content: content,
	})
	c.LastActivity = time.Now()
}

// BuildContext constructs the conversation context with proper diff management (thread-safe)
func (c *Conversation) BuildContext() []Message {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	context := make([]Message, 0, len(c.Messages)+2)

	// Add system message
	context = append(context, Message{
		Role:    "system",
		Content: "You are a developer assistant designed to use the context below to make informed and relevant insights based on the nature of the question and the context below.",
	})

	// Add initial diff if applicable
	if c.InitialDiff != "" {
		if len(c.Messages) < 4 || !c.DiffTruncation {
			context = append(context, Message{
				Role:    "user",
				Content: fmt.Sprintf("Initial code changes:\n%s", c.InitialDiff),
			})
		} else {
			context = append(context, Message{
				Role:    "user",
				Content: fmt.Sprintf("Initial code changes (truncated):\n%s", TruncateDiff(c.InitialDiff, 1000)),
			})
			c.DiffTruncation = true
		}
	}

	// Add conversation messages
	context = append(context, c.Messages...)
	return context
}

// SetMessages replaces all messages in the conversation (thread-safe)
func (c *Conversation) SetMessages(messages []Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Messages = messages
}

// truncateDiff intelligently reduces the diff size while preserving key info.
func TruncateDiff(diff string, maxLines int) string {
	lines := coreutils.SplitLines(diff)
	if len(lines) <= maxLines {
		return diff
	}

	var truncated []string
	var currentFile string
	var addedLines []string
	var removedCount int

	for _, line := range lines {
		// Detect file changes (e.g., `diff --git a/path/to/file b/path/to/file`)
		if strings.HasPrefix(line, "diff --git") {
			// Store previous file's changes if we hit a new file
			if currentFile != "" {
				truncated = append(truncated, summarizeFileChanges(currentFile, addedLines, removedCount))
				addedLines = nil
				removedCount = 0
			}
			currentFile = extractFilePath(line)
			truncated = append(truncated, line)
		} else if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			// Keep file modification headers
			truncated = append(truncated, line)
		} else if strings.HasPrefix(line, "new file mode") || strings.HasPrefix(line, "deleted file mode") {
			// Keep metadata for file creation/deletion
			truncated = append(truncated, line)
		} else if strings.HasPrefix(line, "+") {
			// Store added lines (prioritize these)
			addedLines = append(addedLines, line)
		} else if strings.HasPrefix(line, "-") {
			// Count removed lines, do not keep them
			removedCount++
		} else {
			// Keep general metadata (e.g., `@@ -12,5 +12,8 @@`)
			truncated = append(truncated, line)
		}

		// Stop adding new lines if we exceed maxLines
		if len(truncated)+len(addedLines) > maxLines {
			break
		}
	}

	// Add last file's summary if needed
	if currentFile != "" {
		truncated = append(truncated, summarizeFileChanges(currentFile, addedLines, removedCount))
	}

	return coreutils.JoinLines(truncated)
}

// summarizeFileChanges generates a summary of a file's modifications.
func summarizeFileChanges(filePath string, addedLines []string, removedCount int) string {
	var summary []string
	summary = append(summary, "### Summary for "+filePath)

	// Include some added lines
	if len(addedLines) > 5 {
		summary = append(summary, addedLines[:5]...)
	} else {
		summary = append(summary, addedLines...)
	}

	// Mention removed lines without including them
	if removedCount > 0 {
		summary = append(summary, fmt.Sprintf("... [%d lines removed] ...", removedCount))
	}

	return coreutils.JoinLines(summary)
}

// extractFilePath extracts the file path from a `diff --git` line.
func extractFilePath(line string) string {
	parts := strings.Split(line, " ")
	if len(parts) < 3 {
		return "unknown_file"
	}
	return strings.TrimPrefix(parts[2], "b/")
}

// AddTask adds a new task to the conversation (thread-safe)
func (cm *ConversationManager) AddTask(conversationID string, task Task) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	conv, exists := cm.conversations[conversationID]
	if !exists {
		return fmt.Errorf("conversation %s not found", conversationID)
	}

	// Add task details as a user message
	taskContent := fmt.Sprintf("Task: %s\nFiles: %s\nFunctions: %s\nDependencies: %s\nNotes: %s",
		task.Description,
		strings.Join(task.Files, ", "),
		strings.Join(task.Functions, ", "),
		strings.Join(task.Dependencies, ", "),
		strings.Join(task.Notes, "; "),
	)
	conv.AddMessage("user", taskContent)

	return nil
}

// GetTasks retrieves all tasks from the conversation (thread-safe)
func (cm *ConversationManager) GetTasks(conversationID string) ([]Task, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	conv, exists := cm.conversations[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation %s not found", conversationID)
	}

	var tasks []Task
	for _, msg := range conv.Messages {
		if strings.HasPrefix(msg.Content, "Task: ") {
			task, err := parseTaskMessage(msg.Content)
			if err == nil {
				tasks = append(tasks, task)
			}
		}
	}

	return tasks, nil
}

func parseTaskMessage(content string) (Task, error) {
	var task Task
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Task: ") {
			task.Description = strings.TrimPrefix(line, "Task: ")
		} else if strings.HasPrefix(line, "Files: ") {
			files := strings.TrimPrefix(line, "Files: ")
			task.Files = strings.Split(files, ", ")
		} else if strings.HasPrefix(line, "Functions: ") {
			funcs := strings.TrimPrefix(line, "Functions: ")
			task.Functions = strings.Split(funcs, ", ")
		} else if strings.HasPrefix(line, "Dependencies: ") {
			deps := strings.TrimPrefix(line, "Dependencies: ")
			task.Dependencies = strings.Split(deps, ", ")
		} else if strings.HasPrefix(line, "Notes: ") {
			notes := strings.TrimPrefix(line, "Notes: ")
			task.Notes = strings.Split(notes, "; ")
		}
	}
	return task, nil
}

// ConversationManagerInstance is the singleton instance of ConversationManager
var ConversationManagerInstance = NewConversationManager()

// GenerateConversationID creates a unique conversation ID
func GenerateConversationID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// GetActiveModel retrieves the active model from the context
func GetActiveModel() string {
	// Implement logic to retrieve active model if stored within context
	return ""
}

// BuildEphemeralContext returns a minimal, stateless context for ephemeral usage
func BuildEphemeralContext(input string) []Message {
	return []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: input},
	}
}
