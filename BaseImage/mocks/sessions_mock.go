package mocks

import (
	"time"
)

type MockSessionManager struct {
	GetModelByHandleFn func(modelHandle string) (*ModelInfo, error)
	CreateSessionFn    func(modelId string, stakeAmount string) (*SessionResponse, error)
	SendChatMessageFn  func(sessionToken string, modelId string, message string) (*ChatResponse, error)
}

type ModelInfo struct {
	ID          string
	Name        string
	Endpoint    string
	Description string
}

type SessionResponse struct {
	SessionToken string
	ExpiresAt    time.Time
}

type ChatResponse struct {
	Response string
}

func NewMockSessionManager() *MockSessionManager {
	return &MockSessionManager{
		GetModelByHandleFn: func(modelHandle string) (*ModelInfo, error) {
			return &ModelInfo{
				ID:   "test-model-id",
				Name: modelHandle,
			}, nil
		},
		CreateSessionFn: func(modelId string, stakeAmount string) (*SessionResponse, error) {
			return &SessionResponse{
				SessionToken: "test-session-token",
				ExpiresAt:    time.Now().Add(1 * time.Hour),
			}, nil
		},
		SendChatMessageFn: func(sessionToken string, modelId string, message string) (*ChatResponse, error) {
			return &ChatResponse{
				Response: "Test response",
			}, nil
		},
	}
} 