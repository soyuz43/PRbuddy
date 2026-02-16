// internal/dce/context_manager.go
package dce

import (
	"sync"
)

// DCEContextManager handles the association between conversations and DCE contexts
// This breaks the import cycle between contextpkg and dce
type DCEContextManager struct {
	contexts map[string]*LittleGuy
	mutex    sync.RWMutex
}

var (
	// Global instance of the DCE context manager
	contextManagerInstance *DCEContextManager
	contextManagerOnce     sync.Once
)

// GetDCEContextManager returns the singleton instance of DCEContextManager
func GetDCEContextManager() *DCEContextManager {
	contextManagerOnce.Do(func() {
		contextManagerInstance = &DCEContextManager{
			contexts: make(map[string]*LittleGuy),
		}
	})
	return contextManagerInstance
}

// AddContext associates a LittleGuy instance with a conversation ID
func (cm *DCEContextManager) AddContext(conversationID string, littleguy *LittleGuy) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.contexts[conversationID] = littleguy
}

// GetContext retrieves the LittleGuy instance for a conversation ID
func (cm *DCEContextManager) GetContext(conversationID string) (*LittleGuy, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	littleguy, exists := cm.contexts[conversationID]
	return littleguy, exists
}

// RemoveContext removes the LittleGuy instance for a conversation ID
func (cm *DCEContextManager) RemoveContext(conversationID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.contexts, conversationID)
}
