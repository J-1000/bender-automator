package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/user/bender/internal/config"
)

// Router manages multiple LLM providers and routes requests
type Router struct {
	providers       map[string]Provider
	defaultProvider string
	visionProvider  string
	mu              sync.RWMutex
}

// NewRouter creates a new provider router from config
func NewRouter(cfg *config.LLMConfig) (*Router, error) {
	r := &Router{
		providers:       make(map[string]Provider),
		defaultProvider: cfg.DefaultProvider,
	}

	for name, provCfg := range cfg.Providers {
		if !provCfg.Enabled {
			continue
		}

		var provider Provider
		switch name {
		case "ollama":
			provider = NewOllamaProvider(OllamaConfig{
				BaseURL:        provCfg.BaseURL,
				Model:          provCfg.Model,
				TimeoutSeconds: provCfg.TimeoutSeconds,
			})
		case "openai":
			provider = NewOpenAIProvider(OpenAIConfig{
				APIKey:         provCfg.APIKey,
				Model:          provCfg.Model,
				VisionModel:    provCfg.VisionModel,
				TimeoutSeconds: provCfg.TimeoutSeconds,
			})
		case "anthropic":
			provider = NewAnthropicProvider(AnthropicConfig{
				APIKey:         provCfg.APIKey,
				Model:          provCfg.Model,
				TimeoutSeconds: provCfg.TimeoutSeconds,
			})
		default:
			continue
		}

		r.providers[name] = provider
	}

	if len(r.providers) == 0 {
		return nil, fmt.Errorf("no providers enabled")
	}

	if _, ok := r.providers[r.defaultProvider]; !ok {
		// Fall back to first available provider
		for name := range r.providers {
			r.defaultProvider = name
			break
		}
	}

	return r, nil
}

// GetProvider returns a specific provider by name
func (r *Router) GetProvider(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if name == "" {
		name = r.defaultProvider
	}

	provider, ok := r.providers[name]
	if !ok {
		return nil, ErrProviderNotFound
	}

	return provider, nil
}

// GetVisionProvider returns a vision-capable provider
func (r *Router) GetVisionProvider(preferred string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try preferred first
	if preferred != "" {
		if p, ok := r.providers[preferred]; ok && p.SupportsVision() {
			return p, nil
		}
	}

	// Try default
	if p, ok := r.providers[r.defaultProvider]; ok && p.SupportsVision() {
		return p, nil
	}

	// Find any vision provider
	for _, p := range r.providers {
		if p.SupportsVision() {
			return p, nil
		}
	}

	return nil, ErrVisionNotSupport
}

// Complete sends a completion request to the default provider
func (r *Router) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	provider, err := r.GetProvider("")
	if err != nil {
		return nil, err
	}
	return provider.Complete(ctx, req)
}

// CompleteWithVision sends a vision request to a vision-capable provider
func (r *Router) CompleteWithVision(ctx context.Context, req VisionRequest, preferredProvider string) (*CompletionResponse, error) {
	provider, err := r.GetVisionProvider(preferredProvider)
	if err != nil {
		return nil, err
	}
	return provider.CompleteWithVision(ctx, req)
}

// ListProviders returns the names of all enabled providers
func (r *Router) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// DefaultProviderName returns the name of the default provider
func (r *Router) DefaultProviderName() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultProvider
}
