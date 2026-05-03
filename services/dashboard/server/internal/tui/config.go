package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// WatchConfig mirrors services/scout-go/configs/repos.yaml.
type WatchConfig struct {
	Repos               []string `yaml:"repos"`
	Contracts           []string `yaml:"contracts"`
	Wallets             []string `yaml:"wallets"`
	PollIntervalSeconds int      `yaml:"poll_interval_seconds"`
}

func watchConfigPath(baseDir string) string {
	return filepath.Join(baseDir, "configs", "repos.yaml")
}

func loadRepoConfig(path string) (*WatchConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read watch config: %w", err)
	}
	var cfg WatchConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse watch config: %w", err)
	}
	if cfg.PollIntervalSeconds == 0 {
		cfg.PollIntervalSeconds = 60
	}
	return &cfg, nil
}

func saveRepoConfig(path string, cfg *WatchConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal watch config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write watch config tmp: %w", err)
	}
	return os.Rename(tmp, path)
}
