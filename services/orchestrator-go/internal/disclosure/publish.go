package disclosure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/local/swarm/orchestrator/internal/x402"
	"github.com/local/swarm/orchestrator/pkg/axl"
	"github.com/local/swarm/orchestrator/pkg/messages"
)

// Publisher handles vulnerability disclosure and monetization
type Publisher struct {
	node         *axl.Node
	paymentGate  *x402.Gate
	defaultPrice float64
}

func NewPublisher(node *axl.Node, gate *x402.Gate) *Publisher {
	return &Publisher{
		node:         node,
		paymentGate:  gate,
		defaultPrice: 1000.0, // Default $1000 USD per report
	}
}

// Publish creates a gated report and publishes the disclosure
func (p *Publisher) Publish(ctx context.Context, exploit messages.VerifiedExploit, teaserPath, fullPath string) error {
	// Create payment-gated report
	report, err := p.paymentGate.CreateGatedReport(
		exploit.FindingID,
		teaserPath,
		fullPath,
		p.defaultPrice,
	)
	if err != nil {
		return fmt.Errorf("failed to create gated report: %w", err)
	}

	// Create disclosure message
	d := messages.Disclosure{
		ID:          generateID(),
		ExploitID:   exploit.ID,
		Lane:        "bounty",
		X402URL:     report.PaymentURL,
		PublishedAt: time.Now().UTC(),
	}

	slog.Info("publishing disclosure",
		"disclosure_id", d.ID,
		"exploit_id", exploit.ID,
		"x402_url", report.PaymentURL,
		"price_usd", p.defaultPrice,
	)

	// Publish to AXL mesh
	b, _ := json.Marshal(d)
	if err := p.node.Publish("disclosure/published", b); err != nil {
		return fmt.Errorf("failed to publish disclosure: %w", err)
	}

	slog.Info("disclosure published successfully", "disclosure_id", d.ID)
	return nil
}

// PublishTeaser publishes a free teaser (no payment required)
func (p *Publisher) PublishTeaser(ctx context.Context, exploit messages.VerifiedExploit, teaserPath string) error {
	d := messages.Disclosure{
		ID:          generateID(),
		ExploitID:   exploit.ID,
		Lane:        "bounty",
		X402URL:     "", // No payment required for teaser
		PublishedAt: time.Now().UTC(),
	}

	b, _ := json.Marshal(d)
	return p.node.Publish("disclosure/teaser", b)
}

func generateID() string {
	return fmt.Sprintf("disclosure-%d", time.Now().UnixNano())
}
