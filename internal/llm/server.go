// internal/llm/server.go

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

// -------------------------------------------
// Global model config in memory
// -------------------------------------------
var (
	modelMutex     sync.RWMutex
	activeLLMModel string
)

// fetchOllamaModels queries Ollama at /api/ps to list currently loaded models.
func fetchOllamaModels() ([]map[string]interface{}, error) {
	// Hard-coded to the default endpoint (http://localhost:11434)
	resp, err := http.Get("http://localhost:11434/api/ps")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama /api/ps: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama /api/ps returned status %d", resp.StatusCode)
	}

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Ollama /api/ps response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Ollama /api/ps: %w", err)
	}

	// The response structure is:
	// {
	//   "models": [
	//       { "name": "mistral:latest", "model": "mistral:latest", ...},
	//       ...
	//   ]
	// }
	modelsRaw, ok := result["models"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected /api/ps JSON format (no 'models' array)")
	}

	var models []map[string]interface{}
	for _, item := range modelsRaw {
		if m, valid := item.(map[string]interface{}); valid {
			models = append(models, m)
		}
	}
	return models, nil
}

// setActiveModel updates the in-memory "activeLLMModel"
func setActiveModel(model string) {
	modelMutex.Lock()
	defer modelMutex.Unlock()
	activeLLMModel = model
}

// getActiveModel reads the in-memory "activeLLMModel"
func getActiveModel() string {
	modelMutex.RLock()
	defer modelMutex.RUnlock()
	return activeLLMModel
}

const (
	defaultHost              = "localhost"
	defaultInactivityTimeout = 30 * time.Minute
	shutdownGracePeriod      = 5 * time.Second
)

type ServerConfig struct {
	Host              string
	InactivityTimeout time.Duration
}

// StartServer initializes and runs the HTTP server with full lifecycle management
func StartServer(cfg ServerConfig) error {
	// Ensure cache directory exists first
	if err := utils.EnsureAppCacheDir(); err != nil {
		return fmt.Errorf("cache directory initialization failed: %w", err)
	}

	listener, err := net.Listen("tcp", cfg.Host+":0")
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	if err := utils.WritePortFile(port); err != nil {
		return fmt.Errorf("port file write failed: %w", err)
	}

	// Configure server with all handlers
	router := http.NewServeMux()
	registerHandlers(router)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, port),
		Handler: router,
	}

	// Manage server lifecycle
	return manageServerLifecycle(server, listener, cfg.InactivityTimeout)
}

func registerHandlers(router *http.ServeMux) {
	// Quick Assist (ephemeral or short-lived) conversation
	router.HandleFunc("/extension/quick-assist", QuickAssistHandler)

	// Endpoint to clear ephemeral conversation context
	router.HandleFunc("/extension/quick-assist/clear", QuickAssistClearHandler)

	// Draft context management
	router.HandleFunc("/extension/drafts", SaveDraftHandler)
	router.HandleFunc("/extension/drafts/load", LoadDraftHandler)

	// Summaries / 'what' functionality
	router.HandleFunc("/what", WhatHandler)

	// -------------------------
	//  NEW: Model Management
	// -------------------------
	router.HandleFunc("/extension/models", ListModelsHandler)
	router.HandleFunc("/extension/model", SetModelHandler)
}

func manageServerLifecycle(server *http.Server, listener net.Listener, timeout time.Duration) error {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	go func() {
		select {
		case <-shutdownChan:
			fmt.Println("\nReceived shutdown signal")
		case <-timeoutTimer.C:
			fmt.Println("Inactivity timeout reached")
		}

		ctx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			fmt.Printf("Graceful shutdown failed: %v\n", err)
		}
	}()

	fmt.Printf("Server listening on %s\n", listener.Addr())
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	// Cleanup resources
	if err := utils.DeletePortFile(); err != nil {
		fmt.Printf("Port file cleanup error: %v\n", err)
	}

	fmt.Println("Server shutdown completed successfully")
	return nil
}

// -------------------
//    HTTP Handlers
// -------------------

// QuickAssistHandler handles ephemeral or short-lived user queries
// POST /extension/quick-assist
// Request JSON format:
//
//	{
//	  "conversationId": "abc123",  optional; if absent, a new ephemeral conversation is created
//	  "message": "user's question",
//	  "ephemeral": true
//	}
func QuickAssistHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConversationID string `json:"conversationId"`
		Message        string `json:"message"`
		Ephemeral      bool   `json:"ephemeral"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	response, err := HandleExtensionQuickAssist(req.ConversationID, req.Message, req.Ephemeral)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseMap := map[string]string{"response": response}
	jsonResponse, err := utils.MarshalJSON(responseMap)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonResponse))
}

// QuickAssistClearHandler allows a client (e.g., VSCode extension)
// to clear an ephemeral conversation from memory
// POST /extension/quick-assist/clear
// Request JSON format:
//
//	{
//	  "conversationId": "abc123"
//	}
func QuickAssistClearHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConversationID string `json:"conversationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ConversationID == "" {
		http.Error(w, "conversationId is required", http.StatusBadRequest)
		return
	}

	// Use contextpkg.ConversationManagerInstance to remove the conversation
	contextpkg.ConversationManagerInstance.RemoveConversation(req.ConversationID)

	responseMap := map[string]string{"status": "cleared"}
	jsonResponse, err := utils.MarshalJSON(responseMap)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonResponse))
}

// SaveDraftHandler handles saving a conversation/draft context to disk
// POST /extension/drafts
func SaveDraftHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Branch   string               `json:"branch"`
		Commit   string               `json:"commit"`
		Messages []contextpkg.Message `json:"messages"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if request.Branch == "" || request.Commit == "" {
		http.Error(w, "branch and commit are required", http.StatusBadRequest)
		return
	}

	if len(request.Messages) == 0 {
		http.Error(w, "messages are required", http.StatusBadRequest)
		return
	}

	if err := SaveDraftContext(request.Branch, request.Commit, request.Messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseMap := map[string]string{"status": "success"}
	jsonResponse, err := utils.MarshalJSON(responseMap)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonResponse))
}

// LoadDraftHandler retrieves a saved conversation/draft context from disk
// POST /extension/drafts/load
func LoadDraftHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Branch string `json:"branch"`
		Commit string `json:"commit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if request.Branch == "" || request.Commit == "" {
		http.Error(w, "branch and commit are required", http.StatusBadRequest)
		return
	}

	context, err := LoadDraftContext(request.Branch, request.Commit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	responseMap := map[string]interface{}{
		"status":   "success",
		"messages": context,
	}

	jsonResponse, err := utils.MarshalJSON(responseMap)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonResponse))
}

// WhatHandler - Summarize "what changed"
// GET or POST /what
func WhatHandler(w http.ResponseWriter, r *http.Request) {
	summary, err := GenerateWhatSummary()
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "no commits found" {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	responseMap := map[string]string{"summary": summary}
	jsonResponse, err := utils.MarshalJSON(responseMap)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonResponse))
}

// ----------------------
//   NEW: Model Endpoints
// ----------------------

// ListModelsHandler returns the list of models Ollama has loaded
// GET /extension/models
func ListModelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models, err := fetchOllamaModels()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list models: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse, err := utils.MarshalJSON(models)
	if err != nil {
		http.Error(w, "Failed to marshal models", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonResponse))
}

// SetModelHandler updates the in-memory model that PRBuddy-Go will use
// POST /extension/model
// Request JSON format:
//
//	{
//	  "model": "mistral:latest"
//	}
func SetModelHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Model == "" {
		http.Error(w, "Missing 'model' field", http.StatusBadRequest)
		return
	}

	// Optionally, you can confirm that 'body.Model' is in the list from fetchOllamaModels()
	setActiveModel(body.Model)

	responseMap := map[string]string{
		"status":       "model updated",
		"active_model": getActiveModel(),
	}
	jsonResponse, err := utils.MarshalJSON(responseMap)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonResponse))
}

// ----------------------------------
// ServeCmd for CLI usage
// ----------------------------------
var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start API server for extension integration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := ServerConfig{
			Host:              defaultHost,
			InactivityTimeout: defaultInactivityTimeout,
		}

		if err := StartServer(cfg); err != nil {
			fmt.Printf("Server startup failed: %v\n", err)
			os.Exit(1)
		}
	},
}
