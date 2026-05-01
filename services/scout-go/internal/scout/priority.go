package scout

import (
	"fmt"
	"math"
	"time"

	"github.com/local/swarm/scout/pkg/messages"
)

func Score(t messages.Target) float64 {
	// 0.5 * tvl_score + 0.3 * novelty_score + 0.2 * kind_score
	tvlScore := tvlScore(t.TVLUsd)
	noveltyScore := noveltyScore(t.DiscoveredAt)
	kindScore := kindScore(t.Kind)

	score := 0.5*tvlScore + 0.3*noveltyScore + 0.2*kindScore
	if score > 100 {
		score = 100
	}
	return score
}

func tvlScore(tvl float64) float64 {
	if tvl <= 0 {
		return 0
	}
	score := math.Log10(tvl) * 10
	if score > 100 {
		score = 100
	}
	return score
}

func noveltyScore(discoveredAt time.Time) float64 {
	elapsed := time.Since(discoveredAt)
	switch {
	case elapsed < time.Hour:
		return 100
	case elapsed < 24*time.Hour:
		return 50
	default:
		return 25
	}
}

func kindScore(kind messages.TargetKind) float64 {
	switch kind {
	case messages.TargetOnchain:
		return 80
	case messages.TargetGitHub:
		return 60
	case messages.TargetImmunefi:
		return 70
	default:
		return 50
	}
}
