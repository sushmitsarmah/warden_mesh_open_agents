package llm

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Client interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

type AnthropicClient struct {
	apiKey string
	model  string
}

func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	if model == "" {
		model = "claude-sonnet-4-20250501"
	}
	return &AnthropicClient{apiKey: apiKey, model: model}
}

func (c *AnthropicClient) Complete(ctx context.Context, system, user string) (string, error) {
	// Placeholder: integrate real Anthropic SDK
	_ = system
	slog.Info("llm.complete", "model", c.model, "user_len", len(user))
	// Retry loop is handled by caller
	return "", fmt.Errorf("llm not implemented")
}
