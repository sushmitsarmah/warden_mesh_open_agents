package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/local/swarm/scout/pkg/messages"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: simulate-demo <target|finding|auto>")
		fmt.Println("")
		fmt.Println("  target   - inject a synthetic contract target")
		fmt.Println("  finding  - inject a synthetic vulnerability finding")
		fmt.Println("  auto     - inject target then finding (orchestrator will pick it up)")
		fmt.Println("")
		fmt.Println("Environment:")
		fmt.Println("  SIM_SRC_NODE   - AXL source node API URL (default: http://127.0.0.1:9004)")
		fmt.Println("  SIM_DEST_NODE  - AXL destination node API URL (default: http://127.0.0.1:9002)")
		os.Exit(1)
	}

	mode := os.Args[1]

	switch mode {
	case "target":
		injectTarget()
	case "finding":
		injectFinding()
	case "auto":
		injectTarget()
		time.Sleep(1 * time.Second)
		injectFinding()
	default:
		fmt.Printf("unknown mode: %s\n", mode)
		os.Exit(1)
	}
}

func injectTarget() {
	t := messages.Target{
		ID:           uuid.NewString(),
		Kind:         "onchain",
		ChainID:      1,
		Address:      "0x" + randomHex(40),
		DiscoveredAt: time.Now().UTC(),
		Priority:     0.95,
		TVLUsd:       5_000_000,
	}
	if err := postToAXL("targets/discovered", t); err != nil {
		slog.Error("inject target failed", "err", err)
	} else {
		slog.Info("injected target", "id", t.ID, "address", t.Address)
	}
}

func injectFinding() {
	f := messages.Finding{
		ID:       uuid.NewString(),
		TargetID: uuid.NewString(),
		Category: "reentrancy",
		Severity: messages.SevCritical,
		Tools:    []string{"aderyn", "slither"},
		Location: messages.Location{
			File:      "VulnerableVault.sol",
			LineStart: 42,
			LineEnd:   58,
		},
		Description: "External call before state update allows recursive withdrawal.",
	}
	if err := postToAXL("analysis/findings", f); err != nil {
		slog.Error("inject finding failed", "err", err)
	} else {
		slog.Info("injected finding", "id", f.ID, "severity", f.Severity, "category", f.Category)
	}
}

// postToAXL sends a message via the source AXL node to the destination AXL node
// across the real Yggdrasil mesh. Discovery: source node queries destination's
// public key from its /topology, then sends via source's /send endpoint.
func postToAXL(topic string, payload any) error {
	srcNode := os.Getenv("SIM_SRC_NODE")
	if srcNode == "" {
		srcNode = "http://127.0.0.1:9004"
	}
	destNode := os.Getenv("SIM_DEST_NODE")
	if destNode == "" {
		destNode = "http://127.0.0.1:9002"
	}

	// 1. Discover destination node's public key.
	destKey, err := getNodePublicKey(destNode)
	if err != nil {
		return fmt.Errorf("cannot discover dest node public key: %w", err)
	}

	// 2. Wrap in AXL Message envelope.
	msg := struct {
		Topic   string          `json:"topic"`
		Payload json.RawMessage `json:"payload"`
	}{
		Topic: topic,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg.Payload = payloadBytes

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 3. POST to SOURCE node's /send with DESTINATION node's key as peer.
	req, err := http.NewRequest("POST", srcNode+"/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Destination-Peer-Id", destKey)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("axl send failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("axl send returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func getNodePublicKey(nodeURL string) (string, error) {
	resp, err := http.Get(nodeURL + "/topology")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var topo struct {
		OurPublicKey string `json:"our_public_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&topo); err != nil {
		return "", err
	}
	if topo.OurPublicKey == "" {
		return "", fmt.Errorf("our_public_key not in topology response from %s", nodeURL)
	}
	return topo.OurPublicKey, nil
}

func randomHex(n int) string {
	const alphabet = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[i%16]
	}
	return string(b)
}
