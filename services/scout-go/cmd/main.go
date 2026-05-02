package main

import (
	"context"
	"log/slog"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/local/swarm/scout/internal/config"
	"github.com/local/swarm/scout/internal/scout"
	"github.com/local/swarm/scout/pkg/axl"
	"github.com/local/swarm/scout/pkg/messages"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		return
	}

	node, err := axl.NewNode(axl.LoadBootstrapFromEnv())
	if err != nil {
		slog.Error("axl", "err", err)
		return
	}
	defer node.Close()

	out := make(chan messages.Target, 64)
	deduper := scout.NewDeduper()

	go func() {
		for t := range out {
			t.Priority = scout.Score(t)
			if !deduper.IsNew(t) {
				slog.Info("deduped target", "id", t.ID)
				continue
			}
			if err := axl.PublishTarget(node, "targets/discovered", t); err != nil {
				slog.Error("publish", "err", err)
			} else {
				slog.Info("published target", "id", t.ID, "kind", t.Kind, "priority", t.Priority)
			}
		}
	}()

	var wg sync.WaitGroup

	// Mempool watcher (on-chain targets)
	wg.Add(1)
	go func() {
		defer wg.Done()
		w := scout.NewMempoolWatcher(cfg.SepoliaRPC, out)
		if err := w.Run(ctx); err != nil {
			slog.Error("mempool watcher", "err", err)
		}
	}()

	// GitHub commit watcher (repo targets)
	wg.Add(1)
	go func() {
		defer wg.Done()
		repoCfg, err := scout.LoadRepoConfig("")
		if err != nil {
			slog.Error("load repo config", "err", err)
			return
		}
		gh := scout.NewGitHubWatcher(
			cfg.GitHubToken,
			out,
			repoCfg.Repos,
			time.Duration(repoCfg.PollIntervalSeconds)*time.Second,
		)
		if err := gh.Run(ctx); err != nil {
			slog.Error("github watcher", "err", err)
		}
	}()

	wg.Wait()
}
