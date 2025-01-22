// internal/llm/quick_assist.go

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

// StartServer starts the HTTP server with dynamic port allocation
func StartServer() error {
	// Create a listener with dynamic port selection
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Get the allocated port
	port := listener.Addr().(*net.TCPAddr).Port

	// Write the port to .git/.prbuddy_port
	if err := utils.WritePortFile(port); err != nil {
		return fmt.Errorf("failed to write port file: %w", err)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: nil, // Use http.DefaultServeMux
	}

	// Graceful shutdown handling
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	// Inactivity timeout
	inactivityTimeout := 10 * time.Minute
	timeoutTimer := time.NewTimer(inactivityTimeout)

	go func() {
		<-shutdownChan
		fmt.Println("\nShutting down server...")
		_ = server.Shutdown(context.Background())
	}()

	go func() {
		<-timeoutTimer.C
		fmt.Println("Server inactive for too long. Shutting down...")
		_ = server.Shutdown(context.Background())
	}()

	// Start the server
	fmt.Printf("Starting PRBuddy server on port %d...\n", port)
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	// Clean up on shutdown
	if err := utils.DeletePortFile(); err != nil {
		fmt.Printf("Warning: failed to delete port file: %v\n", err)
	}

	fmt.Println("Server shutdown complete.")
	return nil
}

func QuickAssistHandler(w http.ResponseWriter, r *http.Request) {
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
	json.NewEncoder(w).Encode(map[string]string{
		"response": response,
	})
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
	json.NewEncoder(w).Encode(map[string]string{
		"status": "Draft context saved successfully",
	})
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
		if err.Error() == "no commits found in the repository" {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"summary": summary,
	})
}

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start API server for VS Code extension",
	Run: func(cmd *cobra.Command, args []string) {
		// Check extension installation first
		installed, err := utils.CheckExtensionInstalled()
		if err != nil {
			fmt.Printf("Extension check failed: %v\n", err)
			os.Exit(1)
		}
		if !installed {
			fmt.Println("Extension not installed. Server not started.")
			return
		}

		// Start server with dynamic port
		if err := StartServer(); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
			os.Exit(1)
		}
	},
}
