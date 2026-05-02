package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RepoConfig mirrors the structure of services/scout-go/configs/repos.yaml.
type RepoConfig struct {
	Repos               []string `yaml:"repos"`
	PollIntervalSeconds int      `yaml:"poll_interval_seconds"`
}

// repoConfigPath computes the path to repos.yaml relative to a service's Dir.
// It expects the given baseDir to be the working directory of the Scout service
// (e.g. "../../scout-go").
func repoConfigPath(baseDir string) string {
	return filepath.Join(baseDir, "configs", "repos.yaml")
}

// loadRepoConfig reads the repo watch list from the YAML file.
func loadRepoConfig(path string) (*RepoConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read repos config: %w", err)
	}
	var cfg RepoConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse repos config: %w", err)
	}
	if cfg.PollIntervalSeconds == 0 {
		cfg.PollIntervalSeconds = 60
	}
	return &cfg, nil
}

// saveRepoConfig writes the repo watch list to the YAML file atomically.
func saveRepoConfig(path string, cfg *RepoConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal repos config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write repos config tmp: %w", err)
	}
	return os.Rename(tmp, path)
}
