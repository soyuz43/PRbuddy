// internal/llm/server.go

package llm

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

// Global model config in memory

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

	router := http.NewServeMux()
	registerHandlers(router)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, port),
		Handler: router,
	}

	return manageServerLifecycle(server, listener, cfg.InactivityTimeout)
}

func registerHandlers(router *http.ServeMux) {
	router.HandleFunc("/quickassist", quickAssistHandler())
	router.HandleFunc("/dce", dceHandler())
	router.HandleFunc("/quickassist/clear", quickAssistClearHandler())
	router.HandleFunc("/extension/drafts", saveDraftHandler())
	router.HandleFunc("/extension/drafts/load", loadDraftHandler())
	router.HandleFunc("/what", whatHandler())
	router.HandleFunc("/extension/models", listModelsHandler())
	router.HandleFunc("/extension/model", setModelHandler())
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

		_ = server.Shutdown(ctx)
	}()

	fmt.Printf("Server listening on %s\n", listener.Addr())
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	_ = utils.DeletePortFile()
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

// Request/Response types
type (
	QuickAssistRequest struct {
		ConversationID string `json:"conversationId"`
		Input          string `json:"input"`
	}

	DCERequest struct {
		ConversationID string `json:"conversationId"`
		Input          string `json:"input"`
	}

	ClearRequest struct {
		ConversationID string `json:"conversationId"`
	}

	DraftSaveRequest struct {
		Branch   string               `json:"branch"`
		Commit   string               `json:"commit"`
		Messages []contextpkg.Message `json:"messages"`
	}

	DraftLoadRequest struct {
		Branch string `json:"branch"`
		Commit string `json:"commit"`
	}

	ModelRequest struct {
		Model string `json:"model"`
	}
)

// Handlers
func quickAssistHandler() http.HandlerFunc {
	return JSONHandler(func(req QuickAssistRequest) (any, error) {
		return HandleQuickAssist(req.ConversationID, req.Input)
	})
}

func dceHandler() http.HandlerFunc {
	return JSONHandler(func(req DCERequest) (any, error) {
		return HandleDCERequest(req.ConversationID, req.Input)
	})
}

func quickAssistClearHandler() http.HandlerFunc {
	return JSONHandler(func(req ClearRequest) (any, error) {
		if req.ConversationID == "" {
			return nil, fmt.Errorf("conversationId is required")
		}
		contextpkg.ConversationManagerInstance.RemoveConversation(req.ConversationID)
		return map[string]string{"status": "cleared"}, nil
	})
}

func saveDraftHandler() http.HandlerFunc {
	return JSONHandler(func(req DraftSaveRequest) (any, error) {
		if req.Branch == "" || req.Commit == "" {
			return nil, fmt.Errorf("branch and commit are required")
		}
		if len(req.Messages) == 0 {
			return nil, fmt.Errorf("messages are required")
		}
		if err := SaveDraftContext(req.Branch, req.Commit, req.Messages); err != nil {
			return nil, err
		}
		return map[string]string{"status": "success"}, nil
	})
}

func loadDraftHandler() http.HandlerFunc {
	return JSONHandler(func(req DraftLoadRequest) (any, error) {
		context, err := LoadDraftContext(req.Branch, req.Commit)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "success", "messages": context}, nil
	})
}

func whatHandler() http.HandlerFunc {
	return JSONHandler(func(_ struct{}) (any, error) {
		summary, err := GenerateWhatSummary()
		return map[string]string{"summary": summary}, err
	})
}

func listModelsHandler() http.HandlerFunc {
	return JSONHandler(func(_ struct{}) (any, error) {
		endpoint := os.Getenv("PRBUDDY_LLM_ENDPOINT")
		if endpoint == "" {
			endpoint = "http://localhost:11434"
		}
		return fetchOllamaModels(endpoint)
	})
}

func setModelHandler() http.HandlerFunc {
	return JSONHandler(func(req ModelRequest) (any, error) {
		if req.Model == "" {
			return nil, fmt.Errorf("missing 'model' field")
		}
		contextpkg.SetActiveModel(req.Model)
		return map[string]string{
			"status":       "model updated",
			"active_model": contextpkg.GetActiveModel(),
		}, nil
	})
}
