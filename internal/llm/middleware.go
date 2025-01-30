// ./llm/middleware.go
package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// JSONHandler creates a handler for JSON requests/responses with unified error handling
func JSONHandler[T any](logic func(T) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set content type first
		w.Header().Set("Content-Type", "application/json")

		// Validate HTTP method
		if r.Method != http.MethodPost {
			writeError(w, fmt.Sprintf("Method %s not allowed", r.Method), http.StatusMethodNotAllowed)
			return
		}

		// Decode request
		var req T
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Execute handler logic
		response, err := logic(req)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Marshal response
		jsonResponse, err := utils.MarshalJSON(response)
		if err != nil {
			writeError(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jsonResponse))
	}
}

// writeError handles error responses consistently
func writeError(w http.ResponseWriter, message string, code int) {
	log.Printf("HTTP %d: %s", code, message)
	w.WriteHeader(code)
	errorResponse := map[string]string{"error": message}
	if jsonErr, err := utils.MarshalJSON(errorResponse); err == nil {
		w.Write([]byte(jsonErr))
	}
}
