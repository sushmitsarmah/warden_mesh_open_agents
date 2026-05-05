package scout

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/google/uuid"
	"github.com/local/swarm/scout/pkg/messages"
	"gopkg.in/yaml.v3"
)

const (
	stateFile     = ".scout-state.json"
	defaultConfig = "configs/repos.yaml"
)

// WatchConfig mirrors the structure of configs/repos.yaml.
type WatchConfig struct {
	Repos               []string `yaml:"repos"`
	Contracts           []string `yaml:"contracts"`
	Wallets             []string `yaml:"wallets"`
	BountyType          string   `yaml:"bounty_type"`
	PollIntervalSeconds int      `yaml:"poll_interval_seconds"`
}

// GitHubWatcher polls configured repositories for new commits and emits targets.
type GitHubWatcher struct {
	client       *github.Client
	out          chan<- messages.Target
	repos        []string
	pollInterval time.Duration
	stateFile    string
	bountyType   string
}

// LoadRepoConfig reads the YAML watch config from path (falls back to defaultConfig).
func LoadRepoConfig(path string) (*WatchConfig, error) {
	if path == "" {
		path = defaultConfig
	}
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

// NewGitHubWatcher creates a watcher that polls repos for new commits.
// token may be empty for unauthenticated requests (lower rate limits).
func NewGitHubWatcher(token string, out chan<- messages.Target, repos []string, pollInterval time.Duration, bountyType string) *GitHubWatcher {
	client := github.NewClient(nil)
	if token != "" {
		client = client.WithAuthToken(token)
	}
	return &GitHubWatcher{
		client:       client,
		out:          out,
		repos:        repos,
		pollInterval: pollInterval,
		stateFile:    stateFile,
		bountyType:   bountyType,
	}
}

// Run polls GitHub commits until ctx is cancelled.
func (w *GitHubWatcher) Run(ctx context.Context) error {
	state := loadState(w.stateFile)

	// Bootstrap: if no state exists, seed with the latest commit so we don't
	// flood the mesh with historical commits on first run.
	if len(state) == 0 {
		for _, repo := range w.repos {
			owner, name := splitRepo(repo)
			if owner == "" || name == "" {
				slog.Warn("invalid repo slug, skipping", "repo", repo)
				continue
			}
			latest, _, err := w.client.Repositories.ListCommits(ctx, owner, name, &github.CommitsListOptions{ListOptions: github.ListOptions{PerPage: 1}})
			if err != nil {
				slog.Warn("github bootstrap failed", "repo", repo, "err", err)
				continue
			}
			if len(latest) > 0 && latest[0].SHA != nil {
				state[repo] = *latest[0].SHA
				slog.Info("github bootstrapped", "repo", repo, "sha", *latest[0].SHA)
			}
		}
		_ = saveState(w.stateFile, state)
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for _, repo := range w.repos {
				owner, name := splitRepo(repo)
				if owner == "" || name == "" {
					slog.Warn("invalid repo slug, skipping", "repo", repo)
					continue
				}

				commits, resp, err := w.client.Repositories.ListCommits(ctx, owner, name, &github.CommitsListOptions{
					ListOptions: github.ListOptions{PerPage: 10},
				})
				if err != nil {
					slog.Warn("github list commits failed", "repo", repo, "err", err)
					continue
				}

				// Respect rate limits: if remaining is low, back off.
				if resp != nil && resp.Rate.Remaining < 5 {
					slog.Warn("github rate limit low, backing off",
						"remaining", resp.Rate.Remaining,
						"reset", resp.Rate.Reset,
					)
					continue
				}

				lastSeen := state[repo]
				newCommits := 0
				for _, c := range commits {
					if c.SHA == nil || *c.SHA == lastSeen {
						break // reached previously known commit
					}
					newCommits++

					sha := *c.SHA
					msg := ""
					if c.Commit != nil && c.Commit.Message != nil {
						msg = *c.Commit.Message
					}

				t := messages.Target{
					ID:           uuid.NewString(),
					BountyType:   w.bountyType,
					Kind:         "github",
					Repo:         repo,
					CommitSha:    sha,
					SourceURL:    fmt.Sprintf("https://github.com/%s/commit/%s", repo, sha),
					DiscoveredAt: time.Now().UTC(),
					Priority:     Score(messages.Target{Kind: "github", DiscoveredAt: time.Now().UTC()}),
					TVLUsd:       0,
				}
					w.out <- t
					slog.Info("github target emitted",
						"repo", repo,
						"sha", sha,
						"msg_preview", truncate(msg, 60),
					)
				}

				if len(commits) > 0 && commits[0].SHA != nil {
					state[repo] = *commits[0].SHA
					_ = saveState(w.stateFile, state)
				}

				if newCommits > 0 {
					slog.Info("github poll done", "repo", repo, "new_commits", newCommits)
				}
			}
		}
	}
}

// splitRepo converts "owner/name" into (owner, name). Returns empty strings on invalid input.
func splitRepo(repo string) (string, string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// truncate shortens a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// loadState reads the persisted last-seen commit SHAs.
func loadState(path string) map[string]string {
	state := make(map[string]string)
	data, err := os.ReadFile(path)
	if err != nil {
		return state
	}
	_ = json.Unmarshal(data, &state)
	return state
}

// saveState persists the last-seen commit SHAs.
func saveState(path string, state map[string]string) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
