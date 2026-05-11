package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SepoliaRPC  string
	GitHubToken string
	FortaAPIKey string
	MinTVLUsd   float64
}

func Load() (*Config, error) {
	_ = godotenv.Load("../../.env")
	c := &Config{
		SepoliaRPC:  os.Getenv("SEPOLIA_RPC_URL"),
		GitHubToken: os.Getenv("GITHUB_TOKEN"),
		FortaAPIKey: os.Getenv("FORTA_API_KEY"),
	}
	if c.SepoliaRPC == "" {
		return nil, fmt.Errorf("SEPOLIA_RPC_URL not set")
	}
	return c, nil
}
