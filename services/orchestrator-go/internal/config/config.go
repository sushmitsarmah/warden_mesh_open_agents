package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MainnetRPC   string
	LLMKey       string
	LLMModel     string
	SwarmPrivKey string
}

func Load() (*Config, error) {
	_ = godotenv.Load("../../.env")
	c := &Config{
		MainnetRPC:   os.Getenv("MAINNET_RPC_URL"),
		LLMKey:       os.Getenv("ANTHROPIC_API_KEY"),
		LLMModel:     os.Getenv("LLM_MODEL"),
		SwarmPrivKey: os.Getenv("SWARM_PRIVATE_KEY"),
	}
	if c.LLMKey == "" {
		c.LLMKey = os.Getenv("OPENAI_API_KEY")
	}
	if c.MainnetRPC == "" {
		return nil, fmt.Errorf("MAINNET_RPC_URL not set")
	}
	return c, nil
}
