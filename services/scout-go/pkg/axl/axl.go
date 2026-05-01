package axl

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/local/swarm/scout/pkg/messages"
)

// Node is a thin wrapper around the AXL mesh SDK.
// Since the AXL SDK is not yet vendored, this implementation uses an in-memory
// gossip bus for local development. Replace with real AXL calls when SDK is available.
type Node struct {
	id       string
	mu       sync.Mutex
	subs     map[string][]chan []byte
	peers    map[string]*Node // in-memory mesh for local dev
	bootstrap []string
}

func NewNode(bootstrap []string) (*Node, error) {
	n := &Node{
		id:        fmt.Sprintf("node-%d", time.Now().UnixNano()),
		subs:      make(map[string][]chan []byte),
		peers:     make(map[string]*Node),
		bootstrap: bootstrap,
	}
	return n, nil
}

// Publish sends a payload to a topic.
func (n *Node) Publish(topic string, payload []byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, ch := range n.subs[topic] {
		ch <- payload
	}
	return nil
}

// Subscribe returns a channel that receives payloads for a topic.
func (n *Node) Subscribe(topic string) (<-chan []byte, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	ch := make(chan []byte, 64)
	n.subs[topic] = append(n.subs[topic], ch)
	return ch, nil
}

// Health returns nil if the node has at least one peer (or always nil for local dev).
func (n *Node) Health() error {
	return nil
}

// Close cleans up subscriptions.
func (n *Node) Close() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, chs := range n.subs {
		for _, ch := range chs {
			close(ch)
		}
	}
}

func LoadBootstrapFromEnv() []string {
	raw := os.Getenv("AXL_BOOTSTRAP_NODES")
	if raw == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

// PublishTarget marshals and publishes a Target.
func PublishTarget(node *Node, topic string, t messages.Target) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return node.Publish(topic, b)
}
