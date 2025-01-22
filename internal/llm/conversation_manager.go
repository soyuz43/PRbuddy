// internal/llm/conversation_manager.go

package llm

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ConversationManager handles stateful conversations with context management
type ConversationManager struct {
	conversations map[string]*Conversation
	mutex         sync.RWMutex
}

// Conversation represents a single developer's conversation thread
type Conversation struct {
	ID             string
	Ephemeral      bool
	InitialDiff    string
	Messages       []Message
	LastActivity   time.Time
	DiffTruncation bool
}

// NewConversationManager creates a new conversation manager
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

// RemoveConversation removes a conversation from memory (useful for ephemeral clearing)
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

// AddMessage adds a new message to the conversation
func (c *Conversation) AddMessage(role, content string) {
	c.Messages = append(c.Messages, Message{
		Role:    role,
		Content: content,
	})
	c.LastActivity = time.Now()
}

// BuildContext constructs the conversation context with proper diff management
func (c *Conversation) BuildContext() []Message {
	context := make([]Message, 0, len(c.Messages)+2)

	// If this is ephemeral and no initial diff is relevant, you could skip
	// or heavily simplify. But for demonstration, we apply the same logic.
	// Add system message
	context = append(context, Message{
		Role:    "system",
		Content: "You are a helpful assistant for creating and refining pull requests.",
	})

	// Add initial diff if within first two turns or not truncated
	if c.InitialDiff != "" {
		if len(c.Messages) < 4 || !c.DiffTruncation {
			context = append(context, Message{
				Role:    "user",
				Content: fmt.Sprintf("Initial code changes:\n%s", c.InitialDiff),
			})
		} else {
			// Add truncated diff after two turns
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

// truncateDiff reduces the size of large diffs while preserving important context
func truncateDiff(diff string) string {
	const maxLines = 100
	lines := strings.Split(diff, "\n")
	if len(lines) <= maxLines {
		return diff
	}

	// Keep first and last 50 lines
	start := lines[:50]
	end := lines[len(lines)-50:]
	return strings.Join(append(start, end...), "\n")
}
