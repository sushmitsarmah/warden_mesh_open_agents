package disclosure

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/local/swarm/orchestrator/internal/inft"
	"github.com/local/swarm/orchestrator/internal/storage"
	"github.com/local/swarm/orchestrator/internal/x402"
	"github.com/local/swarm/orchestrator/pkg/axl"
	"github.com/local/swarm/orchestrator/pkg/messages"
)

type Publisher struct {
	node          *axl.Node
	paymentGate   *x402.Gate
	inftRecorder  inft.Recorder
	storageClient *storage.LightClient
	defaultPrice  float64
}

func NewPublisher(node *axl.Node, gate *x402.Gate, recorder inft.Recorder, storageClient *storage.LightClient) *Publisher {
	return &Publisher{
		node:          node,
		paymentGate:   gate,
		inftRecorder:  recorder,
		storageClient: storageClient,
		defaultPrice:  1000.0,
	}
}

func (p *Publisher) Publish(ctx context.Context, exploit messages.VerifiedExploit, teaserPath, fullPath string) error {
	report, err := p.paymentGate.CreateGatedReport(
		exploit.FindingID,
		teaserPath,
		fullPath,
		p.defaultPrice,
	)
	if err != nil {
		return fmt.Errorf("failed to create gated report: %w", err)
	}

	storageHash, memoryDelta, err := p.uploadReport(ctx, exploit.FindingID, fullPath)
	if err != nil {
		slog.Warn("0G Storage upload failed, using fallback memory pointer", "err", err)
		copy(memoryDelta[:], []byte(exploit.FindingID))
	}

	d := messages.Disclosure{
		ID:          generateID(),
		ExploitID:   exploit.ID,
		Lane:        "bounty",
		X402URL:     report.PaymentURL,
		StorageHash: storageHash,
		PublishedAt: time.Now().UTC(),
	}

	slog.Info("publishing disclosure",
		"disclosure_id", d.ID,
		"exploit_id", exploit.ID,
		"finding_id", exploit.FindingID,
		"impact_type", exploit.ImpactType,
		"x402_url", report.PaymentURL,
		"storage_hash", storageHash,
		"price_usd", p.defaultPrice,
	)

	if p.inftRecorder != nil {
		if err := p.inftRecorder.RecordDisclosure(ctx, 1, int64(p.defaultPrice), memoryDelta); err != nil {
			slog.Error("failed to record disclosure on iNFT", "err", err)
		}
	}

	b, _ := json.Marshal(d)
	if err := p.node.Publish("disclosure/published", b); err != nil {
		return fmt.Errorf("failed to publish disclosure: %w", err)
	}

	slog.Info("disclosure published successfully", "disclosure_id", d.ID)
	return nil
}

func (p *Publisher) PublishTeaser(ctx context.Context, exploit messages.VerifiedExploit, teaserPath string) error {
	d := messages.Disclosure{
		ID:          generateID(),
		ExploitID:   exploit.ID,
		Lane:        "bounty",
		PublishedAt: time.Now().UTC(),
	}
	b, _ := json.Marshal(d)
	return p.node.Publish("disclosure/teaser", b)
}

func (p *Publisher) uploadReport(ctx context.Context, findingID, fullPath string) (string, [32]byte, error) {
	var memoryDelta [32]byte

	if p.storageClient == nil {
		return "", memoryDelta, fmt.Errorf("no storage client configured")
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", memoryDelta, fmt.Errorf("read full report: %w", err)
	}

	result, err := p.storageClient.Upload(ctx, findingID+"-report.md", data)
	if err != nil {
		return "", memoryDelta, fmt.Errorf("0G upload: %w", err)
	}

	hashBytes, err := hex.DecodeString(result.RootHash)
	if err == nil && len(hashBytes) >= 32 {
		copy(memoryDelta[:], hashBytes[:32])
	} else {
		copy(memoryDelta[32-len(hashBytes):], hashBytes)
	}

	return result.RootHash, memoryDelta, nil
}

func generateID() string {
	return fmt.Sprintf("disclosure-%d", time.Now().UnixNano())
}
