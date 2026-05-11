package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/local/swarm/orchestrator/internal/llm"
	"github.com/local/swarm/orchestrator/pkg/axl"
	"github.com/local/swarm/orchestrator/pkg/messages"
)

func TripleVerify(ctx context.Context, client llm.Client, node *axl.Node, exploitPath string, finding messages.Finding, sourcePath string) (messages.VerifiedExploit, error) {
	// 1. Scope check (firedancer-specific rules)
	scopeCheck := NewScopeCheck()
	if err := scopeCheck.Check(finding); err != nil {
		return messages.VerifiedExploit{}, fmt.Errorf("scope check failed: %w", err)
	}

	// 2. Dispatch by bounty type
	bountyType := finding.BountyType
	if bountyType == "" {
		bountyType = "solidity-evm"
	}

	slog.Info("verifying finding", "id", finding.ID, "bounty_type", bountyType)

	switch bountyType {
	case "firedancer":
		return verifyFiredancer(ctx, exploitPath, finding)
	case "solana-program":
		return verifySolanaProgram(ctx, exploitPath, finding, node)
	default:
		return verifySolidityEVM(ctx, client, node, exploitPath, finding, sourcePath)
	}
}

func verifySolanaProgram(ctx context.Context, exploitPath string, finding messages.Finding, node *axl.Node) (messages.VerifiedExploit, error) {
	slog.Info("verifying solana program", "finding_id", finding.ID, "poc", exploitPath)

	res, err := VerifyWithSolanaValidator(ctx, exploitPath, finding)
	if err != nil {
		return messages.VerifiedExploit{}, fmt.Errorf("solana validator verification failed: %w", err)
	}
	if !res.Passed {
		return messages.VerifiedExploit{}, fmt.Errorf("solana PoC did not demonstrate impact (output: %s)", res.ErrorOutput)
	}

	result := messages.VerifiedExploit{
		ID:            generateID(),
		FindingID:     finding.ID,
		PoCPath:       exploitPath,
		ImpactType:    res.ImpactType,
		AttackerModel: "remote",
		VerifiedAt:    time.Now().UTC(),
	}

	b, _ := json.Marshal(result)
	node.Publish("exploit/verified", b)

	slog.Info("solana verification passed", "id", result.ID, "impact", result.ImpactType)
	return result, nil
}

func verifyFiredancer(ctx context.Context, exploitPath string, finding messages.Finding) (messages.VerifiedExploit, error) {
	var result messages.VerifiedExploit

	// Run cluster verification
	clusterResult, err := VerifyWithCluster(ctx, exploitPath, finding)
	if err != nil {
		return result, fmt.Errorf("cluster verification failed: %w", err)
	}

	// Extract attacker model from PoC
	attackerModel := extractAttackerModel(exploitPath)
	if attackerModel == "unknown" {
		slog.Warn("PoC missing attacker model metadata", "path", exploitPath)
		// Don't fail — let the operator decide
	}

	// Check for modified validator
	modified, justification := checkModifiedValidator(exploitPath)
	if modified && justification == "" {
		slog.Warn("PoC modifies validator but lacks justification")
		// Justification is required per scope.md:25-26 — flag it
	}

	result = messages.VerifiedExploit{
		ID:                     generateID(),
		FindingID:              finding.ID,
		PoCPath:                exploitPath,
		ImpactType:             clusterResult.ImpactType,
		ValidatedAgainstConfig: clusterResult.ValidatedAgainstConfig,
		AttackerModel:          attackerModel,
		ModifiedValidator:      modified,
		Justification:          justification,
		VerifiedAt:             time.Now().UTC(),
	}

	slog.Info("firedancer verification passed",
		"id", result.ID,
		"impact_type", result.ImpactType,
		"attacker_model", attackerModel,
		"modified_validator", modified,
	)

	return result, nil
}

func verifySolidityEVM(ctx context.Context, client llm.Client, node *axl.Node, exploitPath string, finding messages.Finding, sourcePath string) (messages.VerifiedExploit, error) {
	var result messages.VerifiedExploit

	forkURL := os.Getenv("MAINNET_RPC_URL")
	tr, err := RunForkTest(exploitPath, forkURL, 0)
	if err != nil {
		return result, fmt.Errorf("live-fork test failed: %w", err)
	}
	if !tr.Passed {
		return result, fmt.Errorf("live-fork test did not pass")
	}

	minDrain := 1000.0
	if v := os.Getenv("MIN_DRAIN_THRESHOLD_USD"); v != "" {
		fmt.Sscanf(v, "%f", &minDrain)
	}

	drain, err := ExtractDrainUsd(tr.Logs)
	if err != nil {
		return result, fmt.Errorf("drain extraction: %w", err)
	}
	if drain < minDrain {
		return result, fmt.Errorf("drain %f below threshold %f", drain, minDrain)
	}

	if os.Getenv("ENABLE_DIFFERENTIAL") == "true" {
		_, err = DifferentialCheck(ctx, finding.ID)
		if err != nil {
			return result, fmt.Errorf("differential check: %w", err)
		}
	}

	result = messages.VerifiedExploit{
		ID:        generateID(),
		FindingID: finding.ID,
		PoCPath:   exploitPath,
		ImpactType: "drain",
		AttackerModel: "onchain",
		VerifiedAt: time.Now().UTC(),
	}

	b, _ := json.Marshal(result)
	node.Publish("exploit/verified", b)

	return result, nil
}
