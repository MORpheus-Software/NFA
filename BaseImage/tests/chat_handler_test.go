package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/MORpheusSoftware/NFA/BaseImage/sessions"
)

const (
	testModelHandle = "LMR-Hermes-2-Theta-Llama-3-8B"
	consumerNodeURL = "https://consumer-node-2cmojdnxfq-uw.a.run.app"
)

func setupTestEnvironment(t *testing.T) func() {
	// Save original env vars
	originalEnv := map[string]string{
		"CONSUMER_NODE_URL": os.Getenv("CONSUMER_NODE_URL"),
		"MARKETPLACE_URL":   os.Getenv("MARKETPLACE_URL"),
		"SESSION_DURATION":  os.Getenv("SESSION_DURATION"),
		"INTERNAL_API_PORT": os.Getenv("INTERNAL_API_PORT"),
		"MODEL_NAME":        os.Getenv("MODEL_NAME"),
	}

	// Set test env vars to use the remote consumer API
	os.Setenv("CONSUMER_NODE_URL", consumerNodeURL)
	os.Setenv("MARKETPLACE_URL", consumerNodeURL)
	os.Setenv("SESSION_DURATION", "1h")
	os.Setenv("INTERNAL_API_PORT", "8081")
	os.Setenv("MODEL_NAME", testModelHandle)

	// Initialize the sessions package
	if err := sessions.LoadConfig(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Return cleanup function
	return func() {
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}

// checkConsumerNodeHealth verifies the consumer node is accessible
func checkConsumerNodeHealth(t *testing.T) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/system/config", consumerNodeURL), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.SetBasicAuth("admin", os.Getenv("AUTH_TOKEN"))
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("consumer node health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("consumer node returned status %d: %s", resp.StatusCode, string(body))
	}

	var config struct {
		Version string `json:"version"`
		Network string `json:"network"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return fmt.Errorf("failed to decode system config: %v", err)
	}

	t.Logf("Connected to consumer node version %s on network %s", config.Version, config.Network)
	return nil
}

// getConsumerNodeLogs retrieves recent logs from the consumer node
func getConsumerNodeLogs(t *testing.T) string {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/logs", consumerNodeURL), nil)
	if err != nil {
		t.Logf("Failed to create request: %v", err)
		return ""
	}

	req.SetBasicAuth("admin", os.Getenv("AUTH_TOKEN"))
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Failed to get consumer node logs: %v", err)
		return ""
	}
	defer resp.Body.Close()

	logs, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Failed to read consumer node logs: %v", err)
		return ""
	}
	return string(logs)
}

func TestChatHandlerIntegration(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Check consumer node health before starting tests
	if err := checkConsumerNodeHealth(t); err != nil {
		t.Fatalf("Consumer node is not healthy: %v", err)
	}

	tests := []struct {
		name           string
		request       *sessions.ChatCompletionRequest
		expectedStatus int
		validateResponse func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name: "Valid Non-Streaming Request",
			request: &sessions.ChatCompletionRequest{
				Model: testModelHandle,
				Messages: []sessions.ChatMessage{
					{Role: "user", Content: "Hello"},
				},
				Stream: false,
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
					// If response validation fails, check consumer node logs
					t.Logf("Consumer node logs:\n%s", getConsumerNodeLogs(t))
					t.Fatalf("Failed to decode response: %v\nResponse body: %s", err, resp.Body.String())
				}

				// Validate session token and model info are present
				if response["session_token"] == nil {
					t.Logf("Consumer node logs:\n%s", getConsumerNodeLogs(t))
					t.Error("Expected session_token in response")
				}
				if response["model"] != testModelHandle {
					t.Logf("Consumer node logs:\n%s", getConsumerNodeLogs(t))
					t.Errorf("Expected model %s, got %v", testModelHandle, response["model"])
				}
			},
		},
		{
			name: "Invalid Model Handle",
			request: &sessions.ChatCompletionRequest{
				Model: "invalid-model",
				Messages: []sessions.ChatMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
					t.Logf("Consumer node logs:\n%s", getConsumerNodeLogs(t))
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response["error"] == nil {
					t.Logf("Consumer node logs:\n%s", getConsumerNodeLogs(t))
					t.Error("Expected error message in response")
				}
			},
		},
		{
			name: "Empty Messages",
			request: &sessions.ChatCompletionRequest{
				Model:    testModelHandle,
				Messages: []sessions.ChatMessage{},
			},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
					t.Logf("Consumer node logs:\n%s", getConsumerNodeLogs(t))
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response["error"] == nil {
					t.Logf("Consumer node logs:\n%s", getConsumerNodeLogs(t))
					t.Error("Expected error message in response")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request body
			body, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Create test request
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			resp := httptest.NewRecorder()

			// Call handler directly
			sessions.HandleChatCompletions(resp, req)

			// Check status code
			if resp.Code != tc.expectedStatus {
				t.Logf("Consumer node logs:\n%s", getConsumerNodeLogs(t))
				t.Errorf("Expected status code %d, got %d\nResponse body: %s", 
					tc.expectedStatus, resp.Code, resp.Body.String())
			}

			// Validate response
			tc.validateResponse(t, resp)
		})
	}
} 