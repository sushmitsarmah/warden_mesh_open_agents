package x402

import (
	"fmt"
	"net/http"
)

// Gate holds payment endpoint config.
type Gate struct {
	BaseURL string
}

func NewGate(baseURL string) *Gate {
	return &Gate{BaseURL: baseURL}
}

func (g *Gate) Handler(reportID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = fmt.Fprintf(w, "Payment required to access report %s\n", reportID)
	}
}
