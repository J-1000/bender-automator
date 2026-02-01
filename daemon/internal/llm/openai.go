package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAIProvider struct {
	apiKey      string
	model       string
	visionModel string
	client      *http.Client
}

type OpenAIConfig struct {
	APIKey         string
	Model          string
	VisionModel    string
	TimeoutSeconds int
}

func NewOpenAIProvider(cfg OpenAIConfig) *OpenAIProvider {
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	if cfg.VisionModel == "" {
		cfg.VisionModel = "gpt-4o"
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &OpenAIProvider{
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		visionModel: cfg.VisionModel,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) SupportsVision() bool {
	return true
}

type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []openaiContent
}

type openaiContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openaiImageURL `json:"image_url,omitempty"`
}

type openaiImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type openaiResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openaiMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]openaiMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openaiMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	openaiReq := openaiRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	return p.doRequest(ctx, openaiReq)
}

func (p *OpenAIProvider) CompleteWithVision(ctx context.Context, req VisionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.visionModel
	}

	messages := make([]openaiMessage, len(req.Messages))
	for i, m := range req.Messages {
		if i == len(req.Messages)-1 && m.Role == "user" && len(req.Images) > 0 {
			// Build content array with text and images
			content := []openaiContent{
				{Type: "text", Text: m.Content},
			}
			for _, img := range req.Images {
				var url string
				if img.URL != "" {
					url = img.URL
				} else if len(img.Data) > 0 {
					mimeType := img.MimeType
					if mimeType == "" {
						mimeType = "image/png"
					}
					url = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(img.Data))
				}
				if url != "" {
					content = append(content, openaiContent{
						Type:     "image_url",
						ImageURL: &openaiImageURL{URL: url, Detail: "auto"},
					})
				}
			}
			messages[i] = openaiMessage{
				Role:    m.Role,
				Content: content,
			}
		} else {
			messages[i] = openaiMessage{
				Role:    m.Role,
				Content: m.Content,
			}
		}
	}

	openaiReq := openaiRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	return p.doRequest(ctx, openaiReq)
}

func (p *OpenAIProvider) doRequest(ctx context.Context, req openaiRequest) (*CompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	var openaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("decode response (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if openaiResp.Error != nil {
		return nil, fmt.Errorf("openai error: %s", openaiResp.Error.Message)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, ErrNoContent
	}

	content, ok := openaiResp.Choices[0].Message.Content.(string)
	if !ok {
		return nil, ErrNoContent
	}

	return &CompletionResponse{
		Content:      content,
		FinishReason: openaiResp.Choices[0].FinishReason,
		Usage: Usage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
	}, nil
}
