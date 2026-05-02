package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/local/swarm/orchestrator/pkg/axl"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Parse peer keys from environment
	var peerKeys []string
	if keysEnv := os.Getenv("AXL_PEER_KEYS"); keysEnv != "" {
		peerKeys = strings.Split(keysEnv, ",")
	}

	// Initialize AXL node
	node, err := axl.NewNode(peerKeys)
	if err != nil {
		slog.Error("failed to create AXL node", "err", err)
		return
	}
	defer node.Close()

	slog.Info("dashboard started", "axl_peers", len(peerKeys))

	// Subscribe to all major topics for dashboard display
	targetsCh, _ := node.Subscribe("targets/discovered")
	findingsCh, _ := node.Subscribe("analysis/findings")
	exploitsCh, _ := node.Subscribe("exploit/verified")
	disclosureCh, _ := node.Subscribe("disclosure/published")

	// Log incoming messages
	go func() {
		for {
			select {
			case msg := <-targetsCh:
				slog.Info("target discovered", "data", string(msg))
			case msg := <-findingsCh:
				slog.Info("finding received", "data", string(msg))
			case msg := <-exploitsCh:
				slog.Info("exploit verified", "data", string(msg))
			case msg := <-disclosureCh:
				slog.Info("disclosure published", "data", string(msg))
			case <-ctx.Done():
				return
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// TODO: upgrade to WebSocket and stream AXL events to browser
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	srv := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server", "err", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}