package safety

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// ScopeMonitor polls the Firedancer changelog and known-issues trackers for mid-contest updates.
// Per assets.md:65-69, bug fixes may be applied mid-contest, invalidating cached findings.
type ScopeMonitor struct {
	mu             sync.Mutex
	knownIssues    []string // updated list of known-issue fingerprints
	lastChecked    time.Time
	checkInterval  time.Duration
	changelogURL   string
	knownIssuesURL map[string]string // tracker name -> URL
}

func NewScopeMonitor() *ScopeMonitor {
	return &ScopeMonitor{
		checkInterval: 24 * time.Hour,
		changelogURL:  "https://github.com/firedancer-io/firedancer/releases",
		knownIssuesURL: map[string]string{
			"bundle":        "https://github.com/firedancer-io/firedancer/issues?q=label%3Abundle",
			"consensus":     "https://github.com/firedancer-io/firedancer/issues?q=label%3Aconsensus",
			"gossip":        "https://github.com/firedancer-io/firedancer/issues?q=label%3Agossip",
			"runtime":       "https://github.com/firedancer-io/firedancer/issues?q=label%3Aruntime",
			"rpc":           "https://github.com/firedancer-io/firedancer/issues?q=label%3Arpc",
			"repair":        "https://github.com/firedancer-io/firedancer/issues?q=label%3Arepair",
			"sandboxing":    "https://github.com/firedancer-io/firedancer/issues?q=label%3Asandboxing",
			"shred":         "https://github.com/firedancer-io/firedancer/issues?q=label%3Ashred",
			"sign":          "https://github.com/firedancer-io/firedancer/issues?q=label%3Asign",
			"quic/net":      "https://github.com/firedancer-io/firedancer/issues?q=label%3Aquic",
		},
	}
}

// Run starts the background polling loop.
func (m *ScopeMonitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	slog.Info("scope_monitor: starting", "interval", m.checkInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.check(ctx)
		}
	}
}

// check polls all known-issue trackers and the changelog.
func (m *ScopeMonitor) check(ctx context.Context) {
	slog.Info("scope_monitor: checking for mid-contest updates")

	// In a production system, this would:
	// 1. Fetch the GitHub API for new issue labels matching known-issue patterns
	// 2. Fetch the changelog/releases page for new fixes
	// 3. Check if the v1.0 branch has new commits
	// 4. Invalidate cached finding fingerprints

	// For now, we log that the check ran
	m.mu.Lock()
	m.lastChecked = time.Now()
	m.mu.Unlock()

	slog.Info("scope_monitor: check complete", "last_checked", m.lastChecked)
}

// IsKnownIssue checks if a finding fingerprint matches a known issue.
func (m *ScopeMonitor) IsKnownIssue(fingerprint string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, ki := range m.knownIssues {
		if ki == fingerprint {
			return true
		}
	}
	return false
}

// fetchURL is a helper for fetching tracker data.
func (m *ScopeMonitor) fetchURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}

	return string(body), nil
}

// UpdateKnownIssues updates the internal known issues list from the fetched data.
func (m *ScopeMonitor) UpdateKnownIssues(ctx context.Context) error {
	for name, url := range m.knownIssuesURL {
		data, err := m.fetchURL(ctx, url)
		if err != nil {
			slog.Warn("scope_monitor: failed to fetch tracker", "name", name, "err", err)
			continue
		}

		// Parse the data and extract fingerprints
		// This is a placeholder — real implementation would parse GitHub API responses
		_ = data
		slog.Debug("scope_monitor: fetched tracker", "name", name)
	}

	return nil
}

// LogCheckpoint logs the current known-issue state for auditing.
func (m *ScopeMonitor) LogCheckpoint() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, _ := json.Marshal(map[string]interface{}{
		"known_issues":   len(m.knownIssues),
		"last_checked":   m.lastChecked,
		"check_interval": m.checkInterval.String(),
	})
	slog.Info("scope_monitor: checkpoint", "data", string(data))
}
