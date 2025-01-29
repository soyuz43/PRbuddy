package contextpkg

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Message represents a chat message for LLM interactions
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
				Content: fmt.Sprintf("Initial code changes (truncated):\n%s", truncateDiff(c.InitialDiff)),
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

// truncateDiff reduces the size of large diffs while preserving important context
func truncateDiff(diff string) string {
	const maxLines = 100
	lines := splitLines(diff)
	if len(lines) <= maxLines {
		return diff
	}

	// Keep first and last 50 lines
	start := lines[:50]
	end := lines[len(lines)-50:]
	return joinLines(append(start, end...))
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
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

func BuildEphemeralContext(input string) []Message {
	return []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: input},
	}
}
