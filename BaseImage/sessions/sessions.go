package sessions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	ConsumerNodeURL   string
	MarketplaceURL   string
	SessionDuration  string
	InternalAPIPort  string
	AuthToken       string
}

type SessionResponse struct {
	SessionToken string    `json:"sessionId"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool         `json:"stream"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ModelInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Description string `json:"description"`
}

type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

var config Config

func LoadConfig() error {
	config = Config{
		ConsumerNodeURL:  os.Getenv("CONSUMER_NODE_URL"),
		MarketplaceURL:  os.Getenv("MARKETPLACE_URL"),
		SessionDuration: os.Getenv("SESSION_DURATION"),
		InternalAPIPort: os.Getenv("INTERNAL_API_PORT"),
		AuthToken:       os.Getenv("AUTH_TOKEN"),
	}

	if config.ConsumerNodeURL == "" {
		return errors.New("CONSUMER_NODE_URL environment variable is required")
	}
	if config.SessionDuration == "" {
		config.SessionDuration = "1h" // Default session duration
	}
	if config.InternalAPIPort == "" {
		config.InternalAPIPort = "8081" // Default port
	}

	return nil
}

func StartServer() error {
	if err := LoadConfig(); err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	http.HandleFunc("/v1/chat/completions", HandleChatCompletions)
	
	log.Printf("Starting server on port %s", config.InternalAPIPort)
	return http.ListenAndServe(":"+config.InternalAPIPort, nil)
}

func getModelByHandle(modelHandle string) (*ModelInfo, error) {
	// If DUMMY_MODEL is set, validate against known test model
	if os.Getenv("DUMMY_MODEL") != "" {
		if !strings.EqualFold(modelHandle, "LMR-Hermes-2-Theta-Llama-3-8B") {
			return nil, fmt.Errorf("model not found: %s", modelHandle)
		}
		return &ModelInfo{
			ID:          1,
			Name:        modelHandle,
			Endpoint:    "dummy",
			Description: "Dummy model for testing",
		}, nil
	}

	url := fmt.Sprintf("%s/v1/models", config.ConsumerNodeURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", config.AuthToken)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get models, status: %d, body: %s", resp.StatusCode, string(body))
	}
	
	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %v", err)
	}
	
	for _, model := range modelsResp.Models {
		if strings.EqualFold(model.Name, modelHandle) {
			return &model, nil
		}
	}
	
	return nil, fmt.Errorf("model not found: %s", modelHandle)
}

// HandleChatCompletions processes chat completion requests
func HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read and parse the request body
	var chatReq ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate request
	if len(chatReq.Messages) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Messages array cannot be empty",
		})
		return
	}

	// Get model info based on the requested model handle
	model, err := getModelByHandle(chatReq.Model)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Error getting model info: %v", err),
		})
		return
	}

	// Create session with default test parameters via POST /sessions
	session, err := CreateSession("admin", "test-auth-token-12345")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Error creating session: %v", err),
		})
		return
	}

	// Return response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"model": chatReq.Model,
		"session_token": session.SessionToken,
		"expires_at": session.ExpiresAt,
		"model_id": model.ID,
	})
}

func CreateSession(username, password string) (*SessionResponse, error) {
	// If DUMMY_MODEL is set, return a dummy session for testing
	if os.Getenv("DUMMY_MODEL") != "" {
		return &SessionResponse{
			SessionToken: "test-session-123",
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		}, nil
	}

	url := fmt.Sprintf("%s/v1/sessions", config.ConsumerNodeURL)

	// For the new API, the payload expects a bidId. We'll use a fixed bidId (e.g., 1) for testing.
	payload := map[string]interface{}{
		"bidId": 1,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", config.AuthToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var eResp ErrorResponse
		if err := json.Unmarshal(respBody, &eResp); err != nil {
			return nil, fmt.Errorf("failed to create session, status: %d, body: %s", resp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("failed to create session: %s", eResp.Error)
	}

	var sessionResp SessionResponse
	if err := json.Unmarshal(respBody, &sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode session response: %v", err)
	}

	// If ExpiresAt is zero, set a default value
	if sessionResp.ExpiresAt.IsZero() {
		sessionResp.ExpiresAt = time.Now().Add(1 * time.Hour)
	}

	return &sessionResp, nil
} 