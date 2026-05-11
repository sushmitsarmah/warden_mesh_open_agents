// Package cloak provides a Go HTTP client for the Cloak TypeScript service.
//
// The Cloak service runs as a sidecar (default :4000) and wraps the Cloak SDK.
// This client is called from the disclosure pipeline to:
//   - Disburse bug-bounty rewards privately (batch disbursement, amounts hidden)
//   - Issue a viewing key for the protocol's finance team
//   - Generate stealth addresses for researchers
package cloak

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type Mint string

const (
	MintUSDC Mint = "USDC"
	MintUSDT Mint = "USDT"
	MintSOL  Mint = "SOL"
)

type ViewingKeyScope string

const (
	ScopeFull        ViewingKeyScope = "full"
	ScopeAmountsOnly ViewingKeyScope = "amounts_only"
	ScopeTimeLimited ViewingKeyScope = "time_limited"
)

type PayoutRecipient struct {
	StealthAddress string `json:"stealthAddress"`
	AmountUsdc     int64  `json:"amountUsdc"`
	Label          string `json:"label,omitempty"`
}

type PayoutRequest struct {
	FindingID          string          `json:"findingId"`
	Recipients         []PayoutRecipient `json:"recipients"`
	Mint               Mint            `json:"mint"`
	GenerateViewingKey bool            `json:"generateViewingKey"`
	ViewingKeyLabel    string          `json:"viewingKeyLabel,omitempty"`
	ViewingKeyScope    ViewingKeyScope `json:"viewingKeyScope,omitempty"`
}

type ViewingKeyRecord struct {
	ID         string          `json:"id"`
	Scope      ViewingKeyScope `json:"scope"`
	Label      string          `json:"label"`
	CreatedAt  string          `json:"createdAt"`
	ExpiresAt  string          `json:"expiresAt,omitempty"`
	FindingIDs []string        `json:"findingIds"`
}

type PayoutResult struct {
	TxSignature    string            `json:"txSignature"`
	FindingID      string            `json:"findingId"`
	TotalAmountUSDC int64            `json:"totalAmountUsdc"`
	RecipientCount int               `json:"recipientCount"`
	ViewingKey     *ViewingKeyRecord `json:"viewingKey,omitempty"`
	ShieldedAt     string            `json:"shieldedAt"`
}

type StealthAddressResult struct {
	StealthAddress string `json:"stealthAddress"`
	Note           string `json:"note"`
}

// Client is a lightweight HTTP client for the Cloak sidecar service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Client from CLOAK_SERVICE_URL env var (default: http://127.0.0.1:4000).
func NewClient() *Client {
	baseURL := os.Getenv("CLOAK_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:4000"
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// IsAvailable pings the Cloak service health endpoint.
func (c *Client) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GenerateStealthAddress generates a researcher stealth address via the Cloak service.
// The researcher's real wallet never appears on-chain.
func (c *Client) GenerateStealthAddress(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/audit/stealth", nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloak stealth request: %w", err)
	}
	defer resp.Body.Close()

	var result StealthAddressResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("cloak stealth decode: %w", err)
	}

	return result.StealthAddress, nil
}

// PayBounty executes a private batch disbursement for a verified finding.
// Returns the payout result including the viewing key ID for the protocol.
//
// If the Cloak service is unreachable, it logs a warning and returns nil, nil
// so the caller can fall back to alternative payment methods without crashing.
func (c *Client) PayBounty(ctx context.Context, req PayoutRequest) (*PayoutResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+"/payout", bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		slog.Warn("cloak service unreachable — skipping private payout", "err", err)
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		// Idempotent: payout already executed for this finding
		var existing PayoutResult
		if err := json.NewDecoder(resp.Body).Decode(&existing); err == nil {
			slog.Info("cloak payout already exists for finding", "finding_id", req.FindingID)
			return &existing, nil
		}
	}

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cloak payout failed %d: %s", resp.StatusCode, string(b))
	}

	var result PayoutResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("cloak payout decode: %w", err)
	}

	return &result, nil
}
