# KeeperHub x402 Payment Integration

## Overview

The x402 payment integration enables monetization of vulnerability reports using HTTP 402 Payment Required status codes. This implementation uses **KeeperHub** as the payment gateway for USDC transactions on-chain.

## Architecture

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│  Disclosure │─────>│ Payment Gate │─────>│  KeeperHub  │
│  Publisher  │      │   (x402)     │      │     API     │
└─────────────┘      └──────────────┘      └─────────────┘
                            │
                            v
                    ┌──────────────┐
                    │ HTTP Handler │
                    │  (402/200)   │
                    └──────────────┘
```

### Flow

1. **Report Creation**: Orchestrator generates teaser and full vulnerability reports
2. **Payment Gate**: Creates a gated report with KeeperHub payment URL
3. **Disclosure**: Publishes teaser with payment URL to AXL mesh
4. **Access Control**: HTTP handler returns 402 for unpaid, serves full report after payment
5. **Webhook**: KeeperHub notifies when payment completes, unlocks full report

## Configuration

### Environment Variables

Add to `.env`:

```bash
# KeeperHub - x402 payments
KEEPERHUB_API_KEY=your_api_key_here
KEEPERHUB_X402_ENDPOINT=https://api.keeperhub.io/v1/payments
REPORT_PRICE_USD=1000
REPORT_SERVER_BASE_URL=http://localhost:8080
```

### Configuration Details

- `KEEPERHUB_API_KEY`: Your KeeperHub API key (leave empty for stub mode)
- `KEEPERHUB_X402_ENDPOINT`: KeeperHub payment API endpoint
- `REPORT_PRICE_USD`: Default price per vulnerability report (USD)
- `REPORT_SERVER_BASE_URL`: Base URL for your report server

## Implementation

### Creating Gated Reports

```go
import "github.com/local/swarm/orchestrator/internal/x402"

// Initialize payment gate
gate := x402.NewGate("http://localhost:8080")

// Create gated report
report, err := gate.CreateGatedReport(
    "finding-123",           // Finding ID
    "/tmp/teaser.md",        // Teaser file path
    "/tmp/full-report.md",   // Full report file path
    1000.0,                  // Price in USD
)

fmt.Printf("Payment URL: %s\n", report.PaymentURL)
```

### HTTP Handler for Report Access

```go
// Serve gated report
http.HandleFunc("/reports/"+report.ID, gate.Handler(report.ID))
```

**Behavior**:
- **Unpaid**: Returns HTTP 402 with payment information
- **Paid**: Returns HTTP 200 with full report content

### Example 402 Response

```json
{
  "error": "Payment required",
  "report_id": "6206e6f6b0faf7cf1383834b0f37cd6d",
  "price_usd": 1000.0,
  "payment_url": "https://pay.keeperhub.io/xyz123",
  "message": "Complete payment to access the full vulnerability report"
}
```

### Webhook Handler

```go
// Set up webhook endpoint
http.HandleFunc("/webhook", gate.WebhookHandler())
```

**Webhook Payload** (from KeeperHub):
```json
{
  "payment_id": "pay_xyz123",
  "status": "completed",
  "tx_hash": "0x1234..."
}
```

## Integration with Disclosure System

```go
import (
    "github.com/local/swarm/orchestrator/internal/disclosure"
    "github.com/local/swarm/orchestrator/internal/x402"
    "github.com/local/swarm/orchestrator/pkg/axl"
)

// Initialize components
node := axl.NewNode(axlURL, peerKeys)
gate := x402.NewGate(baseURL)
publisher := disclosure.NewPublisher(node, gate)

// Publish gated disclosure
err := publisher.Publish(ctx, exploit, teaserPath, fullPath)
```

This automatically:
1. Creates payment-gated report
2. Generates KeeperHub payment URL
3. Publishes disclosure to AXL mesh with payment URL

## Testing

### Test Command

```bash
cd services/orchestrator-go
make build
./bin/test-x402
```

### Expected Output

```
✅ Gated Report Created
   Report ID: 6206e6f6b0faf7cf1383834b0f37cd6d
   Price: $1000.00 USD
   Payment URL: http://localhost:8080/reports/6206e6f6b0faf7cf1383834b0f37cd6d/pay

📝 Test 1: Accessing report WITHOUT payment
   Status Code: 402
   ✅ Correctly returned 402 Payment Required
   Payment Info: {"error":"Payment required",...}

📝 Test 2: Simulating payment webhook
   ℹ️  Payment not found (expected in stub mode): payment not found: test-payment-123

📝 Test 3: Payment flow demonstration
   In production:
   1. User clicks payment URL
   2. User pays via KeeperHub (USDC)
   3. KeeperHub sends webhook to /webhook
   4. System marks payment as complete
   5. User can access full report

🎯 x402 Payment Gate Test Complete
```

## Stub Mode vs Production

### Stub Mode (No API Key)

When `KEEPERHUB_API_KEY` is not set:
- Returns local payment URLs: `http://localhost:8080/reports/{id}/pay`
- Payment URLs are placeholders
- Reports remain gated (return 402)
- Useful for development and testing

### Production Mode (With API Key)

When `KEEPERHUB_API_KEY` is configured:
- Creates real payment URLs via KeeperHub API
- Users pay with USDC on-chain
- Webhook receives payment confirmations
- Full reports unlock automatically after payment

## Payment Flow Diagram

```
User Request (no payment)
        │
        v
┌───────────────────┐
│  HTTP 402 Response│
│  + Payment URL    │
└───────────────────┘
        │
        v
┌───────────────────┐
│  User pays via    │
│   KeeperHub       │
└───────────────────┘
        │
        v
┌───────────────────┐
│  Webhook callback │
│  marks paid       │
└───────────────────┘
        │
        v
┌───────────────────┐
│  HTTP 200 Response│
│  + Full Report    │
└───────────────────┘
```

## KeeperHub API Integration

### Payment Creation Request

```json
POST https://api.keeperhub.io/v1/payments
Authorization: Bearer YOUR_API_KEY
Content-Type: application/json

{
  "amount": 1000.0,
  "currency": "USDC",
  "description": "Vulnerability Report: finding-123",
  "metadata": {
    "report_id": "6206e6f6b0faf7cf1383834b0f37cd6d",
    "finding_id": "finding-123",
    "type": "vulnerability_report"
  }
}
```

### Payment Creation Response

```json
{
  "payment_id": "pay_xyz123",
  "payment_url": "https://pay.keeperhub.io/xyz123",
  "amount": 1000.0,
  "expires_at": "2026-05-03T12:00:00Z"
}
```

## Security Considerations

1. **API Key Protection**: Never commit `KEEPERHUB_API_KEY` to version control
2. **Webhook Verification**: In production, verify webhook signatures from KeeperHub
3. **Payment Expiration**: Payments expire after a set time (configured in KeeperHub)
4. **HTTPS Only**: Use HTTPS in production for webhook endpoints
5. **Rate Limiting**: Implement rate limiting on payment endpoints to prevent abuse

## Error Handling

```go
report, err := gate.CreateGatedReport(findingID, teaserPath, fullPath, priceUSD)
if err != nil {
    // Handle error cases:
    // - KeeperHub API error
    // - Network timeout
    // - Invalid configuration
    slog.Error("failed to create gated report", "err", err)
    return err
}
```

## Monitoring

Track payment metrics:
- Total reports created
- Payment success rate
- Average time to payment
- Revenue per vulnerability severity

Example logging:
```go
slog.Info("payment confirmed",
    "payment_id", paymentID,
    "report_id", payment.ReportID,
    "amount_usd", payment.AmountUSD,
    "tx_hash", txHash,
)
```

## Next Steps

1. **Set up KeeperHub account** and obtain API key
2. **Configure webhook endpoint** on a public URL
3. **Test payment flow** with testnet USDC
4. **Monitor payments** via KeeperHub dashboard
5. **Adjust pricing** based on vulnerability severity

## Resources

- KeeperHub Documentation: https://docs.keeperhub.com/api
- HTTP 402 Specification: https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/402
- Test Implementation: `services/orchestrator-go/cmd/test-x402/main.go`
