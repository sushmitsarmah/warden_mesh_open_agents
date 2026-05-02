package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Registry holds multiple LLM clients so callers can route by sensitivity.
type Registry struct {
	Default *OpenAICompatibleClient // general tasks
	Sealed  *OpenAICompatibleClient // sensitive tasks → 0G Compute TEE
}

func (r *Registry) Complete(ctx context.Context, system, user string) (string, error) {
	return r.Default.Complete(ctx, system, user)
}

func (r *Registry) CompleteSealed(ctx context.Context, system, user string) (string, error) {
	if r.Sealed == nil {
		slog.Warn("sealed client not configured, falling back to default")
		return r.Default.Complete(ctx, system, user)
	}
	return r.Sealed.Complete(ctx, system, user)
}

type Client interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

// Provider hints for framework-level routing and UI display.
type Provider string

const (
	ProviderOpenAI     Provider = "openai"
	ProviderAnthropic  Provider = "anthropic"
	ProviderOllama     Provider = "ollama"
	ProviderZeroG      Provider = "0g"
	ProviderOpenRouter Provider = "openrouter"
	ProviderChutes     Provider = "chutes"
	ProviderCustom     Provider = "custom"
)

// OpenAICompatibleClient supports any OpenAI-compatible API endpoint.
type OpenAICompatibleClient struct {
	baseURL    string
	apiKey     string
	model      string
	provider   Provider
	httpClient *http.Client
}

// NewOpenAICompatibleClient creates a client with auto-detected provider.
// Backward-compatible 3-arg signature.
func NewOpenAICompatibleClient(baseURL, apiKey, model string) *OpenAICompatibleClient {
	return NewOpenAICompatibleClientWithProvider(baseURL, apiKey, model, "")
}

// NewOpenAICompatibleClientWithProvider creates a client with an explicit provider hint.
// If provider is empty, it is auto-detected from baseURL and apiKey.
// baseURL examples:
//   - OpenAI:      "https://api.openai.com/v1"
//   - Anthropic:   "https://api.anthropic.com/v1"
//   - Ollama:      "http://localhost:11434/v1"
//   - OpenRouter:  "https://openrouter.ai/api/v1"
//   - Chutes:      "https://api.chutes.ai/v1"
//   - 0G Compute:  "https://<provider-service-url>/v1/proxy"
func NewOpenAICompatibleClientWithProvider(baseURL, apiKey, model string, provider Provider) *OpenAICompatibleClient {
	if model == "" {
		model = "gpt-4o-mini"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if provider == "" {
		provider = detectProvider(baseURL, apiKey)
	}
	return &OpenAICompatibleClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		provider: provider,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Provider returns the detected provider type.
func (c *OpenAICompatibleClient) Provider() Provider { return c.provider }

// BaseURL returns the configured endpoint.
func (c *OpenAICompatibleClient) BaseURL() string { return c.baseURL }

// SetTimeout allows runtime adjustment of HTTP timeout.
func (c *OpenAICompatibleClient) SetTimeout(d time.Duration) {
	c.httpClient.Timeout = d
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

func (c *OpenAICompatibleClient) Complete(ctx context.Context, system, user string) (string, error) {
	messages := []chatMessage{}
	if system != "" {
		messages = append(messages, chatMessage{Role: "system", Content: system})
	}
	messages = append(messages, chatMessage{Role: "user", Content: user})

	reqBody := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   4096,
	}

	slog.Info("llm.complete",
		"provider", c.provider,
		"model", c.model,
		"base_url", c.baseURL,
		"user_len", len(user),
	)

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Provider-specific headers
	switch c.provider {
	case ProviderAnthropic:
		httpReq.Header.Set("anthropic-version", "2023-06-01")
	case ProviderOpenRouter:
		httpReq.Header.Set("HTTP-Referer", "https://github.com/local/swarm")
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
		"provider", c.provider,
		"tokens_used", chatResp.Usage.TotalTokens,
		"response_len", len(content),
		"finish_reason", chatResp.Choices[0].FinishReason,
	)

	return content, nil
}

func detectProvider(baseURL, apiKey string) Provider {
	if strings.Contains(baseURL, "0g") || strings.HasPrefix(apiKey, "app-sk-") {
		return ProviderZeroG
	}
	if strings.Contains(baseURL, "anthropic") || strings.HasPrefix(apiKey, "sk-ant-") {
		return ProviderAnthropic
	}
	if strings.Contains(baseURL, "openrouter") {
		return ProviderOpenRouter
	}
	if strings.Contains(baseURL, "chutes") {
		return ProviderChutes
	}
	if strings.Contains(baseURL, "localhost") || strings.Contains(baseURL, "127.0.0.1") {
		return ProviderOllama
	}
	if strings.Contains(baseURL, "openai") && strings.HasPrefix(apiKey, "sk-") {
		return ProviderOpenAI
	}
	return ProviderCustom
}
