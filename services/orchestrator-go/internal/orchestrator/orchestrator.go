package orchestrator

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/local/swarm/orchestrator/internal/disclosure"
	"github.com/local/swarm/orchestrator/internal/exploit"
	"github.com/local/swarm/orchestrator/internal/llm"
	"github.com/local/swarm/orchestrator/internal/report"
	"github.com/local/swarm/orchestrator/internal/verify"
	"github.com/local/swarm/orchestrator/pkg/axl"
	"github.com/local/swarm/orchestrator/pkg/messages"
)

type Orchestrator struct {
	node      *axl.Node
	llmClient llm.Client
	publisher *disclosure.Publisher
}

func New(node *axl.Node, llmClient llm.Client, publisher *disclosure.Publisher) *Orchestrator {
	return &Orchestrator{node: node, llmClient: llmClient, publisher: publisher}
}

func (o *Orchestrator) Run(ctx context.Context) error {
	ch, err := o.node.Subscribe("analysis/findings")
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			var f messages.Finding
			if err := json.Unmarshal(msg, &f); err != nil {
				slog.Error("unmarshal finding", "err", err)
				continue
			}
			slog.Info("finding received", "id", f.ID, "bounty_type", f.BountyType, "category", f.Category, "severity", f.Severity)
			go o.process(ctx, f)
		}
	}
}

func (o *Orchestrator) process(ctx context.Context, f messages.Finding) {
	slog.Info("processing finding", "id", f.ID)

	exploitPath, err := exploit.Generate(ctx, o.llmClient, f, f.Location.File)
	if err != nil {
		slog.Error("exploit generation failed", "id", f.ID, "err", err)
		return
	}
	slog.Info("exploit generated", "id", f.ID, "path", exploitPath)

	verified, err := verify.TripleVerify(ctx, o.llmClient, o.node, exploitPath, f, f.Location.File)
	if err != nil {
		slog.Warn("verification failed, continuing with unverified finding", "id", f.ID, "err", err)
		verified = messages.VerifiedExploit{
			ID:        "unverified-" + f.ID,
			FindingID: f.ID,
			PoCPath:   exploitPath,
			AttackerModel: "unknown",
			ImpactType: "unknown",
		}
	}

	teaserPath, fullPath, err := report.Generate(ctx, o.llmClient, verified)
	if err != nil {
		slog.Error("report generation failed", "id", f.ID, "err", err)
		return
	}
	slog.Info("report generated", "id", f.ID, "teaser", teaserPath)

	if err := o.publisher.Publish(ctx, verified, teaserPath, fullPath); err != nil {
		slog.Error("disclosure failed", "id", f.ID, "err", err)
		return
	}
	slog.Info("disclosure published", "id", f.ID)
}
