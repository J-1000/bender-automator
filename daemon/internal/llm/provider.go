package llm

import (
	"context"
	"errors"
)

var (
	ErrProviderNotFound  = errors.New("provider not found")
	ErrProviderDisabled  = errors.New("provider is disabled")
	ErrVisionNotSupport  = errors.New("provider does not support vision")
	ErrNoContent         = errors.New("no content in response")
)

// Provider is the interface for LLM providers
type Provider interface {
	Name() string
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	CompleteWithVision(ctx context.Context, req VisionRequest) (*CompletionResponse, error)
	SupportsVision() bool
}

// CompletionRequest represents a text completion request
type CompletionRequest struct {
	Model       string
	Messages    []Message
	Temperature float64
	MaxTokens   int
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// VisionRequest extends CompletionRequest with image support
type VisionRequest struct {
	CompletionRequest
	Images []Image
}

// Image represents an image for vision requests
type Image struct {
	Data     []byte
	MimeType string
	URL      string // Alternative to Data
}

// CompletionResponse represents the response from an LLM
type CompletionResponse struct {
	Content      string
	FinishReason string
	Usage        Usage
}

// Usage tracks token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
