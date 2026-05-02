package orchestrator

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/local/swarm/orchestrator/internal/disclosure"
	"github.com/local/swarm/orchestrator/internal/llm"
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
			slog.Info("finding", "id", f.ID, "category", f.Category, "severity", f.Severity)
			_ = o.publisher
		}
	}
}
