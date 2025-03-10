package sessions

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	maxRetries = 3
	sessionTimeout = 60 * time.Second
)

type Config struct {
	ConsumerNodeURL   string
	MarketplaceURL   string
	SessionDuration  string
	InternalAPIPort  string
	AuthToken       string
}

type SessionResponse struct {
	SessionToken string    `json:"session_id"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool         `json:"stream"`
	StakeAmount string       `json:"stake_amount,omitempty"` // Amount to stake in wei
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Description string `json:"description"`
}

type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

// Cookie file credentials
type Credentials struct {
	Username string
	Password string
}

var config Config
var credentials *Credentials

type SessionManager interface {
	GetModelByHandle(modelHandle string) (*ModelInfo, error)
	CreateSession(modelId string, stakeAmount string) (*SessionResponse, error)
	SendChatMessage(sessionToken string, modelId string, message string, stream bool, w StreamWriter) (*ChatResponse, error)
}

type DefaultSessionManager struct{}

var sessionManager SessionManager = &DefaultSessionManager{}

// SetSessionManager allows injecting a mock for testing
func SetSessionManager(sm SessionManager) {
	sessionManager = sm
}

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

func HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func StartServer() error {
	if err := LoadConfig(); err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	http.HandleFunc("/health", HandleHealthCheck)
	http.HandleFunc("/v1/chat/completions", HandleChatCompletions)
	
	log.Printf("Starting server on port %s", config.InternalAPIPort)
	return http.ListenAndServe(":"+config.InternalAPIPort, nil)
}

func authenticate() error {
	// Only authenticate once
	if credentials != nil {
		return nil
	}

	// Try to read credentials from cookie file first
	cookiePath := os.Getenv("COOKIE_FILE_PATH")
	if cookiePath == "" {
		cookiePath = ".cookie" // Default to current directory
	}

	// Try to read existing cookie file
	if data, err := os.ReadFile(cookiePath); err == nil {
		parts := strings.Split(strings.TrimSpace(string(data)), ":")
		if len(parts) == 2 {
			credentials = &Credentials{
				Username: parts[0],
				Password: parts[1],
			}
			return nil
		}
	}

	// If no cookie file, try environment variables
	username := os.Getenv("CONSUMER_USERNAME")
	if username == "" {
		return fmt.Errorf("CONSUMER_USERNAME environment variable is required")
	}
	password := os.Getenv("CONSUMER_PASSWORD")
	if password == "" {
		return fmt.Errorf("no credentials found in cookie file or environment")
	}

	credentials = &Credentials{
		Username: username,
		Password: password,
	}

	return nil
}

// Helper function to set Basic Auth header
func setBasicAuth(req *http.Request) error {
	if err := authenticate(); err != nil {
		return err
	}
	
	// Use credentials in Basic Auth header
	auth := fmt.Sprintf("%s:%s", credentials.Username, credentials.Password)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	log.Printf("Debug - Auth string before encoding: %s", auth)
	log.Printf("Debug - Auth string after encoding: %s", encodedAuth)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", encodedAuth))
	return nil
}

func getModelByHandle(modelHandle string) (*ModelInfo, error) {
	url := fmt.Sprintf("%s/blockchain/models", config.ConsumerNodeURL)
	
	// Create request once, reuse for retries
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	
	// Set required headers according to spec
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if err := setBasicAuth(req); err != nil {
		return nil, fmt.Errorf("failed to set auth: %v", err)
	}
	
	client := &http.Client{
		Timeout: 30 * time.Second,  // Increased timeout
		Transport: &http.Transport{
			ResponseHeaderTimeout: 25 * time.Second,
			IdleConnTimeout:      20 * time.Second,
			DisableKeepAlives:    true,
		},
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Getting models attempt %d of %d", attempt, maxRetries)
		
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error getting models (attempt %d): %v", attempt, err)
			lastErr = err
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			if err, ok := err.(net.Error); ok && err.Timeout() {
				return nil, fmt.Errorf("models request timed out after %d attempts: %v", attempt, err)
			}
			return nil, fmt.Errorf("failed to connect to consumer node after %d attempts: %v", attempt, err)
		}
		defer resp.Body.Close()
		
		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading models response body (attempt %d): %v", attempt, err)
			lastErr = err
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, fmt.Errorf("failed to read response body: %v", err)
		}

		log.Printf("Got models response: %s", string(body))
		
		// Handle different status codes according to spec
		switch resp.StatusCode {
		case http.StatusOK:
			var modelsResp ModelsResponse
			if err := json.Unmarshal(body, &modelsResp); err != nil {
				return nil, fmt.Errorf("failed to decode models response: %v", err)
			}
			
			// Look for the requested model
			for _, model := range modelsResp.Models {
				if strings.EqualFold(model.Name, modelHandle) {
					return &model, nil
				}
			}
			return nil, fmt.Errorf("model not found: %s", modelHandle)
			
		case http.StatusUnauthorized:
			var errorResp ErrorResponse
			if err := json.Unmarshal(body, &errorResp); err != nil {
				return nil, fmt.Errorf("unauthorized: %s", string(body))
			}
			return nil, fmt.Errorf("unauthorized: %s", errorResp.Error)
			
		case http.StatusBadRequest:
			var errorResp ErrorResponse
			if err := json.Unmarshal(body, &errorResp); err != nil {
				return nil, fmt.Errorf("bad request: %s", string(body))
			}
			return nil, fmt.Errorf("bad request: %s", errorResp.Error)
			
		case http.StatusServiceUnavailable:
			log.Printf("Service unavailable (attempt %d), retrying...", attempt)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, fmt.Errorf("service unavailable after %d attempts", maxRetries)
			
		default:
			if attempt < maxRetries {
				log.Printf("Unexpected status code %d (attempt %d), retrying...", resp.StatusCode, attempt)
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
		}
	}

	return nil, fmt.Errorf("failed to get models after %d attempts, last error: %v", maxRetries, lastErr)
}

func CreateSession(modelId string, stakeAmount string) (*SessionResponse, error) {
	url := fmt.Sprintf("%s/blockchain/models/%s/session", config.ConsumerNodeURL, modelId)

	// Create session request payload according to OpenSessionWithFailover spec
	duration, err := time.ParseDuration(config.SessionDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse session duration: %v", err)
	}
	durationSecs := int64(duration.Seconds())

	payload := map[string]interface{}{
		"sessionDuration": fmt.Sprintf("%d", durationSecs),
		"directPayment": false,
		"failover": false,
		"fee": "300000000000",  // Adding standard fee amount
		"stake": stakeAmount,   // Including stake amount from request
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	log.Printf("Creating session with URL: %s and payload: %s", url, string(body))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if err := setBasicAuth(req); err != nil {
		return nil, fmt.Errorf("failed to set auth: %v", err)
	}

	// Log request details
	log.Printf("Making session request to URL: %s", url)
	log.Printf("Session request payload: %s", string(body))
	log.Printf("Session request headers: %+v", req.Header)

	client := &http.Client{
		Timeout: sessionTimeout,
		Transport: &http.Transport{
			ResponseHeaderTimeout: 55 * time.Second,
			IdleConnTimeout:      30 * time.Second,
			DisableKeepAlives:    true,
		},
	}
	
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Session creation attempt %d of %d", attempt, maxRetries)
		
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error making session request (attempt %d): %v", attempt, err)
			lastErr = err
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			if err, ok := err.(net.Error); ok && err.Timeout() {
				return nil, fmt.Errorf("session creation timed out after %d attempts: %v", attempt, err)
			}
			return nil, fmt.Errorf("failed to connect to consumer node after %d attempts: %v", attempt, err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading session response body (attempt %d): %v", attempt, err)
			lastErr = err
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, fmt.Errorf("failed to read response body: %v", err)
		}

		log.Printf("Session creation response - Status: %d, Headers: %+v", resp.StatusCode, resp.Header)
		log.Printf("Session creation response body: %s", string(respBody))

		if resp.StatusCode == http.StatusServiceUnavailable {
			log.Printf("Service unavailable (attempt %d), retrying...", attempt)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
			var eResp ErrorResponse
			if err := json.Unmarshal(respBody, &eResp); err != nil {
				log.Printf("Failed to parse error response: %v", err)
				lastErr = fmt.Errorf("failed to create session, status: %d, body: %s", resp.StatusCode, string(respBody))
				if attempt < maxRetries {
					time.Sleep(time.Duration(attempt) * time.Second)
					continue
				}
				return nil, lastErr
			}
			log.Printf("Session creation error: %s", eResp.Error)
			lastErr = fmt.Errorf("failed to create session: %s", eResp.Error)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, lastErr
		}

		// Parse the session response
		var sessionResp struct {
			SessionID string `json:"sessionID"`
		}
		if err := json.Unmarshal(respBody, &sessionResp); err != nil {
			log.Printf("Failed to parse session response: %v", err)
			lastErr = fmt.Errorf("failed to decode session response: %v", err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, lastErr
		}

		log.Printf("Successfully created session with token: %s", sessionResp.SessionID)

		// Return the session response with the session ID
		return &SessionResponse{
			SessionToken: sessionResp.SessionID,
			ExpiresAt:   time.Now().Add(duration),
		}, nil
	}

	return nil, fmt.Errorf("failed to create session after %d attempts, last error: %v", maxRetries, lastErr)
}

type StreamWriter interface {
	Write([]byte) (int, error)
	Flush()
}

func SendChatMessage(sessionToken string, modelId string, message string, stream bool, w StreamWriter) (*ChatResponse, error) {
	url := fmt.Sprintf("%s/v1/chat/completions", config.ConsumerNodeURL)

	payload := map[string]interface{}{
		"model": modelId,
		"messages": []map[string]string{
			{
				"role": "user",
				"content": message,
			},
		},
		"stream": stream,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	log.Printf("Sending chat request to %s with payload: %s", url, string(body))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	// Set headers according to API spec
	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	if err := setBasicAuth(req); err != nil {
		return nil, fmt.Errorf("failed to set auth: %v", err)
	}
	req.Header.Set("session_id", sessionToken)

	log.Printf("Chat request headers: %v", req.Header)

	client := &http.Client{Timeout: 60 * time.Second} // Increased timeout for streaming
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to consumer node: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Chat error response: %s", string(respBody))
		return nil, fmt.Errorf("chat request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// For streaming responses
	if stream && w != nil {
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return &ChatResponse{}, nil
				}
				return nil, err
			}

			// Skip empty lines
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Handle SSE data
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					return &ChatResponse{}, nil
				}

				// Write the data directly to the response writer
				fmt.Fprintf(w, "data: %s\n\n", data)
				w.Flush()
			}
		}
	}

	// For non-streaming responses
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode chat response: %v", err)
	}

	log.Printf("Successfully received chat response: %s", chatResp.Response)
	return &chatResp, nil
}

// HandleChatCompletions processes chat completion requests
func HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	// Set headers for streaming response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
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
	model, err := sessionManager.GetModelByHandle(chatReq.Model)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Error getting model info: %v", err),
		})
		return
	}

	// Create session with the consumer node using the model ID
	session, err := sessionManager.CreateSession(model.ID, chatReq.StakeAmount)
	if err != nil {
		// Check for specific error cases
		if strings.Contains(err.Error(), "no provider accepting session") {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Error creating session: %v", err),
		})
		return
	}

	// Wait for 20 seconds after session creation
	log.Printf("Waiting 20 seconds for session initialization...")
	time.Sleep(20 * time.Second)
	log.Printf("Resuming after wait, sending chat message...")

	// Send the chat message
	lastMessage := chatReq.Messages[len(chatReq.Messages)-1]

	// Get the flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Streaming not supported",
		})
		return
	}

	// Create a wrapper that implements StreamWriter
	streamWriter := struct {
		http.ResponseWriter
		http.Flusher
	}{w, flusher}

	// Start streaming
	_, err = sessionManager.SendChatMessage(session.SessionToken, model.ID, lastMessage.Content, true, streamWriter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Error sending chat message: %v", err),
		})
		return
	}
}

func (sm *DefaultSessionManager) GetModelByHandle(modelHandle string) (*ModelInfo, error) {
	return getModelByHandle(modelHandle)
}

func (sm *DefaultSessionManager) CreateSession(modelId string, stakeAmount string) (*SessionResponse, error) {
	return CreateSession(modelId, stakeAmount)
}

func (sm *DefaultSessionManager) SendChatMessage(sessionToken string, modelId string, message string, stream bool, w StreamWriter) (*ChatResponse, error) {
	return SendChatMessage(sessionToken, modelId, message, stream, w)
} 