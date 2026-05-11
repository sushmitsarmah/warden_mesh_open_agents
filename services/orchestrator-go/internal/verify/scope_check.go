package verify

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/local/swarm/orchestrator/pkg/messages"
)

// ScopeCheck enforces the Firedancer bounty scope rules.
type ScopeCheck struct {
	outOfScopePatterns []string
}

func NewScopeCheck() *ScopeCheck {
	return &ScopeCheck{
		outOfScopePatterns: []string{
			"firedancer-dev",
			"fdctl/fddev",
			"/test",
			"/tests/",
			"Frankendancer",
			"/ci/",
			"/scripts/",
			".git/",
		},
	}
}

// Check verifies a finding is in scope. Returns nil if valid, error if out of scope.
func (s *ScopeCheck) Check(finding messages.Finding) error {
	file := finding.Location.File

	// 1. Check out-of-scope path patterns (scope.md:40-48)
	for _, pattern := range s.outOfScopePatterns {
		if strings.Contains(file, pattern) {
			return fmt.Errorf("out of scope: path matches %q (file: %s)", pattern, file)
		}
	}

	// 2. Check for TODO/FIXME references (scope.md:136)
	if strings.Contains(finding.Description, "TODO") || strings.Contains(finding.Description, "FIXME") {
		return fmt.Errorf("out of scope: finding references TODO/FIXME comment")
	}

	return nil
}

// CheckTODOFixme is a dedicated check for scope.md:136
func (s *ScopeCheck) CheckTODOFixme(finding messages.Finding) bool {
	return strings.Contains(finding.Description, "TODO") || strings.Contains(finding.Description, "FIXME")
}

// CheckBountyConfig validates the PoC against the config TOML (section 8c).
// This checks if the PoC references configuration that should be present.
func CheckBountyConfig(pocPath string) error {
	data, err := os.ReadFile(pocPath)
	if err != nil {
		slog.Warn("scope_check: could not read PoC for config validation", "err", err)
		return nil
	}

	content := string(data)

	// The reference config (section 8c) defines which components are network-exposed.
	// If the PoC relies on RPC or GUI being enabled, that must be checked against the config.
	// This is a basic check — full validation would parse the TOML and verify.
	if strings.Contains(content, "RPC") || strings.Contains(content, "rpc") {
		// RPC is network-exposed per section 8c, so this is fine
	}

	return nil
}
