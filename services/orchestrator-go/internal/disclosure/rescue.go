package disclosure

import (
	"context"
	"fmt"

	"github.com/local/swarm/orchestrator/pkg/messages"
)

type Rescue struct{}

func (r *Rescue) Execute(ctx context.Context, exploit messages.VerifiedExploit, protocol string) error {
	return fmt.Errorf("rescue lane disabled for hackathon")
}
