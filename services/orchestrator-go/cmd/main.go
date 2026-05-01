package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/local/swarm/orchestrator/internal/config"
	"github.com/local/swarm/orchestrator/internal/llm"
	"github.com/local/swarm/orchestrator/internal/orchestrator"
	"github.com/local/swarm/orchestrator/pkg/axl"
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

	llmClient := llm.NewOpenAICompatibleClient(cfg.LLMBaseURL, cfg.LLMKey, cfg.LLMModel)

	o := orchestrator.New(node, llmClient)
	if err := o.Run(ctx); err != nil {
		slog.Error("orchestrator", "err", err)
	}
}
