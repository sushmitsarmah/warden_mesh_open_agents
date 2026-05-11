package scout

import (
	"testing"
	"time"

	"github.com/local/swarm/scout/pkg/messages"
	"github.com/stretchr/testify/assert"
)

func TestScore(t *testing.T) {
	onchainHigh := messages.Target{
		Kind:         "onchain",
		DiscoveredAt: time.Now().UTC(),
		TVLUsd:       1e9,
	}
	score := Score(onchainHigh)
	assert.GreaterOrEqual(t, score, 60.0)
	assert.LessOrEqual(t, score, 100.0)

	github := messages.Target{
		Kind:         "github",
		DiscoveredAt: time.Now().UTC().Add(-2 * time.Hour),
		TVLUsd:       0,
	}
	score = Score(github)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)

	immunefi := messages.Target{
		Kind:         "immunefi",
		DiscoveredAt: time.Now().UTC().Add(-48 * time.Hour),
		TVLUsd:       1e7,
	}
	score = Score(immunefi)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
}
