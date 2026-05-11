package scout

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

	"github.com/google/uuid"
	"github.com/local/swarm/scout/pkg/messages"
)

const fortaGraphQLURL = "https://api.forta.network/graphql"

// FortaWatcher polls the Forta Network GraphQL API for high-severity alerts
// on watched contract addresses and converts them to Scout targets.
type FortaWatcher struct {
	apiKey       string
	out          chan<- messages.Target
	contracts    []string
	pollInterval time.Duration
	lastSeen     map[string]bool // alertId dedup
	bountyType   string
}

func NewFortaWatcher(apiKey string, out chan<- messages.Target, contracts []string, pollInterval time.Duration, bountyType string) *FortaWatcher {
	return &FortaWatcher{
		apiKey:       apiKey,
		out:          out,
		contracts:    contracts,
		pollInterval: pollInterval,
		lastSeen:     make(map[string]bool),
		bountyType:   bountyType,
	}
}

func (w *FortaWatcher) Run(ctx context.Context) error {
	if w.apiKey == "" {
		slog.Info("forta watcher: FORTA_API_KEY not set, skipping")
		<-ctx.Done()
		return nil
	}
	if len(w.contracts) == 0 {
		slog.Info("forta watcher: no contracts configured, skipping")
		<-ctx.Done()
		return nil
	}

	slog.Info("forta watcher started", "contracts", len(w.contracts))

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Fetch once immediately, then on each tick
	w.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *FortaWatcher) poll(ctx context.Context) {
	alerts, err := w.fetchAlerts(ctx)
	if err != nil {
		slog.Warn("forta watcher: fetch failed", "err", err)
		return
	}

	new := 0
	for _, a := range alerts {
		if w.lastSeen[a.AlertID] {
			continue
		}
		w.lastSeen[a.AlertID] = true

		// Map Forta severity to priority score
		priority := severityToPriority(a.Severity)

		// Use the first matched contract address as the target address
		addr := ""
		for _, candidate := range a.Addresses {
			for _, watched := range w.contracts {
				if strings.EqualFold(candidate, watched) {
					addr = candidate
					break
				}
			}
			if addr != "" {
				break
			}
		}
		if addr == "" && len(a.Addresses) > 0 {
			addr = a.Addresses[0]
		}

		chainID := 1
		if a.Source.Block.ChainID != 0 {
			chainID = a.Source.Block.ChainID
		}

		t := messages.Target{
			ID:           uuid.NewString(),
			BountyType:   w.bountyType,
			Kind:         "forta-alert",
			ChainID:      chainID,
			Address:      addr,
			TxHash:       a.Source.TransactionHash,
			SourceURL:    fmt.Sprintf("https://app.forta.network/alert/%s", a.AlertID),
			DiscoveredAt: time.Now().UTC(),
			Priority:     priority,
		}

		w.out <- t
		slog.Info("forta alert emitted",
			"alert_id", a.AlertID,
			"name", a.Name,
			"severity", a.Severity,
			"address", addr,
			"priority", priority,
		)
		new++
	}

	if new > 0 {
		slog.Info("forta poll complete", "new_alerts", new)
	}
}

// ---- GraphQL types ---- //

type fortaQuery struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type fortaResponse struct {
	Data struct {
		Alerts struct {
			Alerts []fortaAlert `json:"alerts"`
		} `json:"alerts"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type fortaAlert struct {
	AlertID     string   `json:"alertId"`
	Name        string   `json:"name"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	Addresses   []string `json:"addresses"`
	Source      struct {
		TransactionHash string `json:"transactionHash"`
		Block           struct {
			Number  int `json:"number"`
			ChainID int `json:"chainId"`
		} `json:"block"`
	} `json:"source"`
	CreatedAt string `json:"createdAt"`
}

const alertsQuery = `
query Alerts($input: AlertsInput!) {
  alerts(input: $input) {
    alerts {
      alertId
      name
      severity
      description
      addresses
      source {
        transactionHash
        block { number chainId }
      }
      createdAt
    }
  }
}`

func (w *FortaWatcher) fetchAlerts(ctx context.Context) ([]fortaAlert, error) {
	// Fetch alerts from the last poll window — Forta returns most recent first
	since := time.Now().Add(-w.pollInterval * 2).UTC().Format(time.RFC3339)

	body := fortaQuery{
		Query: alertsQuery,
		Variables: map[string]any{
			"input": map[string]any{
				"severities":   []string{"CRITICAL", "HIGH"},
				"addresses":    w.contracts,
				"createdSince": since,
				"first":        50,
			},
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fortaGraphQLURL, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if w.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+w.apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result fortaResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		cap := 300
		if len(raw) < cap {
			cap = len(raw)
		}
		return nil, fmt.Errorf("unmarshal: %w — body: %s", err, string(raw[:cap]))
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("forta API error: %s", result.Errors[0].Message)
	}

	return result.Data.Alerts.Alerts, nil
}

func severityToPriority(s string) float64 {
	switch strings.ToUpper(s) {
	case "CRITICAL":
		return 95
	case "HIGH":
		return 80
	case "MEDIUM":
		return 60
	case "LOW":
		return 40
	default:
		return 30
	}
}
