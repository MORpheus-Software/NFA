package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"bufio"
	"encoding/base64"
	"net/http/httptest"

	"github.com/MORpheusSoftware/NFA/BaseImage/sessions"
)

// clearExistingSessions attempts to clear any existing sessions for our test account
func clearExistingSessions(t *testing.T) (string, error) {
	// Get the model ID first since we need it to list sessions
	url := fmt.Sprintf("%s/blockchain/models", os.Getenv("CONSUMER_NODE_URL"))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Logf("Failed to create request to list models: %v", err)
		return "", err
	}

	// Set up auth
	auth := fmt.Sprintf("%s:%s", os.Getenv("CONSUMER_USERNAME"), os.Getenv("CONSUMER_PASSWORD"))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(auth))))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Failed to list models: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Failed to list models, status: %d, body: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("failed to list models: %s", string(body))
	}

	// Parse the models response to get our target model ID
	var modelsResp struct {
		Models []struct {
			ID   string `json:"Id"`
			Name string `json:"Name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		t.Logf("Failed to decode models response: %v", err)
		return "", err
	}

	var modelID string
	modelHandles := []string{
		"LMR-Hermes-2-Theta-Llama-3-8B",
		"LMR-Capybara Hermes 2.5 Mistral-7B",
		"LMR-HyperB-Qwen2.5-Coder-32B",
		"LMR-OpenAI-GPT-4o",
		"LMR-ClaudeAI-Sonnet",
	}

	// Try to find any of our target models
	for _, model := range modelsResp.Models {
		for _, handle := range modelHandles {
			if model.Name == handle {
				modelID = model.ID
				t.Logf("Found model %s with ID %s", handle, modelID)
				break
			}
		}
		if modelID != "" {
			break
		}
	}

	if modelID == "" {
		t.Logf("Could not find any of the target models")
		return "", fmt.Errorf("no target models found")
	}

	// Get all sessions for our user
	sessionsURL := fmt.Sprintf("%s/blockchain/sessions/user", os.Getenv("CONSUMER_NODE_URL"))
	req, err = http.NewRequest("GET", sessionsURL, nil)
	if err != nil {
		t.Logf("Failed to create request to list sessions: %v", err)
		return modelID, nil
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(auth))))
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		t.Logf("Failed to list sessions: %v", err)
		return modelID, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Failed to list sessions, status: %d, body: %s", resp.StatusCode, string(body))
		return modelID, nil
	}

	// Parse the sessions response
	var sessionsResp struct {
		Sessions []struct {
			ID string `json:"id"`
		} `json:"sessions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sessionsResp); err != nil {
		t.Logf("Failed to decode sessions response: %v", err)
		return modelID, nil
	}

	// Delete each session
	for _, session := range sessionsResp.Sessions {
		deleteURL := fmt.Sprintf("%s/blockchain/sessions/%s", os.Getenv("CONSUMER_NODE_URL"), session.ID)
		req, err := http.NewRequest("DELETE", deleteURL, nil)
		if err != nil {
			t.Logf("Failed to create delete request for session %s: %v", session.ID, err)
			continue
		}

		req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(auth))))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Failed to delete session %s: %v", session.ID, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Logf("Failed to delete session %s, status: %d", session.ID, resp.StatusCode)
		} else {
			t.Logf("Successfully deleted session %s", session.ID)
		}
	}

	return modelID, nil
}

func TestHandleChatCompletions(t *testing.T) {
	// Set required environment variables for testing
	os.Setenv("CONSUMER_NODE_URL", "https://consumer-node-2cmojdnxfq-uw.a.run.app")
	os.Setenv("MARKETPLACE_URL", "https://consumer-node-2cmojdnxfq-uw.a.run.app")
	os.Setenv("CONSUMER_USERNAME", "proxy")
	os.Setenv("CONSUMER_PASSWORD", "yosz9BZCuu7Rli7mYe4G1JbIO0Yprvwl")
	os.Setenv("SESSION_DURATION", "1h")
	os.Setenv("INTERNAL_API_PORT", "8082")
	
	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("CONSUMER_NODE_URL")
		os.Unsetenv("MARKETPLACE_URL")
		os.Unsetenv("CONSUMER_USERNAME")
		os.Unsetenv("CONSUMER_PASSWORD")
		os.Unsetenv("SESSION_DURATION")
		os.Unsetenv("INTERNAL_API_PORT")
	}()

	// Clear any existing sessions
	if _, err := clearExistingSessions(t); err != nil {
		t.Fatalf("Failed to clear sessions: %v", err)
	}

	// Load config
	if err := sessions.LoadConfig(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Start a test server
	server := httptest.NewServer(http.HandlerFunc(sessions.HandleChatCompletions))
	defer server.Close()

	// Create request using the proper struct
	reqBody := sessions.ChatCompletionRequest{
		Model: "LMR-Hermes-2-Theta-Llama-3-8B", // Use model handle, not ID
		Messages: []sessions.ChatMessage{
			{
				Role:    "user",
				Content: "Hello, how are you?",
			},
		},
		Stream: true,
		StakeAmount: "5000000000000000000",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	t.Logf("Making request with body: %s", string(jsonData))

	// Create request with proper headers for SSE
	req, err := http.NewRequest("POST", server.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{
		Timeout: 60 * time.Second, // Increased timeout for streaming
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check response headers
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected Content-Type text/event-stream, got %s. Response body: %s", resp.Header.Get("Content-Type"), string(body))
	}

	// Read the streaming response
	reader := bufio.NewReader(resp.Body)
	var receivedData bool
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Error reading stream: %v", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		t.Logf("Received line: %s", line)

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}
			// Try to parse the response to validate it's proper JSON
			var streamResp map[string]interface{}
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				t.Logf("Warning: Received non-JSON data: %s", data)
			}
			receivedData = true
		}
	}

	if !receivedData {
		t.Fatal("No data received from stream")
	}
}