package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/local/swarm/orchestrator/internal/config"
	"github.com/local/swarm/orchestrator/internal/disclosure"
	"github.com/local/swarm/orchestrator/internal/inft"
	"github.com/local/swarm/orchestrator/internal/llm"
	"github.com/local/swarm/orchestrator/internal/orchestrator"
	"github.com/local/swarm/orchestrator/internal/x402"
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

	// Initialize 0G iNFT client (optional — only if contract is configured)
	var inftClient *inft.Client
	if cfg.OGRpcURL != "" && cfg.OGPrivateKey != "" && cfg.OGINFTAddress != "" {
		inftClient, err = inft.NewClient(cfg.OGRpcURL, cfg.OGINFTAddress, cfg.OGPrivateKey)
		if err != nil {
			slog.Error("inft client init failed, continuing without on-chain recording", "err", err)
			inftClient = nil
		}
	}

	// Initialize x402 payment gate
	gate := x402.NewGate(cfg.ReportServerBaseURL)

	// Initialize disclosure publisher with iNFT recorder
	publisher := disclosure.NewPublisher(node, gate, inftClient)

	o := orchestrator.New(node, llmClient, publisher)
	if err := o.Run(ctx); err != nil {
		slog.Error("orchestrator", "err", err)
	}
}
