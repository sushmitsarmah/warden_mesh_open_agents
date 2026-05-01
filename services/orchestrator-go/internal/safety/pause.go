package safety

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/local/swarm/orchestrator/internal/inft"
)

type PauseChecker struct {
	client   *inft.Client
	paused   bool
	interval time.Duration
}

func NewPauseChecker(client *inft.Client) *PauseChecker {
	return &PauseChecker{
		client:   client,
		interval: 30 * time.Second,
	}
}

func (p *PauseChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			paused, err := p.client.IsPaused()
			if err != nil {
				slog.Error("pause check", "err", err)
				continue
			}
			p.paused = paused
		}
	}
}

func (p *PauseChecker) IsPaused() bool {
	return p.paused
}
