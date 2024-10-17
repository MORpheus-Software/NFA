package marketplacesdk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// CompletionCallback is a function type for handling streamed responses.
type CompletionCallback func(response interface{})

// NewApiGatewayClient creates a new API gateway client.
func NewApiGatewayClient(baseURL string, httpClient *http.Client) *ApiGatewayClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &ApiGatewayClient{
		BaseURL:    baseURL,
		HttpClient: httpClient,
	}
}

// ApiGatewayClient represents the API gateway client.
type ApiGatewayClient struct {
	BaseURL    string
	HttpClient *http.Client
}

// Helper function to make GET requests
func (c *ApiGatewayClient) getRequest(ctx context.Context, endpoint string, result interface{}) error {
	u, err := url.Parse(c.BaseURL + endpoint)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// Helper function to make POST requests
func (c *ApiGatewayClient) postRequest(ctx context.Context, endpoint string, body interface{}, result interface{}) error {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	u, err := url.Parse(c.BaseURL + endpoint)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

// handleErrorResponse handles non-200 HTTP responses.
func (c *ApiGatewayClient) handleErrorResponse(resp *http.Response) error {
	var errResp map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&errResp)
	if err != nil {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return fmt.Errorf("unexpected status code: %d, response: %v", resp.StatusCode, errResp)
}

// DTOs (Data Transfer Objects)

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
	Stake       *big.Int `json:"stake"`
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
	Stake  *big.Int `json:"stake"`
	Fee    *big.Int `json:"fee"`
	Tags   []string `json:"tags"`
}

// Model represents a model in the system.
type Model struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Fee       *big.Int `json:"fee"`
	Stake     *big.Int `json:"stake"`
	Tags      []string `json:"tags"`
	CreatedAt uint64   `json:"createdAt"`
	IpfsCID   string   `json:"ipfsCID"`
	Owner     string   `json:"owner"`
	IsDeleted bool     `json:"isDeleted"`
}

// Session represents a session in the system.
type Session struct {
	SessionID string `json:"sessionID"`
}

// SessionListItem represents a session list item.
type SessionListItem struct {
	ID               string `json:"id"`
	User             string `json:"user"`
	Provider         string `json:"provider"`
	ModelAgentID     string `json:"modelAgentId"`
	OpenedAt         int64  `json:"openedAt"`
	EndsAt           int64  `json:"endsAt"`
	ClosedAt         int64  `json:"closedAt"`
	CloseoutReceipt  string `json:"closeoutReceipt"`
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

// ModelStats represents the statistics of a model.
type ModelStats struct {
	TotalSessions uint64   `json:"totalSessions"`
	TotalStake    *big.Int `json:"totalStake"`
	// Add other fields as defined in the smart contract's ModelStats struct
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// TransactionResponse represents a response containing a transaction hash.
type TransactionResponse struct {
	TxHash string `json:"tx"`
}

// Implementing methods with updated DTOs and proper types

// GetProxyRouterConfig retrieves the proxy router configuration.
func (c *ApiGatewayClient) GetProxyRouterConfig(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.getRequest(ctx, "/config", &result)
	return result, err
}

// GetProxyRouterFiles retrieves the list of opened files by the application.
func (c *ApiGatewayClient) GetProxyRouterFiles(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.getRequest(ctx, "/files", &result)
	return result, err
}

// HealthCheck returns the application health status.
func (c *ApiGatewayClient) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.getRequest(ctx, "/healthcheck", &result)
	return result, err
}

// InitiateSession sends a handshake to the provider to initiate a session.
func (c *ApiGatewayClient) InitiateSession(ctx context.Context, req *SessionRequest) (*Session, error) {
	var result Session
	err := c.postRequest(ctx, "/proxy/sessions/initiate", req, &result)
	return &result, err
}

// PromptStream sends a prompt to a local or remote model and handles streaming responses.
func (c *ApiGatewayClient) PromptStream(ctx context.Context, request *openai.ChatCompletionRequest, modelID string, sessionID string, callback CompletionCallback) error {
	endpoint := "/v1/chat/completions"
	return c.requestChatCompletionStream(ctx, endpoint, request, callback, modelID, sessionID)
}

// GetLatestBlock retrieves the latest block number from the blockchain.
func (c *ApiGatewayClient) GetLatestBlock(ctx context.Context) (uint64, error) {
	var result struct {
		Block uint64 `json:"block"`
	}
	err := c.getRequest(ctx, "/blockchain/latestBlock", &result)
	return result.Block, err
}

// GetAllProviders retrieves the list of providers from the blockchain.
func (c *ApiGatewayClient) GetAllProviders(ctx context.Context) ([]Provider, error) {
	var result struct {
		Providers []Provider `json:"providers"`
	}
	err := c.getRequest(ctx, "/blockchain/providers", &result)
	return result.Providers, err
}

// CreateNewProvider registers or updates a provider in the blockchain.
func (c *ApiGatewayClient) CreateNewProvider(ctx context.Context, req *CreateProviderRequest) (*Provider, error) {
	var result struct {
		Provider Provider `json:"provider"`
	}
	err := c.postRequest(ctx, "/blockchain/providers", req, &result)
	return &result.Provider, err
}

// CreateNewModel registers a new model in the blockchain.
func (c *ApiGatewayClient) CreateNewModel(ctx context.Context, req *CreateModelRequest) (*Model, error) {
	var result struct {
		Model Model `json:"model"`
	}
	err := c.postRequest(ctx, "/blockchain/models", req, &result)
	return &result.Model, err
}

// CreateNewProviderBid creates a new bid in the blockchain.
func (c *ApiGatewayClient) CreateNewProviderBid(ctx context.Context, modelID string, pricePerSecond *big.Int) (*Bid, error) {
	req := &CreateBidRequest{
		ModelID:        modelID,
		PricePerSecond: pricePerSecond,
	}
	var result struct {
		Bid Bid `json:"bid"`
	}
	err := c.postRequest(ctx, "/blockchain/bids", req, &result)
	return &result.Bid, err
}

// GetAllModels retrieves the list of models from the blockchain.
func (c *ApiGatewayClient) GetAllModels(ctx context.Context) ([]Model, error) {
	var result struct {
		Models []Model `json:"models"`
	}
	err := c.getRequest(ctx, "/blockchain/models", &result)
	return result.Models, err
}

// GetBidsByProvider retrieves bids from the blockchain by provider address.
func (c *ApiGatewayClient) GetBidsByProvider(ctx context.Context, providerAddr string, offset *big.Int, limit uint8) ([]Bid, error) {
	endpoint := fmt.Sprintf("/blockchain/providers/%s/bids?offset=%s&limit=%d", providerAddr, offset.String(), limit)
	var result struct {
		Bids []Bid `json:"bids"`
	}
	err := c.getRequest(ctx, endpoint, &result)
	return result.Bids, err
}

// GetBidsByModelAgent retrieves bids from the blockchain by model agent ID.
func (c *ApiGatewayClient) GetBidsByModelAgent(ctx context.Context, modelAgentID string, offset string, limit string) ([]Bid, error) {
	endpoint := fmt.Sprintf("/blockchain/models/%s/bids?offset=%s&limit=%s", modelAgentID, offset, limit)
	var result struct {
		Bids []Bid `json:"bids"`
	}
	err := c.getRequest(ctx, endpoint, &result)
	return result.Bids, err
}

// ListUserSessions retrieves sessions from the blockchain by user address.
func (c *ApiGatewayClient) ListUserSessions(ctx context.Context, user string) ([]SessionListItem, error) {
	endpoint := fmt.Sprintf("/blockchain/sessions/user?user=%s", user)
	var result struct {
		Sessions []SessionListItem `json:"sessions"`
	}
	err := c.getRequest(ctx, endpoint, &result)
	return result.Sessions, err
}

// ListProviderSessions retrieves sessions from the blockchain by provider address.
func (c *ApiGatewayClient) ListProviderSessions(ctx context.Context, provider string) ([]SessionListItem, error) {
	endpoint := fmt.Sprintf("/blockchain/sessions/provider?provider=%s", provider)
	var result struct {
		Sessions []SessionListItem `json:"sessions"`
	}
	err := c.getRequest(ctx, endpoint, &result)
	return result.Sessions, err
}

// OpenStakeSession sends a transaction to the blockchain to open a session with stake.
func (c *ApiGatewayClient) OpenStakeSession(ctx context.Context, req *SessionStakeRequest) (*Session, error) {
	var result Session
	err := c.postRequest(ctx, "/blockchain/sessions", req, &result)
	return &result, err
}

// OpenSession opens a session by model ID in the blockchain.
func (c *ApiGatewayClient) OpenSession(ctx context.Context, req *OpenSessionWithDurationRequest, modelID string) (*Session, error) {
	var result Session
	endpoint := fmt.Sprintf("/blockchain/models/%s/session", modelID)
	err := c.postRequest(ctx, endpoint, req, &result)
	return &result, err
}

// GetLocalModels retrieves a list of available local AI models.
func (c *ApiGatewayClient) GetLocalModels(ctx context.Context) ([]Model, error) {
	var result []Model
	err := c.getRequest(ctx, "/v1/models", &result)
	return result, err
}

// CloseSession sends a transaction to the blockchain to close a session.
func (c *ApiGatewayClient) CloseSession(ctx context.Context, sessionID string) (*Session, error) {
	endpoint := fmt.Sprintf("/blockchain/sessions/%s/close", sessionID)
	var result Session
	err := c.postRequest(ctx, endpoint, nil, &result)
	return &result, err
}

// GetAllowance retrieves MOR token allowance for a spender.
func (c *ApiGatewayClient) GetAllowance(ctx context.Context, spender string) (*big.Int, error) {
	endpoint := fmt.Sprintf("/blockchain/allowance?spender=%s", spender)
	var result struct {
		Allowance string `json:"allowance"`
	}
	err := c.getRequest(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}
	allowance := new(big.Int)
	allowance.SetString(result.Allowance, 10)
	return allowance, nil
}

// ApproveAllowance approves MOR token allowance for a spender.
func (c *ApiGatewayClient) ApproveAllowance(ctx context.Context, spender string, amount *big.Int) (*TransactionResponse, error) {
	endpoint := fmt.Sprintf("/blockchain/approve?spender=%s&amount=%s", spender, amount.String())
	var result TransactionResponse
	err := c.postRequest(ctx, endpoint, nil, &result)
	return &result, err
}

// CreateWallet sets up the wallet using a private key.
func (c *ApiGatewayClient) CreateWallet(ctx context.Context, privateKey string) (*WalletResponse, error) {
	req := WalletRequest{PrivateKey: privateKey}
	var result WalletResponse
	err := c.postRequest(ctx, "/wallet/privateKey", &req, &result)
	return &result, err
}

// GetWallet retrieves the wallet address.
func (c *ApiGatewayClient) GetWallet(ctx context.Context) (*WalletResponse, error) {
	var result WalletResponse
	err := c.getRequest(ctx, "/wallet", &result)
	return &result, err
}

// GetBalance retrieves ETH and MOR balances of the user.
func (c *ApiGatewayClient) GetBalance(ctx context.Context) (ethBalance, morBalance *big.Int, err error) {
	var result struct {
		ETH string `json:"ETH"`
		MOR string `json:"MOR"`
	}
	err = c.getRequest(ctx, "/blockchain/balance", &result)
	if err != nil {
		return nil, nil, err
	}
	ethBalance = new(big.Int)
	ethBalance.SetString(result.ETH, 10)
	morBalance = new(big.Int)
	morBalance.SetString(result.MOR, 10)
	return ethBalance, morBalance, nil
}

// requestChatCompletionStream handles streaming responses for chat completions.
func (c *ApiGatewayClient) requestChatCompletionStream(ctx context.Context, endpoint string, request *openai.ChatCompletionRequest, callback CompletionCallback, modelID string, sessionID string) error {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to encode request: %v", err)
	}

	u, err := url.Parse(c.BaseURL + endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	if sessionID != "" {
		req.Header.Set("session_id", sessionID)
	} else if modelID != "" {
		req.Header.Set("model_id", modelID)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading stream: %v", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == "data: [DONE]" {
			break
		}

		if strings.HasPrefix(line, "data: ") {
			data := line[6:] // Skip the "data: " prefix
			var completion openai.ChatCompletionStreamResponse
			if err := json.Unmarshal([]byte(data), &completion); err != nil {
				fmt.Printf("Error decoding response: %v\n", err)
				continue
			}

			callback(&completion)
		}
	}

	return nil
}

// ModelDeregister sends a request to deregister a model.
func (c *ApiGatewayClient) ModelDeregister(ctx context.Context, modelId string) error {
	endpoint := fmt.Sprintf("/blockchain/models/%s/deregister", modelId)
	err := c.postRequest(ctx, endpoint, nil, nil)
	return err
}

// ModelSetMinStake sends a request to set the minimum stake required for models.
func (c *ApiGatewayClient) ModelSetMinStake(ctx context.Context, minStake *big.Int) error {
	req := struct {
		MinStake *big.Int `json:"minStake"`
	}{MinStake: minStake}
	endpoint := "/blockchain/models/minstake"
	err := c.postRequest(ctx, endpoint, req, nil)
	return err
}

// ModelMinStake retrieves the current minimum stake required for models.
func (c *ApiGatewayClient) ModelMinStake(ctx context.Context) (*big.Int, error) {
	endpoint := "/blockchain/models/minstake"
	var result struct {
		MinStake string `json:"minStake"`
	}
	err := c.getRequest(ctx, endpoint, &result)
	if err != nil {
		return nil, err
	}
	minStake := new(big.Int)
	minStake.SetString(result.MinStake, 10)
	return minStake, nil
}

// ModelStats retrieves the statistics of a model.
func (c *ApiGatewayClient) ModelStats(ctx context.Context, modelId string) (*ModelStats, error) {
	endpoint := fmt.Sprintf("/blockchain/models/%s/stats", modelId)
	var result ModelStats
	err := c.getRequest(ctx, endpoint, &result)
	return &result, err
}

// ModelResetStats sends a request to reset the statistics of a model.
func (c *ApiGatewayClient) ModelResetStats(ctx context.Context, modelId string) error {
	endpoint := fmt.Sprintf("/blockchain/models/%s/resetstats", modelId)
	err := c.postRequest(ctx, endpoint, nil, nil)
	return err
}

// ModelExists checks if a model exists.
func (c *ApiGatewayClient) ModelExists(ctx context.Context, modelId string) (bool, error) {
	endpoint := fmt.Sprintf("/blockchain/models/%s/exists", modelId)
	var result struct {
		Exists bool `json:"exists"`
	}
	err := c.getRequest(ctx, endpoint, &result)
	return result.Exists, err
}
