package safety

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	limits   map[string][]time.Time
	maxPerDay int
}

func NewRateLimiter() *RateLimiter {
	max := 3
	if v := os.Getenv("RATE_LIMIT_PER_DAY"); v != "" {
		fmt.Sscanf(v, "%d", &max)
	}
	return &RateLimiter{
		limits:    make(map[string][]time.Time),
		maxPerDay: max,
	}
}

func (r *RateLimiter) Allow(protocol string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := time.Now().Add(-24 * time.Hour)
	var valid []time.Time
	for _, ts := range r.limits[protocol] {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	if len(valid) >= r.maxPerDay {
		return false
	}
	valid = append(valid, time.Now())
	r.limits[protocol] = valid
	return true
}
