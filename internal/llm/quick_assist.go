// internal/llm/quick_assist.go

package llm

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func QuickAssistHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	response, err := HandleQuickAssistMessage(request.Query)
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

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			return
		}

		next.ServeHTTP(w, r)
	}
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
		http.HandleFunc("/quick-assist", enableCORS(QuickAssistHandler))
		http.HandleFunc("/save-draft", enableCORS(SaveDraftHandler))
		http.HandleFunc("/load-draft", enableCORS(LoadDraftHandler))
		http.HandleFunc("/what", enableCORS(WhatHandler))

		port := "7743"
		fmt.Printf("Starting PRBuddy API server on port %s...\n", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	},
}
