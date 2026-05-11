package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/local/swarm/scout/pkg/axl"
	"github.com/local/swarm/scout/pkg/messages"
)

func main() {
	var addr string
	var chain int
	var priority float64
	var tvl float64
	fmt.Sscanf(os.Args[1], "--address=%s", &addr)
	fmt.Sscanf(os.Args[2], "--chain=%d", &chain)
	fmt.Sscanf(os.Args[3], "--priority=%f", &priority)
	fmt.Sscanf(os.Args[4], "--tvl=%f", &tvl)

	node, _ := axl.NewNode(nil)
	t := messages.Target{
		ID:           uuid.NewString(),
		Kind:         "onchain",
		ChainID:      chain,
		Address:      addr,
		DiscoveredAt: time.Now().UTC(),
		Priority:     priority,
		TVLUsd:       tvl,
	}
	b, _ := json.Marshal(t)
	node.Publish("targets/discovered", b)
	slog.Info("injected target", "id", t.ID)
}
