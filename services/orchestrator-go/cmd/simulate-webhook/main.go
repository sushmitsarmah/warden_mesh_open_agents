package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
)

func main() {
	baseURL := os.Getenv("REPORT_SERVER_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	webhookURL := baseURL + "/webhook"

	if len(os.Args) < 2 {
		fmt.Println("Usage: simulate-webhook <pay|check>")
		fmt.Println("")
		fmt.Println("  pay   - send a fake KeeperHub Workflow completion webhook")
		fmt.Println("  check - query the x402 status endpoint")
		fmt.Println("")
		fmt.Printf("  Webhook endpoint: POST %s\n", webhookURL)
		os.Exit(1)
	}

	mode := os.Args[1]
	switch mode {
	case "pay":
		sendWebhook(webhookURL)
	case "check":
		checkStatus(baseURL)
	default:
		fmt.Printf("unknown mode: %s\n", mode)
		os.Exit(1)
	}
}

func sendWebhook(url string) {
	payload := map[string]interface{}{
		"execution_id": uuid.NewString(),
		"status":       "success",
		"outputs": map[string]string{
			"tx_hash": "0x" + randomHex(64),
		},
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("webhook request failed", "err", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("Webhook POST %s -> %d %s\n", url, resp.StatusCode, string(respBody))
}

func checkStatus(baseURL string) {
	resp, err := http.Get(baseURL + "/status")
	if err != nil {
		slog.Error("status check failed", "err", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d %s\n", resp.StatusCode, string(body))
}

func randomHex(n int) string {
	const alphabet = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[i%16]
	}
	return string(b)
}
