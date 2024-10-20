// dto.go

package marketplacesdk

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sashabaranov/go-openai"
)

// WalletResponse represents a response containing the wallet address information.
type WalletResponse struct {
	Address string `json:"address"`
}

// SessionRequest represents a request to start a new session.
type SessionRequest struct {
	ModelID         string   `json:"modelId"`
	SessionDuration *big.Int `json:"sessionDuration"`
}

// SessionStakeRequest represents a request to stake for a session.
type SessionStakeRequest struct {
	Approval    string   `json:"approval"`
	ApprovalSig string   `json:"approvalSig"`
	Stake       *big.Int `json:"stake"` // Changed from uint64 to *big.Int for consistency
}

// CloseSessionRequest represents a request to close an ongoing session.
type CloseSessionRequest struct {
	SessionID string `json:"id"`
}

// WalletRequest represents a request to create or set up a wallet.
type WalletRequest struct {
	PrivateKey string `json:"privateKey"`
}

// ProdiaGenerationRequest represents a request to generate data from the Prodia model.
type ProdiaGenerationRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	ApiURL string `json:"apiUrl"`
	ApiKey string `json:"apiKey"`
}

// OpenSessionRequest represents a request to open a session.
type OpenSessionRequest struct {
	Approval    string   `json:"approval"`
	ApprovalSig string   `json:"approvalSig"`
	Stake       *big.Int `json:"stake"`
}

// SendRequest represents a request to send a specific amount to a recipient.
type SendRequest struct {
	To     common.Address `json:"to"`
	Amount *big.Int       `json:"amount"`
}

// OpenSessionWithDurationRequest represents a request to open a session with a specific duration.
type OpenSessionWithDurationRequest struct {
	SessionDuration *big.Int `json:"sessionDuration"`
}

// CreateBidRequest represents a request to create a bid with a model ID and price per second.
type CreateBidRequest struct {
	ModelID        string   `json:"modelID"`
	PricePerSecond *big.Int `json:"pricePerSecond"`
}

// CreateProviderRequest represents a request to register a provider.
type CreateProviderRequest struct {
	Stake    *big.Int `json:"stake"`
	Endpoint string   `json:"endpoint"`
}

// CreateModelRequest represents a request to create a model.
type CreateModelRequest struct {
	Name   string   `json:"name"`
	IpfsID string   `json:"ipfsID"`
	Fee    *big.Int `json:"fee"`
	Stake  *big.Int `json:"stake"`
	Tags   []string `json:"tags"`
}

// RawTransaction represents a raw Ethereum transaction.
type RawTransaction struct {
	// Add fields as necessary
}

// RawEthTransactionResponse represents a response for raw Ethereum transactions.
type RawEthTransactionResponse struct {
	Status  string           `json:"status"`
	Message string           `json:"message"`
	Result  []RawTransaction `json:"result"`
}

// ChatCompletionMessage represents a message in a chat completion request.
type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponseFormat represents the response format for a chat completion request.
type ChatCompletionResponseFormat struct {
	Type string `json:"type,omitempty"`
}

// OpenAICompletionRequest represents a request for generating chat completions from OpenAI.
type OpenAICompletionRequest struct {
	Model            string                        `json:"model"`
	Messages         []ChatCompletionMessage       `json:"messages"`
	MaxTokens        int                           `json:"max_tokens,omitempty"`
	Temperature      float32                       `json:"temperature,omitempty"`
	TopP             float32                       `json:"top_p,omitempty"`
	N                int                           `json:"n,omitempty"`
	Stream           bool                          `json:"stream,omitempty"`
	Stop             []string                      `json:"stop,omitempty"`
	PresencePenalty  float32                       `json:"presence_penalty,omitempty"`
	ResponseFormat   *ChatCompletionResponseFormat `json:"response_format,omitempty"`
	Seed             *int                          `json:"seed,omitempty"`
	FrequencyPenalty float32                       `json:"frequency_penalty,omitempty"`
}

// StatusResponse represents a generic status response.
type StatusResponse struct {
	Status string `json:"status"`
}

// BalanceResponse represents a response containing balance information.
type BalanceResponse struct {
	Balance *big.Int `json:"balance"`
}

// BidResponse represents a response containing bid information.
type BidResponse struct {
	Bid *Bid `json:"bid"`
}

// SessionResponse represents a response containing session information.
type SessionResponse struct {
	SessionID string `json:"sessionID"`
}

// ProviderResponse represents a response containing provider information.
type ProviderResponse struct {
	Provider *Provider `json:"provider"`
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// TransactionResponse represents a response containing a transaction hash.
type TransactionResponse struct {
	TxHash string `json:"tx"`
}

// Bid represents a bid in the system.
type Bid struct {
	ID             string   `json:"id"`
	Provider       string   `json:"provider"`
	ModelAgentID   string   `json:"modelAgentId"`
	PricePerSecond *big.Int `json:"pricePerSecond"`
}

// Provider represents a provider in the system.
type Provider struct {
	Address  string   `json:"address"`
	Endpoint string   `json:"endpoint"`
	Stake    *big.Int `json:"stake"`
}

// Model represents a model in the system.
type Model struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Fee   *big.Int `json:"fee"`
	Stake *big.Int `json:"stake"`
	Tags  []string `json:"tags"`
}

// ApproveResponse represents the response from an approval request.
type ApproveResponse struct {
	Success bool `json:"success"`
}

// ApproveRequest represents the request payload for approving an allowance.
type ApproveRequest struct {
	Spender string `json:"spender"`
	Amount  string `json:"amount"` // Amount is represented as a string to accommodate large integers.
}

// AllowanceResponse represents the response when querying an allowance.
type AllowanceResponse struct {
	Allowance string `json:"allowance"` // Allowance is represented as a string to handle large values.
}

// ChatRequest represents the request payload for initiating a chat completion.
type ChatRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream bool `json:"stream"`
}

// ModelStats represents the statistics of a model.
type ModelStats struct {
	TotalSessions uint64   `json:"totalSessions"`
	TotalStake    *big.Int `json:"totalStake"`
	// Add other fields as defined in the smart contract's ModelStats struct
}

// ApiGatewayClientInterface defines the methods that ApiGatewayClient and MockApiGatewayClient must implement.
type ApiGatewayClientInterface interface {
	GetAllowance(ctx context.Context, spender string) (*big.Int, error)
	ApproveAllowance(ctx context.Context, spender string, amount *big.Int) (*ApproveResponse, error)
	OpenStakeSession(ctx context.Context, request *SessionStakeRequest) (*SessionResponse, error)
	PromptStream(ctx context.Context, request *openai.ChatCompletionRequest, modelID string, sessionID string, callback CompletionCallback) error
	// Add other methods as needed for your tests
}