package mocks

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// MockLumerinNodeAPI represents a mock server for the Lumerin Node API
type MockLumerinNodeAPI struct {
	server *http.Server
}

// Start starts the mock server on the specified port
func (m *MockLumerinNodeAPI) Start(port int) error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// Models endpoint
	mux.HandleFunc("/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		models := map[string]interface{}{
			"models": []map[string]interface{}{
				{
					"id":          1,
					"name":        "LMR-Hermes-2-Theta-Llama-3-8B",
					"endpoint":    fmt.Sprintf("http://localhost:%d", port),
					"description": "Test model",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models)
	})

	// Sessions endpoint
	mux.HandleFunc("/api/v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Duration string `json:"duration"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate credentials
		if req.Username != "admin" || req.Password != "test-auth-token-12345" {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Create session response
		session := map[string]interface{}{
			"sessionToken": "test-session-token",
			"expiresAt":   time.Now().Add(time.Hour).Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)
	})

	m.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Printf("Starting mock Lumerin Node API on port %d", port)
	return m.server.ListenAndServe()
}

// Stop gracefully shuts down the mock server
func (m *MockLumerinNodeAPI) Stop() error {
	if m.server != nil {
		return m.server.Close()
	}
	return nil
}

// Helper function to validate Authorization header
func validateAuth(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	return strings.HasPrefix(auth, "Bearer ") && strings.TrimPrefix(auth, "Bearer ") != ""
} 