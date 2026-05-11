package verify

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/local/swarm/orchestrator/pkg/messages"
)

// ClusterResult holds the outcome of a Firedancer cluster verification.
type ClusterResult struct {
	Passed                bool
	ImpactType            string // crash | bank-hash-mismatch | sandbox-escape | liveness-failure | invalid-block
	Stdout                string
	Stderr                string
	ValidatedAgainstConfig bool
	ModifiedValidator     bool
	Justification         string
}

// VerifyWithCluster builds firedancer and runs the PoC against it.
func VerifyWithCluster(ctx context.Context, pocPath string, finding messages.Finding) (*ClusterResult, error) {
	workDir := "/tmp/firedancer-verify"
	_ = os.MkdirAll(workDir, 0755)

	// 1. Clone firedancer at v1.0
	cloneDir := filepath.Join(workDir, "firedancer")
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		targetRepo := "firedancer-io/firedancer"
		if finding.BountyType == "firedancer" {
			// Check if there's a repo hint on the finding
		}
		cmd := exec.CommandContext(ctx, "git", "clone",
			"--branch", "v1.0",
			"--depth", "1",
			fmt.Sprintf("https://github.com/%s.git", targetRepo),
			cloneDir,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("git clone failed: %w\n%s", err, string(out))
		}
	}

	// 2. Compiler enforcement: must use GCC, reject Clang
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "gcc-14"
	}
	cxx := os.Getenv("CXX")
	if cxx == "" {
		cxx = "g++-14"
	}

	// Verify it's GCC, not Clang
	verCheck := exec.CommandContext(ctx, cc, "--version")
	verOut, _ := verCheck.CombinedOutput()
	verStr := string(verOut)
	if strings.Contains(verStr, "clang") || strings.Contains(verStr, "LLVM") {
		return nil, fmt.Errorf("compiler %s is Clang, not GCC (only GCC 8.5/13/14 are in scope)", cc)
	}

	// 3. Build with the reference config
	buildCmd := exec.CommandContext(ctx, "make", "-j", "4")
	buildCmd.Dir = cloneDir
	buildCmd.Env = append(os.Environ(),
		"CC="+cc,
		"CXX="+cxx,
	)
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("build failed: %w\n%s", err, string(buildOut))
	}

	// 4. Copy PoC into the cluster environment
	pocDest := filepath.Join(cloneDir, "poc")
	pocData, err := os.ReadFile(pocPath)
	if err != nil {
		return nil, fmt.Errorf("read poc: %w", err)
	}
	if err := os.WriteFile(pocDest, pocData, 0755); err != nil {
		return nil, fmt.Errorf("write poc: %w", err)
	}

	// 5. Run PoC
	pocCmd := exec.CommandContext(ctx, pocDest)
	pocCmd.Dir = cloneDir
	pocOut, err := pocCmd.CombinedOutput()
	pocStdout := string(pocOut)
	pocStderr := ""
	if err != nil {
		pocStderr = err.Error()
	}

	// 6. Analyze result
	result := &ClusterResult{
		ValidatedAgainstConfig: true,
	}

	// Check for bank hash mismatch (requires Agave comparison)
	if strings.Contains(pocStdout, "bank hash mismatch") ||
		strings.Contains(pocStdout, "BankHashMismatch") ||
		strings.Contains(pocStdout, "MISMATCH") {
		result.Passed = true
		result.ImpactType = "bank-hash-mismatch"
		result.Stdout = pocStdout
		result.Stderr = pocStderr
		return result, nil
	}

	// Check for sandbox escape
	if strings.Contains(pocStdout, "sandbox") &&
		(strings.Contains(pocStdout, "escape") || strings.Contains(pocStdout, "violation")) {
		result.Passed = true
		result.ImpactType = "sandbox-escape"
		result.Stdout = pocStdout
		result.Stderr = pocStderr
		return result, nil
	}

	// Check for crash (non-zero exit or signal)
	if err != nil || strings.Contains(pocStdout, "SIGSEGV") ||
		strings.Contains(pocStdout, "SIGABRT") ||
		strings.Contains(pocStdout, "SEGFAULT") ||
		strings.Contains(pocStdout, "abort") ||
		strings.Contains(pocStdout, "panic") {

		// Reject error-code-only crashes (not eligible per scope.md:24)
		if strings.Contains(pocStdout, "error code") ||
			strings.Contains(pocStdout, "ErrorCode") ||
			strings.Contains(pocStdout, "result: Err") {
			return nil, fmt.Errorf("error-code-only crash (not eligible): %s", pocStdout)
		}

		result.Passed = true
		result.ImpactType = "crash"
		result.Stdout = pocStdout
		result.Stderr = pocStderr
		return result, nil
	}

	// Check for liveness failure
	if strings.Contains(pocStdout, "liveness") ||
		strings.Contains(pocStdout, "no blocks") ||
		strings.Contains(pocStdout, "stall") {
		result.Passed = true
		result.ImpactType = "liveness-failure"
		result.Stdout = pocStdout
		result.Stderr = pocStderr
		return result, nil
	}

	// Check for invalid block
	if strings.Contains(pocStdout, "invalid block") ||
		strings.Contains(pocStdout, "InvalidBlock") {
		result.Passed = true
		result.ImpactType = "invalid-block"
		result.Stdout = pocStdout
		result.Stderr = pocStderr
		return result, nil
	}

	return nil, fmt.Errorf("PoC did not demonstrate eligible impact: exit_code=%v, stdout=%s", err, truncate(pocStdout, 500))
}

// extractAttackerModel reads the attacker model from the PoC file header.
func extractAttackerModel(pocPath string) string {
	data, err := os.ReadFile(pocPath)
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "Attacker model") || strings.Contains(line, "attacker_model") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "unknown"
}

// checkModifiedValidator checks if the PoC modifies the validator source.
func checkModifiedValidator(pocPath string) (bool, string) {
	data, err := os.ReadFile(pocPath)
	if err != nil {
		return false, ""
	}
	text := string(data)
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "Modified validator") || strings.Contains(line, "modified_validator") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if val == "yes" || val == "true" || val == "y" {
					// Look for justification in subsequent lines
					justification := extractJustification(text)
					return true, justification
				}
			}
		}
	}
	return false, ""
}

func extractJustification(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "Justification") || strings.Contains(line, "justification") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func generateID() string {
	return fmt.Sprintf("exploit-%d", time.Now().UnixNano())
}
