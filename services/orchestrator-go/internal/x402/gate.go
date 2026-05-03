package x402

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// Gate manages payment-gated content delivery
type Gate struct {
	apiKey      string
	apiBaseURL  string
	workflowID  string
	baseURL     string
	httpClient  *http.Client
	mu          sync.RWMutex
	reports     map[string]*GatedReport
	payments    map[string]*Payment // paymentID -> Payment
}

// GatedReport represents a report behind a paywall
type GatedReport struct {
	ID          string
	FindingID   string
	TeaserPath  string
	FullPath    string
	PriceUSD    float64
	CreatedAt   time.Time
	PaymentURL  string
}

// Payment tracks a payment transaction
type Payment struct {
	ID         string
	ReportID   string
	AmountUSD  float64
	Status     string // "pending", "paid", "expired"
	CreatedAt  time.Time
	PaidAt     *time.Time
	TxHash     string
}

// KeeperHubWorkflowRequest is sent to trigger the payment generation workflow
type KeeperHubWorkflowRequest struct {
	Inputs map[string]interface{} `json:"inputs"`
}

// KeeperHubWorkflowResponse captures the checkout URL from the workflow result
type KeeperHubWorkflowResponse struct {
	ExecutionID string `json:"execution_id"`
	Status      string `json:"status"`
	Result      struct {
		PaymentURL string `json:"payment_url"`
	} `json:"result"`
}

// KeeperHubWebhookPayload is the payload KeeperHub sends when the workflow completes
type KeeperHubWebhookPayload struct {
	ExecutionID string `json:"execution_id"`
	Status      string `json:"status"`
	Outputs     struct {
		TxHash string `json:"tx_hash"`
	} `json:"outputs"`
}

func NewGate(baseURL string) *Gate {
	apiKey := os.Getenv("KEEPERHUB_API_KEY")

	apiBaseURL := os.Getenv("KEEPERHUB_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "https://app.keeperhub.com/api"
	}

	workflowID := os.Getenv("KEEPERHUB_WORKFLOW_ID")

	return &Gate{
		apiKey:     apiKey,
		apiBaseURL: apiBaseURL,
		workflowID: workflowID,
		baseURL:    baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		reports:  make(map[string]*GatedReport),
		payments: make(map[string]*Payment),
	}
}

// CreateGatedReport creates a payment-gated report
func (g *Gate) CreateGatedReport(findingID, teaserPath, fullPath string, priceUSD float64) (*GatedReport, error) {
	reportID := generateReportID()

	report := &GatedReport{
		ID:         reportID,
		FindingID:  findingID,
		TeaserPath: teaserPath,
		FullPath:   fullPath,
		PriceUSD:   priceUSD,
		CreatedAt:  time.Now().UTC(),
	}

	// Create payment URL
	paymentURL, err := g.createPaymentURL(report)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment URL: %w", err)
	}

	report.PaymentURL = paymentURL

	g.mu.Lock()
	g.reports[reportID] = report
	g.mu.Unlock()

	slog.Info("created gated report",
		"report_id", reportID,
		"price_usd", priceUSD,
		"payment_url", paymentURL,
	)

	return report, nil
}

// createPaymentURL creates a payment URL via KeeperHub or returns a stub URL
func (g *Gate) createPaymentURL(report *GatedReport) (string, error) {
	// If KeeperHub is configured, create real payment
	if g.apiKey != "" {
		return g.createKeeperHubPayment(report)
	}

	// Stub mode: return local URL
	stubURL := fmt.Sprintf("%s/reports/%s/pay", g.baseURL, report.ID)
	slog.Warn("KeeperHub not configured, using stub payment URL", "url", stubURL)
	return stubURL, nil
}

// createKeeperHubPayment triggers the KeeperHub workflow to generate a payment URL
func (g *Gate) createKeeperHubPayment(report *GatedReport) (string, error) {
	req := KeeperHubWorkflowRequest{
		Inputs: map[string]interface{}{
			"amount":      report.PriceUSD,
			"currency":    "USDC",
			"description": fmt.Sprintf("Vulnerability Report: %s", report.FindingID),
			"report_id":   report.ID,
			"finding_id":  report.FindingID,
			"type":        "vulnerability_report",
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/workflows/%s/execute", g.apiBaseURL, g.workflowID)

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("KeeperHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("KeeperHub API error %d: %s", resp.StatusCode, string(respBody))
	}

	var khResp KeeperHubWorkflowResponse
	if err := json.Unmarshal(respBody, &khResp); err != nil {
		return "", fmt.Errorf("failed to parse KeeperHub response: %w", err)
	}

	// Track the payment
	payment := &Payment{
		ID:        khResp.ExecutionID,
		ReportID:  report.ID,
		AmountUSD: report.PriceUSD,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}

	g.mu.Lock()
	g.payments[payment.ID] = payment
	g.mu.Unlock()

	slog.Info("created KeeperHub workflow payment",
		"execution_id", khResp.ExecutionID,
		"status", khResp.Status,
		"url", khResp.Result.PaymentURL,
	)

	return khResp.Result.PaymentURL, nil
}

// Handler returns an HTTP handler for accessing a gated report
func (g *Gate) Handler(reportID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g.mu.RLock()
		report, exists := g.reports[reportID]
		g.mu.RUnlock()

		if !exists {
			http.Error(w, "Report not found", http.StatusNotFound)
			return
		}

		// Check if payment has been made
		paid := g.checkPaymentStatus(reportID)

		if !paid {
			// Return 402 Payment Required with payment info
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)

			response := map[string]interface{}{
				"error":       "Payment required",
				"report_id":   reportID,
				"price_usd":   report.PriceUSD,
				"payment_url": report.PaymentURL,
				"message":     "Complete payment to access the full vulnerability report",
			}

			json.NewEncoder(w).Encode(response)
			return
		}

		// Payment confirmed, serve the full report
		http.ServeFile(w, r, report.FullPath)
	}
}

// checkPaymentStatus checks if a report has been paid for
func (g *Gate) checkPaymentStatus(reportID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, payment := range g.payments {
		if payment.ReportID == reportID && payment.Status == "paid" {
			return true
		}
	}

	return false
}

// MarkPaid marks a payment as completed (webhook handler)
func (g *Gate) MarkPaid(paymentID, txHash string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	payment, exists := g.payments[paymentID]
	if !exists {
		return fmt.Errorf("payment not found: %s", paymentID)
	}

	now := time.Now().UTC()
	payment.Status = "paid"
	payment.PaidAt = &now
	payment.TxHash = txHash

	slog.Info("payment confirmed",
		"payment_id", paymentID,
		"report_id", payment.ReportID,
		"tx_hash", txHash,
	)

	return nil
}

// WebhookHandler handles payment confirmation webhooks from KeeperHub
func (g *Gate) WebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var webhook KeeperHubWebhookPayload

		if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
			http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
			return
		}

		if webhook.Status == "completed" || webhook.Status == "success" {
			if err := g.MarkPaid(webhook.ExecutionID, webhook.Outputs.TxHash); err != nil {
				slog.Error("failed to mark payment as paid", "err", err)
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}

func generateReportID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
