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

// Publish uploads the full report to 0G Storage, records on iNFT using the
// storage root hash as the memory pointer, gates access via x402, and
// broadcasts the disclosure to the AXL mesh.
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

	// Upload full report to 0G Storage and use root hash as memory pointer.
	storageHash, memoryDelta, err := p.uploadReport(ctx, exploit.FindingID, fullPath)
	if err != nil {
		slog.Warn("0G Storage upload failed, using fallback memory pointer", "err", err)
		// Fallback: derive memoryDelta from the disclosure ID string
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

// PublishTeaser publishes a free teaser with no payment gate.
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

// uploadReport stores the full report on 0G Storage and returns the root hash
// and its raw 32-byte form for use as the iNFT memory delta.
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

	// Decode hex root hash into raw bytes for the iNFT memory delta.
	hashBytes, err := hex.DecodeString(result.RootHash)
	if err == nil && len(hashBytes) >= 32 {
		copy(memoryDelta[:], hashBytes[:32])
	} else {
		// Root hash shorter than 32 bytes (e.g., local fallback) — left-pad with zeros.
		copy(memoryDelta[32-len(hashBytes):], hashBytes)
	}

	return result.RootHash, memoryDelta, nil
}

func generateID() string {
	return fmt.Sprintf("disclosure-%d", time.Now().UnixNano())
}
