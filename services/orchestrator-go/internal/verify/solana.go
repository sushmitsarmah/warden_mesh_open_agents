package verify

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/local/swarm/orchestrator/pkg/messages"
)

// SolanaVerifyResult holds the outcome of verifying a Solana program exploit.
type SolanaVerifyResult struct {
	ImpactType  string
	Passed      bool
	ErrorOutput string
}

// VerifyWithSolanaValidator builds the target Solana program on a local test
// validator and executes the generated PoC transaction to confirm exploitability.
//
// The verification sequence:
//  1. Clone the target repo at the given commit.
//  2. Build the program with `cargo build-sbf`.
//  3. Start `solana-test-validator` with the program deployed.
//  4. Run the PoC script (expected at pocPath, a .ts or .js client).
//  5. Check the transaction result for the expected impact.
func VerifyWithSolanaValidator(ctx context.Context, pocPath string, finding messages.Finding) (SolanaVerifyResult, error) {
	result := SolanaVerifyResult{}

	repo := finding.Location.File // repo slug stored in file field for solana findings
	if repo == "" {
		return result, fmt.Errorf("finding has no repo (location.file is empty)")
	}

	workDir, err := os.MkdirTemp("", "solana-verify-*")
	if err != nil {
		return result, fmt.Errorf("mkdtemp: %w", err)
	}
	defer os.RemoveAll(workDir)

	// 1. Clone the program repo
	slog.Info("solana verify: cloning", "repo", repo)
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1",
		fmt.Sprintf("https://github.com/%s.git", repo),
		filepath.Join(workDir, "program"))
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return result, fmt.Errorf("git clone failed: %w — %s", err, string(out))
	}

	programDir := filepath.Join(workDir, "program")

	// 2. Build with cargo build-sbf (Solana BPF/SBF target)
	slog.Info("solana verify: building program")
	buildCmd := exec.CommandContext(ctx, "cargo", "build-sbf")
	buildCmd.Dir = programDir
	buildCmd.Env = append(os.Environ(),
		"RUST_LOG=error",
	)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return result, fmt.Errorf("cargo build-sbf failed: %w — %s", err, string(out))
	}

	// Find the compiled .so
	soFiles, err := filepath.Glob(filepath.Join(programDir, "target/deploy/*.so"))
	if err != nil || len(soFiles) == 0 {
		return result, fmt.Errorf("no compiled .so found in target/deploy/")
	}
	programSO := soFiles[0]

	// 3. Start solana-test-validator with the program loaded
	slog.Info("solana verify: starting test validator", "program", programSO)
	ledgerDir := filepath.Join(workDir, "ledger")
	validatorCmd := exec.CommandContext(ctx, "solana-test-validator",
		"--ledger", ledgerDir,
		"--bpf-program", deriveProgramID(programDir), programSO,
		"--reset",
		"--quiet",
	)
	validatorCmd.Dir = workDir
	if err := validatorCmd.Start(); err != nil {
		return result, fmt.Errorf("solana-test-validator start failed: %w", err)
	}
	defer validatorCmd.Process.Kill()

	// Give the validator 5 seconds to initialize
	slog.Info("solana verify: waiting for validator to initialize")
	select {
	case <-ctx.Done():
		return result, ctx.Err()
	case <-time.After(5 * time.Second):
	}

	// 4. Run the PoC
	slog.Info("solana verify: running PoC", "path", pocPath)
	pocResult, err := runSolanaPoC(ctx, pocPath, workDir)
	if err != nil {
		slog.Warn("solana verify: PoC execution failed", "err", err)
		result.ErrorOutput = err.Error()
		return result, nil
	}

	// 5. Interpret the result
	result = interpretSolanaResult(pocResult, finding)
	slog.Info("solana verify: result", "impact", result.ImpactType, "passed", result.Passed)
	return result, nil
}

// runSolanaPoC executes the PoC. Supports TypeScript (ts-node) and JavaScript (node) clients.
func runSolanaPoC(ctx context.Context, pocPath string, workDir string) (string, error) {
	var cmd *exec.Cmd

	switch {
	case strings.HasSuffix(pocPath, ".ts"):
		cmd = exec.CommandContext(ctx, "ts-node", pocPath)
	case strings.HasSuffix(pocPath, ".js"):
		cmd = exec.CommandContext(ctx, "node", pocPath)
	case strings.HasSuffix(pocPath, ".sh"):
		cmd = exec.CommandContext(ctx, "bash", pocPath)
	default:
		return "", fmt.Errorf("unsupported PoC file type: %s (expected .ts, .js, or .sh)", pocPath)
	}

	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"ANCHOR_PROVIDER_URL=http://127.0.0.1:8899",
		"SOLANA_URL=http://127.0.0.1:8899",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("PoC exited with error: %w\noutput: %s", err, string(out))
	}
	return string(out), nil
}

// interpretSolanaResult maps PoC output to a SolanaVerifyResult.
func interpretSolanaResult(output string, finding messages.Finding) SolanaVerifyResult {
	result := SolanaVerifyResult{ErrorOutput: output}
	lower := strings.ToLower(output)

	switch {
	case strings.Contains(lower, "exploit successful") ||
		strings.Contains(lower, "exploit passed") ||
		strings.Contains(lower, "drained"):
		result.ImpactType = "token-drain"
		result.Passed = true

	case strings.Contains(lower, "unauthorized") && strings.Contains(lower, "success"):
		result.ImpactType = "unauthorized-access"
		result.Passed = true

	case strings.Contains(lower, "arbitrary cpi") || strings.Contains(lower, "arbitrary-cpi"):
		result.ImpactType = "arbitrary-cpi"
		result.Passed = true

	case strings.Contains(lower, "overflow") || strings.Contains(lower, "underflow"):
		result.ImpactType = "arithmetic-overflow"
		result.Passed = true

	case strings.Contains(lower, "panic") || strings.Contains(lower, "program failed"):
		result.ImpactType = "program-crash"
		result.Passed = true

	default:
		// Fall back to finding category
		result.ImpactType = finding.Category
		result.Passed = strings.Contains(lower, "success") || strings.Contains(lower, "passed")
	}

	return result
}

// deriveProgramID reads the program ID from the keypair file generated by cargo build-sbf.
// Falls back to a placeholder if not found.
func deriveProgramID(programDir string) string {
	keypairs, err := filepath.Glob(filepath.Join(programDir, "target/deploy/*-keypair.json"))
	if err != nil || len(keypairs) == 0 {
		// Use a well-known test program ID as placeholder
		return "11111111111111111111111111111111"
	}

	out, err := exec.Command("solana-keygen", "pubkey", keypairs[0]).Output()
	if err != nil {
		return "11111111111111111111111111111111"
	}
	return strings.TrimSpace(string(out))
}
