package main

import (
	"log"
	"os"

	"github.com/MORpheusSoftware/NFA/BaseImage/sessions"
	"github.com/joho/godotenv"
)

func init() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	
	// Load environment variables
	if err := loadEnv(); err != nil {
		log.Printf("Warning: Error loading environment: %v", err)
	}
}

func loadEnv() error {
	// Try loading from different env files in order of preference
	envFiles := []string{".env", ".env.test", ".env.example"}
	
	var loadedFile string
	var err error
	
	for _, file := range envFiles {
		if _, err = os.Stat(file); err == nil {
			if err = godotenv.Load(file); err == nil {
				loadedFile = file
				break
			}
		}
	}

	if loadedFile != "" {
		log.Printf("Loaded environment from %s", loadedFile)
	} else {
		log.Printf("No environment file found, using existing environment variables")
	}

	// Validate required environment variables
	required := []string{
		"CONSUMER_NODE_URL",
		"MARKETPLACE_URL",
	}

	for _, env := range required {
		if os.Getenv(env) == "" {
			log.Printf("Warning: Required environment variable %s is not set", env)
		}
	}

	return nil
}

func main() {
	log.Printf("Starting NFA Proxy Server...")
	
	// Start the proxy server
	if err := sessions.StartServer(); err != nil {
		log.Fatal(err)
	}
}