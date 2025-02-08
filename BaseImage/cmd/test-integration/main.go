package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MORpheusSoftware/NFA/BaseImage/mocks"
	"github.com/MORpheusSoftware/NFA/BaseImage/sessions"
)

func main() {
	// Start mock Lumerin Node API
	mockAPI := &mocks.MockLumerinNodeAPI{}
	go func() {
		if err := mockAPI.Start(8083); err != nil {
			log.Fatalf("Failed to start mock API: %v", err)
		}
	}()

	// Set environment variables for testing
	os.Setenv("CONSUMER_NODE_URL", "http://localhost:8083")
	os.Setenv("MARKETPLACE_URL", "http://localhost:8083")
	os.Setenv("CONSUMER_USERNAME", "admin")
	os.Setenv("CONSUMER_PASSWORD", "test-auth-token-12345")
	os.Setenv("SESSION_DURATION", "1h")
	os.Setenv("INTERNAL_API_PORT", "8081")

	// Start proxy server
	go func() {
		if err := sessions.StartServer(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Cleanup
	mockAPI.Stop()
} 