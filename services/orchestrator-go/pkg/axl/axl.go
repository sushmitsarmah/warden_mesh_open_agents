package axl

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Node is a thin AXL wrapper. Replace with real SDK when available.
type Node struct {
	id    string
	mu    sync.Mutex
	subs  map[string][]chan []byte
}

func NewNode(bootstrap []string) (*Node, error) {
	_ = bootstrap
	return &Node{
		id:   fmt.Sprintf("node-%d", time.Now().UnixNano()),
		subs: make(map[string][]chan []byte),
	}, nil
}

func (n *Node) Publish(topic string, payload []byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, ch := range n.subs[topic] {
		ch <- payload
	}
	return nil
}

func (n *Node) Subscribe(topic string) (<-chan []byte, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	ch := make(chan []byte, 64)
	n.subs[topic] = append(n.subs[topic], ch)
	return ch, nil
}

func (n *Node) Health() error { return nil }

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
