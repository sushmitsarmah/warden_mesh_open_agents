package scout

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/local/swarm/scout/pkg/messages"
)

// Deduper prevents duplicate targets from being published.
type Deduper struct {
	seen map[string]time.Time
	ttl  time.Duration
}

func NewDeduper() *Deduper {
	return &Deduper{
		seen: make(map[string]time.Time),
		ttl:  time.Hour,
	}
}

// ContentID hashes the stable subset of a Target.
func ContentID(t messages.Target) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%d|%s|%s|%s", t.Kind, t.ChainID, t.Address, t.Repo, t.CommitSha)
	return hex.EncodeToString(h.Sum(nil))
}

// IsNew returns true if this target hasn't been seen recently.
func (d *Deduper) IsNew(t messages.Target) bool {
	id := ContentID(t)
	d.expire()
	if _, ok := d.seen[id]; ok {
		return false
	}
	d.seen[id] = time.Now()
	return true
}

func (d *Deduper) expire() {
	cutoff := time.Now().Add(-d.ttl)
	for id, ts := range d.seen {
		if ts.Before(cutoff) {
			delete(d.seen, id)
		}
	}
}
