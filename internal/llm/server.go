// internal/llm/server.go

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soyuz43/prbuddy-go/internal/utils"
	"github.com/spf13/cobra"
)

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
	listener, err := net.Listen("tcp", cfg.Host+":0")
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Write port file before starting server
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
	router.HandleFunc("/extension/quick-assist", QuickAssistHandler)
	router.HandleFunc("/extension/drafts", SaveDraftHandler)
	router.HandleFunc("/extension/drafts/load", LoadDraftHandler)
	router.HandleFunc("/what", WhatHandler)
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

// Handlers implementation
func QuickAssistHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	response, err := HandleExtensionQuickAssist(request.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": response})
}

func SaveDraftHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Branch   string    `json:"branch"`
		Commit   string    `json:"commit"`
		Messages []Message `json:"messages"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if err := SaveDraftContext(request.Branch, request.Commit, request.Messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func LoadDraftHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Branch string `json:"branch"`
		Commit string `json:"commit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	context, err := LoadDraftContext(request.Branch, request.Commit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"messages": context,
	})
}

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"summary": summary})
}

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
