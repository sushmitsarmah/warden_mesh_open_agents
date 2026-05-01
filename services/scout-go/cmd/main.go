package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os/signal"
	"syscall"

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

	w := scout.NewMempoolWatcher(cfg.SepoliaRPC, out)
	if err := w.Run(ctx); err != nil {
		slog.Error("watcher", "err", err)
	}
}
