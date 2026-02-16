// internal/contextpkg/context.go
package contextpkg

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// -----------------------------------------------------------------------------
// Chat Structures and LLM Context
// -----------------------------------------------------------------------------

// Message represents a chat message.
type Message struct {
	Role      string        `json:"role"`                 // e.g., "user", "assistant", "system"
	Content   string        `json:"content,omitempty"`    // The main text content
	Images    []string      `json:"images,omitempty"`     // Optional: image paths for multimodal models
	ToolCalls []interface{} `json:"tool_calls,omitempty"` // Optional: tool calls (if applicable)
}

// Task represents a unit of work.
type Task struct {
	Description  string   `json:"description"`
	Files        []string `json:"files"`
	Functions    []string `json:"functions"`
	Dependencies []string `json:"dependencies"`
	Notes        []string `json:"notes"`
}

// Conversation represents a single conversation thread.
type Conversation struct {
	ID             string
	Ephemeral      bool
	InitialDiff    string
	Messages       []Message
	Tasks          []Task
	LastActivity   time.Time
	DiffTruncation bool
	mutex          sync.RWMutex
	// Removed DCEContext *dce.LittleGuy to break import cycle
	IsActiveDCE bool // Track if DCE is active for this conversation
}

// ConversationManager manages multiple conversations.
type ConversationManager struct {
	conversations map[string]*Conversation
	mutex         sync.RWMutex
}

// NewConversationManager creates and returns a new ConversationManager.
func NewConversationManager() *ConversationManager {
	return &ConversationManager{
		conversations: make(map[string]*Conversation),
	}
}

// StartConversation creates a new conversation with the given id, initial diff, and ephemeral flag.
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

// GetConversation retrieves an existing conversation by id.
func (cm *ConversationManager) GetConversation(id string) (*Conversation, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	conv, exists := cm.conversations[id]
	return conv, exists
}

// RemoveConversation removes a conversation from memory.
func (cm *ConversationManager) RemoveConversation(id string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.conversations, id)
}

// Cleanup removes conversations that have been inactive for longer than maxAge.
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

// AddMessage appends a new message to the conversation.
func (c *Conversation) AddMessage(role, content string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Messages = append(c.Messages, Message{
		Role:    role,
		Content: content,
	})
	c.LastActivity = time.Now()
}

// BuildContext constructs the conversation context to be sent to the LLM.
func (c *Conversation) BuildContext() []Message {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	context := []Message{
		{
			Role:    "system",
			Content: "You are a developer assistant.",
		},
	}
	context = append(context, c.Messages...)
	return context
}

// SetMessages replaces the conversation's messages with the provided slice.
func (c *Conversation) SetMessages(newMessages []Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Messages = newMessages
	c.LastActivity = time.Now()
}

// GenerateConversationID creates a unique conversation ID using the given prefix.
func GenerateConversationID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// TruncateDiff reduces the diff size to at most maxLines while preserving key information.
func TruncateDiff(diff string, maxLines int) string {
	lines := strings.Split(strings.TrimSuffix(diff, "\n"), "\n")
	if len(lines) <= maxLines {
		return diff
	}
	return strings.Join(lines[:maxLines], "\n")
}

// ConversationManagerInstance is a global singleton instance of ConversationManager.
var ConversationManagerInstance = NewConversationManager()

// -----------------------------------------------------------------------------
// Global LLM Model State
// -----------------------------------------------------------------------------

var (
	modelMutex     sync.RWMutex
	activeLLMModel string
)

// SetActiveModel sets the currently active model name (thread-safe).
func SetActiveModel(model string) {
	modelMutex.Lock()
	defer modelMutex.Unlock()
	activeLLMModel = model
}

// GetActiveModel retrieves the current model name (thread-safe).
func GetActiveModel() string {
	modelMutex.RLock()
	defer modelMutex.RUnlock()
	return activeLLMModel
}
