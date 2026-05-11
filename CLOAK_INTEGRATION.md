# Cloak Integration — Private Bug-Bounty Payouts

This document describes how Warden Mesh uses the [Cloak](https://cloak.network) private
execution layer to disburse bug-bounty rewards on Solana without exposing amounts or
addresses on-chain.

---

## Why Cloak

Every bug-bounty payout on Solana is publicly visible by default: the amount, the
recipient wallet, and the timestamp are permanently indexed on every block explorer.
This creates concrete problems:

- **Researcher exposure** — a researcher who receives 50 USDC from a protocol treasury
  address is now linked to that protocol's vulnerability in a permanent public record.
- **Protocol signalling** — a payout telegraphs that a vulnerability was found, how
  serious it was (by amount), and approximately when. Competitors parse this in real time.
- **Chilling effect** — researchers and protocols alike avoid large payouts or structure
  them to avoid attention, which misaligns incentives.

Cloak fixes this with a UTXO shielded pool on Solana and client-side Groth16 proofs.
Amounts are hidden, addresses are hidden, and the public ledger shows nothing meaningful.
Selective disclosure via viewing keys means the protocol's finance team, external auditor,
or a regulator can still see exactly what they need.

---

## Architecture

```
Orchestrator (Go)
    │
    │  POST /payout       (if Cloak sidecar is up)
    ▼
Cloak Service (TypeScript + Cloak SDK)   ← services/cloak-ts/
    │
    │  batchDisbursement()  → Solana shielded pool (Groth16 proof)
    │  generateViewingKey() → scoped audit key for protocol
    │  generateStealthAddress() → one-time address for researcher
    ▼
Cloak Protocol (Solana mainnet)

    ▲
    │  viewing key
    │
Audit Dashboard (Next.js)   ← services/cloak-ts/audit-dashboard/
    │  GET /audit/history?keyId=
    │  GET /audit/export?keyId=&fmt=csv
```

### Flow for a verified finding

1. Orchestrator verifies an exploit → calls `disclosure.Publisher.Publish()`
2. Publisher checks `IsAvailable()` on the Cloak sidecar
3. If available:
   a. `GenerateStealthAddress()` → one-time address for the researcher
   b. `PayBounty()` → `POST /payout` to Cloak service
   c. Cloak service calls `sdk.batchDisbursement()` → shielded Solana tx
   d. Cloak service calls `sdk.generateViewingKey()` → key for protocol finance
   e. `CloakTxSignature` and `CloakViewingKeyID` are attached to the `Disclosure` message
4. Disclosure is published to AXL + recorded on 0G iNFT as usual
5. Protocol finance team opens the Audit Dashboard, enters the `viewingKeyId`,
   and sees the full decrypted history

If the Cloak sidecar is not running the pipeline continues without shielded payouts —
the x402 report URL is still generated and the iNFT is still recorded.

---

## Components

### `services/cloak-ts/` — Cloak sidecar (TypeScript)

Express server wrapping `@cloak-network/sdk`.

| Endpoint | Description |
|---|---|
| `POST /payout` | Batch-disburse shielded USDC/USDT/SOL; optionally issue viewing key |
| `GET /payout/:findingId` | Get payout status (idempotent) |
| `GET /payout` | List all payouts |
| `POST /viewing-key` | Issue a standalone viewing key |
| `GET /viewing-key` | List all keys (metadata only, never raw) |
| `GET /viewing-key/:id/raw` | Return raw key string (add auth in production) |
| `DELETE /viewing-key/:id` | Revoke a key |
| `GET /audit/history?keyId=` | Decrypt history (stored key) |
| `POST /audit/history` | Decrypt history (raw key in body) |
| `GET /audit/export?keyId=&fmt=csv` | Export as CSV or JSON |
| `GET /audit/stealth` | Generate researcher stealth address |
| `GET /health` | Health check |

### `services/cloak-ts/audit-dashboard/` — Compliance dashboard (Next.js)

Finance-team and auditor UI running on port 3100.

Features:
- Input a **Key ID** (stored in the sidecar) or a **raw viewing key** received out-of-band
- Decrypts and displays the shielded transaction history
- Scope badge: **FULL AUDIT** / **AMOUNTS ONLY** / **TIME-LIMITED**
- Per-transaction: timestamp, type, amount, finding ID, shielded addresses (if scope permits), tx signature
- **Export CSV** and **Export JSON** buttons
- **Generate stealth address** panel — for issuing one-time receive addresses to researchers

### `services/orchestrator-go/internal/cloak/client.go` — Go HTTP client

Thin wrapper around the Cloak sidecar REST API:
- `IsAvailable(ctx)` — ping before attempting any call
- `GenerateStealthAddress(ctx)` — get a researcher receive address
- `PayBounty(ctx, req)` — execute batch disbursement; returns `PayoutResult`

All errors are soft: if the sidecar is unreachable the publisher logs a warning and
continues without shielded payouts.

---

## Cloak SDK Capabilities Used

| Capability | Where |
|---|---|
| **Private transfer** (USDC/USDT/SOL) | Every payout via `batchDisbursement` |
| **Batch disbursement** | Single shielded tx fans out to all recipients |
| **Stealth addresses** | Researcher receive address; never linked to real wallet |
| **Viewing keys** | Protocol finance team, external auditor, time-limited regulator key |

---

## Viewing Key Scopes

| Scope | What is visible | Use case |
|---|---|---|
| `full` | Amount + sender + recipient + timestamp | Internal finance team |
| `amounts_only` | Amount + timestamp only | External auditor (no addresses) |
| `time_limited` | Full, but key expires at a fixed time | Regulatory request with time bound |

---

## Setup

### 1. Start the Cloak sidecar

```bash
cd services/cloak-ts
yarn install
CLOAK_RPC_URL=https://api.mainnet-beta.solana.com \
CLOAK_PRIVATE_KEY=<base58-keypair> \
yarn dev
```

Or via Docker:
```bash
docker build -t warden-cloak-ts services/cloak-ts
docker run -e CLOAK_RPC_URL=... -e CLOAK_PRIVATE_KEY=... -p 4000:4000 warden-cloak-ts
```

### 2. Start the audit dashboard

```bash
cd services/cloak-ts/audit-dashboard
yarn install
NEXT_PUBLIC_CLOAK_SERVICE_URL=http://localhost:4000 yarn dev
# opens on http://localhost:3100
```

### 3. Configure the orchestrator

Add to `.env.local`:

```bash
# Cloak private payout sidecar
CLOAK_SERVICE_URL=http://127.0.0.1:4000
CLOAK_BOUNTY_AMOUNT_USDC=5000000   # 5.00 USDC (6 decimal places)
```

The orchestrator auto-detects the sidecar via `CLOAK_SERVICE_URL`. If the variable is
not set or the service is unreachable, disclosures are published normally without
shielded payouts.

### 4. Makefile targets

```bash
make run-cloak          # Start the Cloak sidecar (port 4000)
make run-audit-dashboard  # Start the audit dashboard (port 3100)
```

---

## Security Notes

- The raw viewing key is only returned from `GET /viewing-key/:id/raw`. In production,
  add an authentication layer (e.g. challenge-response with the protocol's signing key)
  before exposing this endpoint.
- The sidecar holds the shielded wallet's private key (`CLOAK_PRIVATE_KEY`). Treat it
  like an HSM secret — do not log it, do not commit it.
- Viewing keys are scoped; issue the narrowest scope sufficient for the use case.
- Time-limited keys expire server-side; the sidecar checks `expiresAt` on every
  `/audit/history` request.
