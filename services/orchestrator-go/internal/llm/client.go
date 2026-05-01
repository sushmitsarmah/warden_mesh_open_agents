package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Client interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

// OpenAICompatibleClient supports any OpenAI-compatible API
type OpenAICompatibleClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
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
	} `json:"error,omitempty"`
}

// NewOpenAICompatibleClient creates a client that works with any OpenAI-compatible API
// baseURL examples:
//   - OpenAI: "https://api.openai.com/v1"
//   - Anthropic: "https://api.anthropic.com/v1" (requires anthropic-version header)
//   - Local (ollama): "http://localhost:11434/v1"
//   - Local (vllm): "http://localhost:8000/v1"
//   - Any other OpenAI-compatible endpoint
func NewOpenAICompatibleClient(baseURL, apiKey, model string) *OpenAICompatibleClient {
	if model == "" {
		model = "gpt-4o-mini" // reasonable default
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAICompatibleClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *OpenAICompatibleClient) Complete(ctx context.Context, system, user string) (string, error) {
	messages := []chatMessage{}
	if system != "" {
		messages = append(messages, chatMessage{
			Role:    "system",
			Content: system,
		})
	}
	messages = append(messages, chatMessage{
		Role:    "user",
		Content: user,
	})

	req := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.3, // Lower temperature for more deterministic code generation
		MaxTokens:   4096,
	}

	slog.Info("llm.complete", "model", c.model, "base_url", c.baseURL, "user_len", len(user))

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Anthropic-specific header if using Anthropic endpoint
	if c.baseURL == "https://api.anthropic.com/v1" {
		httpReq.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s (%s)", chatResp.Error.Message, chatResp.Error.Type)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := chatResp.Choices[0].Message.Content
	slog.Info("llm.complete.done",
		"tokens_used", chatResp.Usage.TotalTokens,
		"response_len", len(content),
		"finish_reason", chatResp.Choices[0].FinishReason,
	)

	return content, nil
}
