package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/local/swarm/scout/pkg/messages"
)

func main() {
	t := messages.Target{
		ID:           "test-target-001",
		Kind:         messages.TargetOnchain,
		ChainID:      11155111,
		Address:      "0x1234567890123456789012345678901234567890",
		DiscoveredAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		Priority:     42.5,
		TVLUsd:       500000,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(t); err != nil {
		fmt.Fprintln(os.Stderr, "encode error:", err)
		os.Exit(1)
	}
}
