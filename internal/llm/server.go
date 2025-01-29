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
	// ----------------------------------------------------------------
	// 1. QuickAssist route (PERSISTENT conversation only)
	// ----------------------------------------------------------------
	router.HandleFunc("/quickassist", quickAssistHandler)

	// ----------------------------------------------------------------
	// 2. DCE route (EPHEMERAL conversation, specialized logic)
	// ----------------------------------------------------------------
	router.HandleFunc("/dce", dceHandler)

	// Endpoint to clear a conversation context (if you still want it)
	router.HandleFunc("/quickassist/clear", quickAssistClearHandler)

	// Draft context management
	router.HandleFunc("/extension/drafts", saveDraftHandler)
	router.HandleFunc("/extension/drafts/load", loadDraftHandler)

	// Summaries / 'what' functionality
	router.HandleFunc("/what", whatHandler)

	// Model management
	router.HandleFunc("/extension/models", listModelsHandler)
	router.HandleFunc("/extension/model", setModelHandler)
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

// ServeCmd is the Cobra command to start the API server
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

// ------------------------------------------------------
//     HTTP Handlers
// ------------------------------------------------------

// quickAssistHandler handles PERSISTENT conversation route:
//
//	POST /quickassist
//
// Request JSON format:
//
//	{
//	  "conversationId": "abc123", // optional; if absent => create new persistent conversation
//	  "input": "user's question"
//	}
func quickAssistHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConversationID string `json:"conversationId"`
		Input          string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	response, err := HandleQuickAssist(req.ConversationID, req.Input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error in QuickAssist: %v", err), http.StatusInternalServerError)
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

// dceHandler handles EPHEMERAL conversation route for DCE logic:
//
//	POST /dce
//
// Request JSON format:
//
//	{
//	  "conversationId": "abc123", // optional; if absent => create ephemeral conversation
//	  "input": "user input or instructions for DCE"
//	}
func dceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConversationID string `json:"conversationId"`
		Input          string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	response, err := HandleDCERequest(req.ConversationID, req.Input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error in DCERequest: %v", err), http.StatusInternalServerError)
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

// quickAssistClearHandler allows a client (e.g., an IDE extension)
// to clear a conversation from memory if needed.
// POST /quickassist/clear
// Request JSON format:
//
//	{
//	  "conversationId": "abc123"
//	}
func quickAssistClearHandler(w http.ResponseWriter, r *http.Request) {
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

// saveDraftHandler handles saving conversation/draft context to disk
// POST /extension/drafts
// Request JSON format:
//
//	{
//	  "branch": "some-branch",
//	  "commit": "some-commit",
//	  "messages": [...messages...]
//	}
func saveDraftHandler(w http.ResponseWriter, r *http.Request) {
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

// loadDraftHandler retrieves a saved conversation/draft context
// POST /extension/drafts/load
// Request JSON format:
//
//	{
//	  "branch": "some-branch",
//	  "commit": "some-commit"
//	}
func loadDraftHandler(w http.ResponseWriter, r *http.Request) {
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

// whatHandler handles summarizing "what changed"
// GET or POST /what
func whatHandler(w http.ResponseWriter, r *http.Request) {
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

// listModelsHandler returns the list of models Ollama has loaded
// GET /extension/models
func listModelsHandler(w http.ResponseWriter, r *http.Request) {
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

// setModelHandler updates the in-memory model that PRBuddy-Go will use
// POST /extension/model
// Request JSON format:
//
//	{
//	  "model": "mistral:latest"
//	}
func setModelHandler(w http.ResponseWriter, r *http.Request) {
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

	// Optionally confirm that 'body.Model' is in the list from fetchOllamaModels()
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
