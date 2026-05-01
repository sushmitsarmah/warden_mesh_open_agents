package verify

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/local/swarm/orchestrator/internal/llm"
	"github.com/local/swarm/orchestrator/pkg/axl"
	"github.com/local/swarm/orchestrator/pkg/messages"
)

func TripleVerify(ctx context.Context, client llm.Client, node *axl.Node, exploitPath string, finding messages.Finding, sourcePath string) (messages.VerifiedExploit, error) {
	var result messages.VerifiedExploit

	// 1. Live-fork test
	forkURL := os.Getenv("MAINNET_RPC_URL")
	tr, err := RunForkTest(exploitPath, forkURL, 0)
	if err != nil {
		return result, fmt.Errorf("live-fork test failed: %w", err)
	}
	if !tr.Passed {
		return result, fmt.Errorf("live-fork test did not pass")
	}

	// 2. Drain threshold
	minDrain := 1000.0 // from env
	drain, err := ExtractDrainUsd(tr.Logs)
	if err != nil {
		return result, fmt.Errorf("drain extraction: %w", err)
	}
	if drain < minDrain {
		return result, fmt.Errorf("drain %f below threshold %f", drain, minDrain)
	}

	// 3. Differential check (optional)
	diffPassed := false
	if os.Getenv("ENABLE_DIFFERENTIAL") == "true" {
		diffPassed, err = DifferentialCheck(ctx, finding.ID)
		if err != nil {
			return result, fmt.Errorf("differential check: %w", err)
		}
	}

	result = messages.VerifiedExploit{
		ID:                 generateID(),
		FindingID:          finding.ID,
		ForgePath:          exploitPath,
		DrainAmountUsd:     drain,
		DifferentialPassed: diffPassed,
		VerifiedAt:         time.Now().UTC(),
	}

	// Publish
	b, _ := json.Marshal(result)
	node.Publish("exploit/verified", b)

	return result, nil
}

func generateID() string {
	return fmt.Sprintf("exploit-%d", time.Now().UnixNano())
}
