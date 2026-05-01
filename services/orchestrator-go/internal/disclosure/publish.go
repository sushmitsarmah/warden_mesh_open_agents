package disclosure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/local/swarm/orchestrator/pkg/axl"
	"github.com/local/swarm/orchestrator/pkg/messages"
)

func Publish(ctx context.Context, node *axl.Node, exploit messages.VerifiedExploit, x402URL string) error {
	d := messages.Disclosure{
		ID:          generateID(),
		ExploitID:   exploit.ID,
		Lane:        "bounty",
		X402URL:     x402URL,
		PublishedAt: time.Now().UTC(),
	}
	b, _ := json.Marshal(d)
	return node.Publish("disclosure/published", b)
}

func generateID() string {
	return fmt.Sprintf("disclosure-%d", time.Now().UnixNano())
}
