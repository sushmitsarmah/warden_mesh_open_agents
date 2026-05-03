package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/local/swarm/orchestrator/internal/storage"
)

func main() {
	slog.Info("test-storage: Uploading a report to 0G Storage")
	ctx := context.Background()

	client := storage.NewLightClient(os.Getenv("STORAGE_GATEWAY_URL"))

	// 1. Self-test (1KB blob)
	fmt.Println("\n--- Self-Test ---")
	client.SelfTest(ctx)

	// 2. Upload synthetic report
	fmt.Println("\n--- Upload Test ---")
	data := []byte("# Vulnerability Report\n\nReentrancy in Vault.sol...")
	result, err := client.Upload(ctx, "test-report.md", data)
	if err != nil {
		slog.Error("upload failed", "err", err)
		os.Exit(1)
	}

	fmt.Printf("uploaded to 0g, root hash: %s\n", result.RootHash)
	fmt.Printf("size: %d bytes\n", result.Size)
}
