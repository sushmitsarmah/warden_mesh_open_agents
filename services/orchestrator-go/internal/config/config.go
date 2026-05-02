package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	MainnetRPC         string
	LLMBaseURL         string
	LLMKey             string
	LLMModel           string
	SwarmPrivKey       string
	OGRpcURL           string
	OGPrivateKey       string
	OGINFTAddress      string
	ReportServerBaseURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load("../../.env")

	// Try to get API key from various env vars
	llmKey := os.Getenv("LLM_API_KEY")
	if llmKey == "" {
		llmKey = os.Getenv("OPENAI_API_KEY")
	}
	if llmKey == "" {
		llmKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	// Get base URL (if not set, will default based on API key type)
	baseURL := os.Getenv("LLM_BASE_URL")

	// Auto-detect base URL from API key if not explicitly set
	if baseURL == "" {
		baseURL = detectBaseURL(llmKey)
	}

	// Get model (if not set, will default in client)
	model := os.Getenv("LLM_MODEL")

	c := &Config{
		MainnetRPC:          os.Getenv("MAINNET_RPC_URL"),
		LLMBaseURL:          baseURL,
		LLMKey:              llmKey,
		LLMModel:            model,
		SwarmPrivKey:        os.Getenv("SWARM_PRIVATE_KEY"),
		OGRpcURL:            os.Getenv("OG_RPC_URL"),
		OGPrivateKey:        os.Getenv("OG_PRIVATE_KEY"),
		OGINFTAddress:       os.Getenv("OG_INFT_ADDRESS"),
		ReportServerBaseURL: os.Getenv("REPORT_SERVER_BASE_URL"),
	}

	if c.MainnetRPC == "" {
		return nil, fmt.Errorf("MAINNET_RPC_URL not set")
	}
	if c.LLMKey == "" {
		return nil, fmt.Errorf("no LLM API key found (set LLM_API_KEY, OPENAI_API_KEY, or ANTHROPIC_API_KEY)")
	}

	return c, nil
}

// detectBaseURL tries to auto-detect the correct base URL from the API key prefix
func detectBaseURL(apiKey string) string {
	if strings.HasPrefix(apiKey, "sk-ant-") {
		// Anthropic API key
		return "https://api.anthropic.com/v1"
	}
	if strings.HasPrefix(apiKey, "sk-") {
		// OpenAI API key
		return "https://api.openai.com/v1"
	}
	// Default to OpenAI for unknown formats
	return "https://api.openai.com/v1"
}
