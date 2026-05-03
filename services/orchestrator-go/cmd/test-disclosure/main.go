package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/local/swarm/orchestrator/internal/disclosure"
	"github.com/local/swarm/orchestrator/internal/inft"
	"github.com/local/swarm/orchestrator/internal/storage"
	"github.com/local/swarm/orchestrator/internal/x402"
	"github.com/local/swarm/orchestrator/pkg/axl"
	"github.com/local/swarm/orchestrator/pkg/messages"
)

func main() {
	slog.Info("test-disclosure: End-to-end disclosure + iNFT storage hash")
	ctx := context.Background()

	// Initialize dependencies (works with or without env keys)
	node, _ := axl.NewNode(nil)
	gate := x402.NewGate("http://localhost:8080")
	storageClient := storage.NewLightClient(os.Getenv("STORAGE_GATEWAY_URL"))
	storageClient.SelfTest(ctx)

	// iNFT client (nil if no config, disclosure still works)
	var inftClient inft.Recorder
	if os.Getenv("OG_RPC_URL") != "" && os.Getenv("OG_PRIVATE_KEY") != "" && os.Getenv("OG_INFT_ADDRESS") != "" {
		cl, err := inft.NewClient(os.Getenv("OG_RPC_URL"), os.Getenv("OG_INFT_ADDRESS"), os.Getenv("OG_PRIVATE_KEY"))
		if err != nil {
			slog.Error("inft client init failed, continuing without on-chain recording", "err", err)
		} else {
			inftClient = cl
		}
	}

	publisher := disclosure.NewPublisher(node, gate, inftClient, storageClient)

	// Create temp report files
	os.WriteFile("/tmp/test-teaser.md", []byte("# Teaser\nReentrancy found!"), 0644)
	os.WriteFile("/tmp/test-full-report.md", []byte("# Full Report\nDetailed PoC..."), 0644)

	exploit := messages.VerifiedExploit{
		ID:        "exploit-test-001",
		FindingID: "finding-test-001",
	}

	if err := publisher.Publish(ctx, exploit, "/tmp/test-teaser.md", "/tmp/test-full-report.md"); err != nil {
		slog.Error("publish failed", "err", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ test-disclosure completed successfully")
	fmt.Println("If iNFT was configured, run:")
	fmt.Printf("  cast call %s \"state(uint256)\" 1 --rpc-url %s\n",
		os.Getenv("OG_INFT_ADDRESS"), os.Getenv("OG_RPC_URL"))
	fmt.Println("Then inspect memoryPointer (bytes32) to confirm the root hash matches the upload.")
}
