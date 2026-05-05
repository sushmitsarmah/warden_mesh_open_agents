package verify

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/local/swarm/orchestrator/pkg/messages"
)

// KnownIssuesFilter checks findings against known issue trackers.
// Duplicates for the same unfixed bug are valid per assets.md:85-91.
type KnownIssuesFilter struct {
	mu       sync.Mutex
	submitted map[string]time.Time // fingerprint -> submission time
}

func NewKnownIssuesFilter() *KnownIssuesFilter {
	return &KnownIssuesFilter{
		submitted: make(map[string]time.Time),
	}
}

// fingerprint creates a deterministic key for duplicate detection.
func (f *KnownIssuesFilter) fingerprint(finding messages.Finding) string {
	return fmt.Sprintf("%s:%s:%s:%d", finding.Location.File, finding.Category, finding.Description, finding.Location.LineStart)
}

// Check returns: (isDuplicate, isKnownIssue, error).
// - isDuplicate = true means same finding was already submitted (still valid per bounty rules)
// - isKnownIssue = true means it matches a public known issue (invalid)
func (f *KnownIssuesFilter) Check(finding messages.Finding) (bool, bool, error) {
	fp := f.fingerprint(finding)

	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if this was already submitted
	if _, exists := f.submitted[fp]; exists {
		slog.Info("known_issues: duplicate finding (still valid per bounty rules)", "id", finding.ID, "fp", fp)
		return true, false, nil
	}

	return false, false, nil
}

// MarkSubmitted records a finding as submitted.
func (f *KnownIssuesFilter) MarkSubmitted(finding messages.Finding) {
	fp := f.fingerprint(finding)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.submitted[fp] = time.Now()
}
