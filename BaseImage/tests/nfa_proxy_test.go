package tests_test

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

// getAvailablePort returns a random available port
func getAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// TestChatEndpointIntegration is an end-to-end integration test for the nfa-proxy chat endpoint.
// It loads environment variables from a .env file, starts a local nfa-proxy instance,
// and tests the chat completion endpoint which forwards to the remote consumer node.
func TestChatEndpointIntegration(t *testing.T) {
	// Load test environment variables
	err := godotenv.Load("../.env.test")
	if err != nil {
		t.Fatalf("Error loading .env.test file: %v", err)
	}

	// Get an available port
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}
	os.Setenv("PORT", fmt.Sprintf("%d", port))

	// Verify required environment variables
	requiredEnvVars := []string{
		"AUTH_TOKEN",
		"MARKETPLACE_URL",
		"MODEL_HANDLE",
	}

	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			t.Fatalf("Required environment variable %s is not set", envVar)
		}
	}

	// Start the local nfa-proxy
	proxyCmd := exec.Command("go", "run", "../main.go")
	proxyCmd.Env = os.Environ()
	
	// Capture proxy output for debugging
	proxyCmd.Stdout = os.Stdout
	proxyCmd.Stderr = os.Stderr
	
	if err := proxyCmd.Start(); err != nil {
		t.Fatalf("Failed to start local nfa-proxy: %v", err)
	}
	defer func() {
		if err := proxyCmd.Process.Kill(); err != nil {
			t.Logf("Failed to kill proxy process: %v", err)
		}
	}()

	// Wait for proxy to start and verify it's running
	time.Sleep(2 * time.Second)
	proxyURL := fmt.Sprintf("http://localhost:%d", port)
	
	// Test proxy health endpoint
	healthURL := fmt.Sprintf("%s/health", proxyURL)
	if resp, err := http.Get(healthURL); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Proxy health check failed: %v", err)
	}

	// Check consumer node health first
	consumerHealthURL := fmt.Sprintf("%s/healthcheck", os.Getenv("MARKETPLACE_URL"))
	t.Logf("Checking consumer node health at %s", consumerHealthURL)
	
	healthResp, err := http.Get(consumerHealthURL)
	if err != nil {
		t.Fatalf("Failed to check consumer node health: %v", err)
	}
	defer healthResp.Body.Close()
	
	if healthResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(healthResp.Body)
		t.Fatalf("Consumer node health check failed with status %d: %s", 
			healthResp.StatusCode, string(bodyBytes))
	}
	
	t.Log("Consumer node health check passed")

	// Build the chat completion request payload
	requestPayload := map[string]interface{}{
		"model":    os.Getenv("MODEL_HANDLE"),
		"messages": []map[string]string{{"role": "user", "content": "Hello"}},
		"stream":   true,
	}

	reqBody, err := json.Marshal(requestPayload)
	if err != nil {
		t.Fatalf("Failed to marshal request payload: %v", err)
	}

	// Build the HTTP request to local proxy
	endpoint := proxyURL + "/v1/chat/completions"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}

	// Set up basic auth header
	authHeader := fmt.Sprintf("Basic %s", 
		base64.StdEncoding.EncodeToString([]byte("admin:"+os.Getenv("AUTH_TOKEN"))))
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	t.Logf("Sending chat request to %s", endpoint)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Chat request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Check that the Content-Type indicates a streaming response
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Expected Content-Type 'text/event-stream', got: %s", contentType)
	}

	// Read and validate the streaming response
	reader := bufio.NewReader(resp.Body)
	for i := 0; i < 5; i++ { // Read up to 5 lines or until EOF
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Error reading from response stream: %v", err)
		}

		if !strings.HasPrefix(line, "data:") {
			t.Fatalf("Expected stream line to start with 'data:', got: %s", line)
		}

		t.Logf("Received streaming response line %d: %s", i+1, line)
	}
} 