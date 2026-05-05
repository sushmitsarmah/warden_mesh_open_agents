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

	// Load unified watch config (repos + contracts + wallets)
	watchCfg, err := scout.LoadRepoConfig("")
	if err != nil {
		slog.Error("load watch config", "err", err)
		// continue anyway — GitHub tag can use defaults if empty
	}

	var wg sync.WaitGroup

	bountyType := "solidity-evm"
	if watchCfg != nil && watchCfg.BountyType != "" {
		bountyType = watchCfg.BountyType
	}

	// Mempool watcher (on-chain targets — everything)
	wg.Add(1)
	go func() {
		defer wg.Done()
		w := scout.NewMempoolWatcher(cfg.SepoliaRPC, out, bountyType)
		if err := w.Run(ctx); err != nil {
			slog.Error("mempool watcher", "err", err)
		}
	}()

	// GitHub commit watcher (repo targets)
	wg.Add(1)
	go func() {
		defer wg.Done()
		repos := []string{}
		pollSec := 60
		if watchCfg != nil {
			repos = watchCfg.Repos
			pollSec = watchCfg.PollIntervalSeconds
		}
		gh := scout.NewGitHubWatcher(
			cfg.GitHubToken,
			out,
			repos,
			time.Duration(pollSec)*time.Second,
			bountyType,
		)
		if err := gh.Run(ctx); err != nil {
			slog.Error("github watcher", "err", err)
		}
	}()

	// Address watcher (specific contracts + wallets)
	wg.Add(1)
	go func() {
		defer wg.Done()
		contracts := []string{}
		wallets := []string{}
		pollSec := 15
		if watchCfg != nil {
			contracts = watchCfg.Contracts
			wallets = watchCfg.Wallets
			if watchCfg.PollIntervalSeconds > 0 {
				pollSec = watchCfg.PollIntervalSeconds
			}
		}
		aw := scout.NewAddressWatcher(
			cfg.SepoliaRPC,
			out,
			contracts,
			wallets,
			bountyType,
			time.Duration(pollSec)*time.Second,
		)
		if err := aw.Run(ctx); err != nil {
			slog.Error("address watcher", "err", err)
		}
	}()

	wg.Wait()
}
