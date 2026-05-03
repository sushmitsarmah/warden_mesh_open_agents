package axl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/local/swarm/scout/pkg/messages"
)

// Message wraps the actual payload with topic routing information
type Message struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

// Node wraps the AXL HTTP API with pub/sub semantics
type Node struct {
	apiURL    string
	peerKeys  []string // List of peer public keys to broadcast to
	mu        sync.Mutex
	subs      map[string][]chan []byte
	ctx       context.Context
	cancel    context.CancelFunc
	httpClient *http.Client
}

// NewNode creates a new AXL node client
// peerKeys: list of peer public keys to broadcast messages to
func NewNode(peerKeys []string) (*Node, error) {
	apiURL := os.Getenv("AXL_API_URL_NODE_A")
	if apiURL == "" {
		apiURL = "http://127.0.0.1:9002"
	}

	ctx, cancel := context.WithCancel(context.Background())
	n := &Node{
		apiURL:   apiURL,
		peerKeys: peerKeys,
		subs:     make(map[string][]chan []byte),
		ctx:      ctx,
		cancel:   cancel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Start background receiver
	go n.receiveLoop()

	slog.Info("axl node initialized", "api_url", apiURL, "peers", len(peerKeys))
	return n, nil
}

// Publish broadcasts a message to all peers on a topic
func (n *Node) Publish(topic string, payload []byte) error {
	msg := Message{
		Topic:   topic,
		Payload: json.RawMessage(payload),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Broadcast to all known peers
	for _, peerKey := range n.peerKeys {
		if err := n.sendToPeer(peerKey, msgBytes); err != nil {
			slog.Warn("failed to send to peer", "peer", peerKey, "err", err)
			// Continue broadcasting to other peers even if one fails
		}
	}

	slog.Debug("published message", "topic", topic, "peers", len(n.peerKeys))
	return nil
}

// sendToPeer sends raw bytes to a specific peer via AXL HTTP API
func (n *Node) sendToPeer(peerKey string, data []byte) error {
	req, err := http.NewRequest("POST", n.apiURL+"/send", bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("X-Destination-Peer-Id", peerKey)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send failed: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// receiveLoop continuously polls for incoming messages and routes them to subscribers
func (n *Node) receiveLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.pollMessages()
		}
	}
}

// pollMessages fetches pending messages from AXL
func (n *Node) pollMessages() {
	resp, err := n.httpClient.Get(n.apiURL + "/recv")
	if err != nil {
		slog.Debug("recv poll error", "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return // No messages available
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("failed to read recv body", "err", err)
		return
	}

	// Try to unmarshal as our Message wrapper
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Debug("received non-wrapped message, ignoring", "err", err)
		return
	}

	// Route to topic subscribers
	n.mu.Lock()
	subscribers := n.subs[msg.Topic]
	n.mu.Unlock()

	for _, ch := range subscribers {
		select {
		case ch <- []byte(msg.Payload):
		default:
			slog.Warn("subscriber channel full, dropping message", "topic", msg.Topic)
		}
	}
}

// Subscribe returns a channel that receives messages for a topic
func (n *Node) Subscribe(topic string) (<-chan []byte, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	ch := make(chan []byte, 100)
	n.subs[topic] = append(n.subs[topic], ch)

	slog.Info("subscribed to topic", "topic", topic)
	return ch, nil
}

// Health checks if the AXL node is reachable
func (n *Node) Health() error {
	resp, err := n.httpClient.Get(n.apiURL + "/topology")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}

// Close shuts down the node
func (n *Node) Close() {
	n.cancel()

	n.mu.Lock()
	defer n.mu.Unlock()

	for _, chs := range n.subs {
		for _, ch := range chs {
			close(ch)
		}
	}
}

// LoadBootstrapFromEnv loads peer keys from environment
func LoadBootstrapFromEnv() []string {
	raw := os.Getenv("AXL_PEERS_FOR_NODE_A")
	if raw == "" {
		return nil
	}

	// Split by comma or newline
	peers := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ' '
	})

	// Filter out empty strings
	var result []string
	for _, p := range peers {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}

	return result
}

// PublishTarget marshals and publishes a Target
func PublishTarget(node *Node, topic string, t messages.Target) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return node.Publish(topic, b)
}
