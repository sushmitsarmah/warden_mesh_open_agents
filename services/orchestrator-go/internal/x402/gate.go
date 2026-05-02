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
	endpoint    string
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

// KeeperHubPaymentRequest is sent to KeeperHub to create a payment
type KeeperHubPaymentRequest struct {
	Amount      float64           `json:"amount"`
	Currency    string            `json:"currency"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

// KeeperHubPaymentResponse is returned by KeeperHub
type KeeperHubPaymentResponse struct {
	PaymentID  string  `json:"payment_id"`
	PaymentURL string  `json:"payment_url"`
	Amount     float64 `json:"amount"`
	ExpiresAt  string  `json:"expires_at"`
}

func NewGate(baseURL string) *Gate {
	apiKey := os.Getenv("KEEPERHUB_API_KEY")
	endpoint := os.Getenv("KEEPERHUB_X402_ENDPOINT")

	if endpoint == "" {
		endpoint = "https://api.keeperhub.io/v1/payments" // Default endpoint
	}

	return &Gate{
		apiKey:   apiKey,
		endpoint: endpoint,
		baseURL:  baseURL,
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

// createKeeperHubPayment creates a payment via KeeperHub API
func (g *Gate) createKeeperHubPayment(report *GatedReport) (string, error) {
	req := KeeperHubPaymentRequest{
		Amount:      report.PriceUSD,
		Currency:    "USDC",
		Description: fmt.Sprintf("Vulnerability Report: %s", report.FindingID),
		Metadata: map[string]string{
			"report_id":  report.ID,
			"finding_id": report.FindingID,
			"type":       "vulnerability_report",
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequest("POST", g.endpoint, bytes.NewReader(body))
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

	var khResp KeeperHubPaymentResponse
	if err := json.Unmarshal(respBody, &khResp); err != nil {
		return "", fmt.Errorf("failed to parse KeeperHub response: %w", err)
	}

	// Track the payment
	payment := &Payment{
		ID:        khResp.PaymentID,
		ReportID:  report.ID,
		AmountUSD: khResp.Amount,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}

	g.mu.Lock()
	g.payments[payment.ID] = payment
	g.mu.Unlock()

	slog.Info("created KeeperHub payment",
		"payment_id", khResp.PaymentID,
		"amount", khResp.Amount,
		"url", khResp.PaymentURL,
	)

	return khResp.PaymentURL, nil
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

		var webhook struct {
			PaymentID string `json:"payment_id"`
			Status    string `json:"status"`
			TxHash    string `json:"tx_hash"`
		}

		if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
			http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
			return
		}

		if webhook.Status == "completed" || webhook.Status == "paid" {
			if err := g.MarkPaid(webhook.PaymentID, webhook.TxHash); err != nil {
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
