package scout

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/local/swarm/scout/pkg/messages"
)

const stateFile = ".scout-state.json"

type GitHubWatcher struct {
	token string
	out   chan<- messages.Target
}

func NewGitHubWatcher(token string, out chan<- messages.Target) *GitHubWatcher {
	return &GitHubWatcher{token: token, out: out}
}

type repoState struct {
	LastSeenSha string `json:"lastSeenSha"`
}

func (w *GitHubWatcher) Run(ctx context.Context) error {
	// Placeholder: poll GitHub commits for watched repos.
	// In a real implementation we'd use go-github/v62.
	repos := []string{"aave/aave-v3-core", "Uniswap/v4-core", "compound-finance/compound-protocol"}
	state := loadState()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		for _, repo := range repos {
			last := state[repo]
			// TODO: real GitHub API call
			_ = last
			_, _ = w.token, repo
			// If new commits exist, emit a target
			_ = emitMockTarget(repo, w.out)
		}
		time.Sleep(60 * time.Second)
	}
}

func emitMockTarget(repo string, out chan<- messages.Target) error {
	t := messages.Target{
		ID:           uuid.NewString(),
		Kind:         messages.TargetGitHub,
		Repo:         repo,
		CommitSha:    "abc123", // placeholder
		DiscoveredAt: time.Now().UTC(),
		Priority:     Score(messages.Target{Kind: messages.TargetGitHub, DiscoveredAt: time.Now().UTC()}),
	}
	out <- t
	return nil
}

func loadState() map[string]string {
	state := make(map[string]string)
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return state
	}
	_ = json.Unmarshal(data, &state)
	return state
}

func saveState(state map[string]string) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0644)
}
