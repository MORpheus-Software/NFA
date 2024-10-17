package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	marketplacesdk "github.com/MORpheusSoftware/NFA/MarketplaceSDK"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

type (
	ChatRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream bool `json:"stream"`
	}
)

var (
	client      *marketplacesdk.ApiGatewayClient
	myAgentID   string // Use this for model ID
	privateKey  *ecdsa.PrivateKey
	spenderAddr string // Spender address loaded from environment
)

func main() {
	// Load .env file
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get agent ID from .env
	myAgentID = os.Getenv("AGENT_ID")
	if myAgentID == "" {
		log.Fatal("AGENT_ID not set in .env file")
	}

	// Get agent wallet private key from .env
	agentWalletPKHex := os.Getenv("AGENT_WALLET_PK")
	if agentWalletPKHex == "" {
		log.Fatal("AGENT_WALLET_PK not set in .env file")
	}
	privateKey, err = crypto.HexToECDSA(agentWalletPKHex)
	if err != nil {
		log.Fatalf("Invalid AGENT_WALLET_PK: %v", err)
	}

	// Get spender address from .env
	spenderAddr = os.Getenv("AGENT_WALLET_ADDRESS")
	if spenderAddr == "" {
		log.Fatal("AGENT_WALLET_ADDRESS not set in .env file")
	}

	// Optional: Validate the spender address format
	if !common.IsHexAddress(spenderAddr) {
		log.Fatal("AGENT_WALLET_ADDRESS is not a valid Ethereum address")
	}

	// Initialize the Marketplace API client
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		log.Fatal("API_BASE_URL not set in .env file")
	}

	client = marketplacesdk.NewApiGatewayClient(apiBaseURL, nil)

	r := gin.Default()

	r.POST("/v1/chat/completions", func(c *gin.Context) {
		// Parse the request body
		var chatRequest ChatRequest
		if err := c.ShouldBindJSON(&chatRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		modelID := chatRequest.Model

		// Define the stake amount (e.g., 1 MOR in wei)
		stakeAmount := big.NewInt(1e18) // Adjust as needed

		// Check allowance and approve if necessary
		err := ensureAllowance(c.Request.Context(), stakeAmount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to ensure allowance: %v", err)})
			return
		}

		// Create approval data
		approvalData := []byte("Approve session") // Placeholder data
		approvalSig, err := signData(approvalData, privateKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to sign approval: %v", err)})
			return
		}

		// Create OpenSessionRequest
		openSessionRequest := &marketplacesdk.SessionStakeRequest{
			Approval:    string(approvalData),
			ApprovalSig: approvalSig,
			Stake:       stakeAmount,
		}

		// Open session using the correct endpoint
		sessionResponse, err := client.OpenStakeSession(c.Request.Context(), openSessionRequest)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to open session: %v", err)})
			return
		}

		currentSessionID := sessionResponse.SessionID

		// Prepare the chat completion request
		chatCompletionRequest := &openai.ChatCompletionRequest{
			Model:    chatRequest.Model,
			Messages: []openai.ChatCompletionMessage{},
			Stream:   chatRequest.Stream,
		}

		// Map messages from ChatRequest to ChatCompletionRequest
		for _, msg := range chatRequest.Messages {
			chatCompletionRequest.Messages = append(chatCompletionRequest.Messages, openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		// Set response headers for SSE
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		// Flush the headers
		c.Writer.Flush()

		// Use the PromptStream method to get the streaming response
		err = client.PromptStream(c.Request.Context(), chatCompletionRequest, modelID, currentSessionID, func(response interface{}) {
			// Handle the streamed response
			if completion, ok := response.(*openai.ChatCompletionStreamResponse); ok {
				// Prepare SSE event
				data, err := json.Marshal(completion)
				if err != nil {
					// Handle error
					log.Printf("Error marshalling completion: %v", err)
					return
				}
				// Write data to the client
				fmt.Fprintf(c.Writer, "data: %s\n\n", data)
				c.Writer.Flush()
			}
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error during prompt stream: %v", err)})
			return
		}
	})

	r.Run(":8080")
}

// ensureAllowance checks the MOR token allowance and approves if necessary
func ensureAllowance(ctx context.Context, amount *big.Int) error {
	allowance, err := client.GetAllowance(ctx, spenderAddr)
	if err != nil {
		return fmt.Errorf("failed to get allowance: %w", err)
	}

	if allowance.Cmp(amount) < 0 {
		// Approve the required amount
		_, err := client.ApproveAllowance(ctx, spenderAddr, amount)
		if err != nil {
			return fmt.Errorf("failed to approve allowance: %w", err)
		}
		// Wait for the transaction to be mined or include polling logic
		time.Sleep(15 * time.Second) // Adjust as needed
	}

	return nil
}

// signData signs the data using the provided private key
func signData(data []byte, key *ecdsa.PrivateKey) (string, error) {
	hash := crypto.Keccak256Hash(data)
	signature, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%x", signature), nil
}
