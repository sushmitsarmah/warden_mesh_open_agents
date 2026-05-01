# Agent Implementation Plan: Autonomous Web3 Security Swarm

> **For AI coding agents (Claude Code, Cursor, Aider, etc.)**
> This document is structured for sequential execution. Do **not** skip phases. Each task has a verification gate; do not advance until the gate passes.

---

## How To Use This Plan

**Execution model.** Work through phases in order. Within a phase, tasks are numbered (e.g. `2.3`) and must be completed in sequence. Each task has five fields:

- **Goal** — one-sentence purpose.
- **Files** — exact paths to create or modify.
- **Steps** — atomic actions to perform.
- **Verify** — a shell command and the expected outcome. Do not advance if it fails.
- **Notes** — gotchas, alternatives, or external docs to consult.

**When in doubt, stop and ask.** If a step references a third-party SDK (Gensyn AXL, KeeperHub, 0G) and the API surface in this document doesn't match the real docs, **stop and ask the human operator** rather than inventing function signatures. Hackathon SDKs change weekly; this plan describes structure, not literal call signatures for those three.

**Repository root convention.** All paths are relative to the repo root, which we'll call `$REPO`. The agent should work inside `$REPO` for every task.

**Validation gate convention.** A "verify" command that exits with code 0 and produces the described output is a pass. Anything else is a fail — fix before advancing.

---

## Prerequisites Checklist

Before Phase 0, confirm the following are available. If any are missing, install them or stop and report.

| Requirement | Version | Verify |
|---|---|---|
| Go | ≥ 1.22 | `go version` |
| Rust + Cargo | ≥ 1.75 | `cargo --version` |
| Foundry (forge, anvil, cast) | latest | `forge --version` |
| Node.js | ≥ 20 | `node --version` |
| Python | ≥ 3.11 | `python3 --version` |
| Slither | latest | `slither --version` |
| Aderyn | latest | `aderyn --version` |
| Docker | ≥ 24 | `docker --version` |
| `make` | any | `make --version` |
| `jq` | any | `jq --version` |

**API keys / accounts required** (collect these before starting and put in `.env`):

- Etherscan API key
- Alchemy or Infura RPC endpoint (Sepolia + mainnet fork)
- LLM API key (Anthropic or OpenAI)
- GitHub personal access token (read-only on public repos is enough)
- 0G testnet RPC + funded testnet wallet
- KeeperHub credentials (per their hackathon onboarding)
- Optional: Immunefi API access (if available)

**Reference target for testing.** We use **Damn Vulnerable DeFi v4** as the standard test corpus. Clone it locally during Phase 0 — it's the input fixture for every later verification step.

---

## Architecture Reference

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────────┐
│  Scout (Go)     │────▶│  AXL Mesh    │────▶│  Auditor (Rust)  │
│  - mempool      │     │  pub/sub     │     │  - alloy-rs fetch│
│  - github       │     │  topics:     │     │  - aderyn        │
│  - immunefi     │     │  - targets   │     │  - slither       │
└─────────────────┘     │  - findings  │     └──────────────────┘
                        │  - verified  │              │
                        │  - published │              ▼
                        └──────────────┘     ┌──────────────────┐
                              ▲              │ Orchestrator (Go)│
                              │              │ - LLM exploit gen│
                              │              │ - Foundry verify │
                              └──────────────│ - x402 publish   │
                                             │ - 0G iNFT update │
                                             └──────────────────┘
```

**Topics on the AXL mesh:**
- `targets/discovered` — Scout → Auditor
- `analysis/findings` — Auditor → Orchestrator
- `exploit/verified` — Orchestrator (internal, also consumed by dashboard)
- `disclosure/published` — Orchestrator → 0G state updater

**Schema files** in `proto/` are the source of truth for cross-language messages.

---

## Phase 0 — Environment & Scaffolding

**Phase goal:** A buildable monorepo with the directory layout, configs, and shared schemas every later phase depends on.

### Task 0.1 — Create the directory tree

**Goal:** Lay down the canonical folder structure.

**Files:** Directories only.

**Steps:**

1. From `$REPO`, run:
   ```bash
   mkdir -p services/scout-go/{cmd,internal,pkg}
   mkdir -p services/auditor-rs/src
   mkdir -p services/orchestrator-go/{cmd,internal,pkg}
   mkdir -p services/dashboard/src
   mkdir -p contracts/{src,test,script}
   mkdir -p proto
   mkdir -p prompts
   mkdir -p infra/{docker,scripts}
   mkdir -p test-fixtures/dvd  # Damn Vulnerable DeFi
   mkdir -p docs
   ```

**Verify:**
```bash
find . -maxdepth 2 -type d | sort
```
Expected: all directories above are listed.

**Notes:** Do not create source files yet — empty dirs are intentional.

---

### Task 0.2 — Initialize Go modules

**Goal:** Two independent Go modules (scout, orchestrator) ready to compile.

**Files:** `services/scout-go/go.mod`, `services/orchestrator-go/go.mod`.

**Steps:**

1. `cd services/scout-go && go mod init github.com/<org>/swarm/scout`
2. `cd ../orchestrator-go && go mod init github.com/<org>/swarm/orchestrator`
3. Replace `<org>` with the actual GitHub org. If unknown, use `local`.
4. In each, create a stub `cmd/main.go`:
   ```go
   package main
   import "fmt"
   func main() { fmt.Println("scout: stub") } // change name per service
   ```

**Verify:**
```bash
cd services/scout-go && go build ./... && cd -
cd services/orchestrator-go && go build ./... && cd -
```
Expected: no output, exit code 0, two binaries written to each service dir's bin path or implicit build cache.

---

### Task 0.3 — Initialize Rust workspace

**Goal:** Auditor service compiles as a binary crate.

**Files:** `services/auditor-rs/Cargo.toml`, `services/auditor-rs/src/main.rs`.

**Steps:**

1. `cd services/auditor-rs`
2. Create `Cargo.toml`:
   ```toml
   [package]
   name = "auditor"
   version = "0.1.0"
   edition = "2021"

   [dependencies]
   tokio = { version = "1", features = ["full"] }
   serde = { version = "1", features = ["derive"] }
   serde_json = "1"
   alloy = { version = "0.5", features = ["full"] }
   anyhow = "1"
   tracing = "0.1"
   tracing-subscriber = "0.3"
   ```
3. Create `src/main.rs`:
   ```rust
   #[tokio::main]
   async fn main() -> anyhow::Result<()> {
       tracing_subscriber::fmt::init();
       tracing::info!("auditor: stub");
       Ok(())
   }
   ```

**Verify:**
```bash
cd services/auditor-rs && cargo build
```
Expected: builds cleanly. First build will take a few minutes.

**Notes:** If `alloy 0.5` errors on a feature mismatch, pin to whatever the latest stable is and report the version used.

---

### Task 0.4 — Initialize Foundry project

**Goal:** Solidity workspace ready for the iNFT contract and exploit tests.

**Files:** `contracts/foundry.toml`, `contracts/src/.gitkeep`, `contracts/test/.gitkeep`.

**Steps:**

1. `cd contracts && forge init --no-commit --no-git --force .`
2. Delete the default `src/Counter.sol`, `test/Counter.t.sol`, `script/Counter.s.sol`.
3. Edit `foundry.toml` to set:
   ```toml
   [profile.default]
   src = "src"
   out = "out"
   libs = ["lib"]
   solc = "0.8.24"
   optimizer = true
   optimizer_runs = 200

   [rpc_endpoints]
   sepolia = "${SEPOLIA_RPC_URL}"
   mainnet = "${MAINNET_RPC_URL}"
   og_testnet = "${OG_RPC_URL}"
   ```

**Verify:**
```bash
cd contracts && forge build
```
Expected: `Compiler run successful` with zero contracts (we deleted defaults). No errors.

---

### Task 0.5 — Create `.env.example` and `.gitignore`

**Goal:** Document required secrets without committing them.

**Files:** `$REPO/.env.example`, `$REPO/.gitignore`.

**Steps:**

1. Create `.env.example`:
   ```bash
   # RPCs
   SEPOLIA_RPC_URL=
   MAINNET_RPC_URL=
   OG_RPC_URL=

   # Explorer / data
   ETHERSCAN_API_KEY=
   GITHUB_TOKEN=
   IMMUNEFI_API_KEY=

   # LLM
   ANTHROPIC_API_KEY=
   # or
   OPENAI_API_KEY=

   # Hackathon SDKs
   AXL_BOOTSTRAP_NODES=
   KEEPERHUB_API_KEY=
   KEEPERHUB_X402_ENDPOINT=

   # Wallets (testnet only — never commit real keys)
   SWARM_PRIVATE_KEY=
   OG_PRIVATE_KEY=

   # Knobs
   MIN_DRAIN_THRESHOLD_USD=1000
   LLM_MAX_RETRIES=3
   DISCLOSURE_WINDOW_DAYS=90
   ```

2. Create `.gitignore`:
   ```
   .env
   *.env.local
   node_modules/
   target/
   contracts/out/
   contracts/cache/
   contracts/lib/
   services/*/bin/
   *.log
   .DS_Store
   /tmp/
   /test-fixtures/dvd/
   ```

**Verify:**
```bash
cat .env.example | grep -c "="
```
Expected: at least 14 (number of env vars defined).

---

### Task 0.6 — Define shared message schemas

**Goal:** A single source of truth for `Target`, `Finding`, `VerifiedExploit`, `Disclosure` that both Go and Rust consume.

**Files:** `proto/messages.json` (JSON Schema, simpler than Protobuf for a hackathon).

**Steps:**

1. Create `proto/messages.json` with four schemas:

   ```json
   {
     "$schema": "http://json-schema.org/draft-07/schema#",
     "definitions": {
       "Target": {
         "type": "object",
         "required": ["id", "kind", "discoveredAt", "priority"],
         "properties": {
           "id": { "type": "string", "description": "uuid v4" },
           "kind": { "enum": ["onchain", "github", "immunefi"] },
           "chainId": { "type": "integer" },
           "address": { "type": "string", "pattern": "^0x[a-fA-F0-9]{40}$" },
           "repo": { "type": "string" },
           "commitSha": { "type": "string" },
           "sourceUrl": { "type": "string" },
           "discoveredAt": { "type": "string", "format": "date-time" },
           "priority": { "type": "number", "minimum": 0, "maximum": 100 },
           "tvlUsd": { "type": "number" }
         }
       },
       "Finding": {
         "type": "object",
         "required": ["id", "targetId", "category", "severity", "tools"],
         "properties": {
           "id": { "type": "string" },
           "targetId": { "type": "string" },
           "category": { "type": "string", "description": "e.g. reentrancy, unchecked-call" },
           "severity": { "enum": ["info", "low", "medium", "high", "critical"] },
           "tools": {
             "type": "array",
             "items": { "enum": ["aderyn", "slither"] },
             "minItems": 1
           },
           "location": {
             "type": "object",
             "properties": {
               "file": { "type": "string" },
               "lineStart": { "type": "integer" },
               "lineEnd": { "type": "integer" }
             }
           },
           "description": { "type": "string" }
         }
       },
       "VerifiedExploit": {
         "type": "object",
         "required": ["id", "findingId", "forgePath", "drainAmountUsd", "verifiedAt"],
         "properties": {
           "id": { "type": "string" },
           "findingId": { "type": "string" },
           "forgePath": { "type": "string" },
           "drainAmountUsd": { "type": "number" },
           "blockNumber": { "type": "integer" },
           "differentialPassed": { "type": "boolean" },
           "verifiedAt": { "type": "string", "format": "date-time" }
         }
       },
       "Disclosure": {
         "type": "object",
         "required": ["id", "exploitId", "lane", "publishedAt"],
         "properties": {
           "id": { "type": "string" },
           "exploitId": { "type": "string" },
           "lane": { "enum": ["bounty", "rescue"] },
           "x402Url": { "type": "string" },
           "txHash": { "type": "string" },
           "publishedAt": { "type": "string", "format": "date-time" }
         }
       }
     }
   }
   ```

**Verify:**
```bash
jq '.definitions | keys' proto/messages.json
```
Expected: `["Disclosure","Finding","Target","VerifiedExploit"]`.

**Notes:** Generate Go and Rust types from this in tasks 0.7 and 0.8 — do not hand-write them in two languages.

---

### Task 0.7 — Generate Go types from schema

**Goal:** Idiomatic Go structs in a shared package.

**Files:** `services/scout-go/pkg/messages/messages.go`, `services/orchestrator-go/pkg/messages/messages.go` (identical content; symlink or copy).

**Steps:**

1. Hand-write Go structs that mirror the JSON schema (the schemas are small enough that a generator is overkill):

   ```go
   package messages

   import "time"

   type TargetKind string
   const (
       TargetOnchain  TargetKind = "onchain"
       TargetGitHub   TargetKind = "github"
       TargetImmunefi TargetKind = "immunefi"
   )

   type Target struct {
       ID           string     `json:"id"`
       Kind         TargetKind `json:"kind"`
       ChainID      int        `json:"chainId,omitempty"`
       Address      string     `json:"address,omitempty"`
       Repo         string     `json:"repo,omitempty"`
       CommitSha    string     `json:"commitSha,omitempty"`
       SourceURL    string     `json:"sourceUrl,omitempty"`
       DiscoveredAt time.Time  `json:"discoveredAt"`
       Priority     float64    `json:"priority"`
       TVLUsd       float64    `json:"tvlUsd,omitempty"`
   }

   type Severity string
   const (
       SevInfo     Severity = "info"
       SevLow      Severity = "low"
       SevMedium   Severity = "medium"
       SevHigh     Severity = "high"
       SevCritical Severity = "critical"
   )

   type Finding struct {
       ID          string   `json:"id"`
       TargetID    string   `json:"targetId"`
       Category    string   `json:"category"`
       Severity    Severity `json:"severity"`
       Tools       []string `json:"tools"`
       Location    Location `json:"location,omitempty"`
       Description string   `json:"description"`
   }

   type Location struct {
       File      string `json:"file"`
       LineStart int    `json:"lineStart"`
       LineEnd   int    `json:"lineEnd"`
   }

   type VerifiedExploit struct {
       ID                 string    `json:"id"`
       FindingID          string    `json:"findingId"`
       ForgePath          string    `json:"forgePath"`
       DrainAmountUsd     float64   `json:"drainAmountUsd"`
       BlockNumber        int64     `json:"blockNumber"`
       DifferentialPassed bool      `json:"differentialPassed"`
       VerifiedAt         time.Time `json:"verifiedAt"`
   }

   type Disclosure struct {
       ID          string    `json:"id"`
       ExploitID   string    `json:"exploitId"`
       Lane        string    `json:"lane"`
       X402URL     string    `json:"x402Url,omitempty"`
       TxHash      string    `json:"txHash,omitempty"`
       PublishedAt time.Time `json:"publishedAt"`
   }
   ```

2. Place identical content in both Go services. (Optional: extract to a third Go module shared by both; for hackathon simplicity, duplicate.)

**Verify:**
```bash
cd services/scout-go && go build ./pkg/messages/...
cd ../orchestrator-go && go build ./pkg/messages/...
```
Expected: clean build in both.

---

### Task 0.8 — Generate Rust types from schema

**Goal:** Serde-compatible Rust structs that round-trip with the Go types.

**Files:** `services/auditor-rs/src/messages.rs`.

**Steps:**

1. Add the module declaration to `src/main.rs`: `mod messages;`
2. Create `src/messages.rs`:

   ```rust
   use serde::{Deserialize, Serialize};
   use chrono::{DateTime, Utc};

   #[derive(Debug, Serialize, Deserialize, Clone)]
   #[serde(rename_all = "lowercase")]
   pub enum TargetKind { Onchain, Github, Immunefi }

   #[derive(Debug, Serialize, Deserialize, Clone)]
   pub struct Target {
       pub id: String,
       pub kind: TargetKind,
       #[serde(rename = "chainId", skip_serializing_if = "Option::is_none")]
       pub chain_id: Option<u64>,
       #[serde(skip_serializing_if = "Option::is_none")]
       pub address: Option<String>,
       #[serde(skip_serializing_if = "Option::is_none")]
       pub repo: Option<String>,
       #[serde(rename = "commitSha", skip_serializing_if = "Option::is_none")]
       pub commit_sha: Option<String>,
       #[serde(rename = "sourceUrl", skip_serializing_if = "Option::is_none")]
       pub source_url: Option<String>,
       #[serde(rename = "discoveredAt")]
       pub discovered_at: DateTime<Utc>,
       pub priority: f64,
       #[serde(rename = "tvlUsd", skip_serializing_if = "Option::is_none")]
       pub tvl_usd: Option<f64>,
   }

   #[derive(Debug, Serialize, Deserialize, Clone)]
   #[serde(rename_all = "lowercase")]
   pub enum Severity { Info, Low, Medium, High, Critical }

   #[derive(Debug, Serialize, Deserialize, Clone, Default)]
   pub struct Location {
       pub file: String,
       #[serde(rename = "lineStart")]
       pub line_start: u32,
       #[serde(rename = "lineEnd")]
       pub line_end: u32,
   }

   #[derive(Debug, Serialize, Deserialize, Clone)]
   pub struct Finding {
       pub id: String,
       #[serde(rename = "targetId")]
       pub target_id: String,
       pub category: String,
       pub severity: Severity,
       pub tools: Vec<String>,
       #[serde(default)]
       pub location: Location,
       #[serde(default)]
       pub description: String,
   }
   ```

3. Add `chrono = { version = "0.4", features = ["serde"] }` to `Cargo.toml`.

**Verify:**
```bash
cd services/auditor-rs && cargo build
```
Expected: clean build.

**Notes:** Skip `VerifiedExploit` and `Disclosure` in Rust — only the Go orchestrator emits those.

---

### Task 0.9 — Cross-language schema round-trip test

**Goal:** Prove a `Target` JSON written by Go parses in Rust and vice versa.

**Files:** `infra/scripts/schema-roundtrip.sh`, two tiny test programs.

**Steps:**

1. Write a Go script `services/scout-go/cmd/dump-target/main.go` that emits one canonical `Target` JSON to stdout.
2. Write a Rust integration test in `services/auditor-rs/tests/roundtrip.rs` that reads stdin and parses to `Target`.
3. Create `infra/scripts/schema-roundtrip.sh`:
   ```bash
   #!/usr/bin/env bash
   set -euo pipefail
   cd "$(dirname "$0")/../.."
   go run ./services/scout-go/cmd/dump-target | \
     cargo run --manifest-path services/auditor-rs/Cargo.toml --bin roundtrip-check
   ```

**Verify:**
```bash
chmod +x infra/scripts/schema-roundtrip.sh
./infra/scripts/schema-roundtrip.sh
```
Expected: prints `roundtrip OK` (or similar from your Rust program). Any deserialization error here means Go and Rust schemas drifted — fix before continuing.

---

### Task 0.10 — Top-level Makefile

**Goal:** One command builds the whole repo.

**Files:** `$REPO/Makefile`.

**Steps:**

1. Create `Makefile`:
   ```makefile
   .PHONY: build test clean scout auditor orchestrator contracts roundtrip

   build: scout auditor orchestrator contracts

   scout:
   	cd services/scout-go && go build ./...

   orchestrator:
   	cd services/orchestrator-go && go build ./...

   auditor:
   	cd services/auditor-rs && cargo build --release

   contracts:
   	cd contracts && forge build

   roundtrip:
   	bash infra/scripts/schema-roundtrip.sh

   test:
   	cd services/scout-go && go test ./...
   	cd services/orchestrator-go && go test ./...
   	cd services/auditor-rs && cargo test
   	cd contracts && forge test

   clean:
   	cd services/auditor-rs && cargo clean
   	cd contracts && forge clean
   	rm -rf services/*/bin
   ```

**Verify:**
```bash
make build && make roundtrip
```
Expected: all four sub-builds succeed; round-trip prints OK.

---

**🚦 Phase 0 Gate.** Do not proceed unless: (a) `make build` is green, (b) `make roundtrip` is green, (c) `.env` exists with all values filled. Report status to operator.

---

## Phase 1 — Scout Agent: Mempool Listener

**Phase goal:** A running Go service that observes Sepolia and emits `Target` records for new contract deployments and large transfers.

### Task 1.1 — Add ethclient dependency and config loader

**Goal:** Scout reads `SEPOLIA_RPC_URL` from env and connects.

**Files:** `services/scout-go/internal/config/config.go`, `services/scout-go/cmd/main.go`.

**Steps:**

1. `cd services/scout-go && go get github.com/ethereum/go-ethereum@latest`
2. `go get github.com/joho/godotenv@latest`
3. Create `internal/config/config.go`:
   ```go
   package config

   import (
       "fmt"
       "os"
       "github.com/joho/godotenv"
   )

   type Config struct {
       SepoliaRPC   string
       GitHubToken  string
       MinTVLUsd    float64
   }

   func Load() (*Config, error) {
       _ = godotenv.Load("../../.env")
       c := &Config{
           SepoliaRPC:  os.Getenv("SEPOLIA_RPC_URL"),
           GitHubToken: os.Getenv("GITHUB_TOKEN"),
       }
       if c.SepoliaRPC == "" {
           return nil, fmt.Errorf("SEPOLIA_RPC_URL not set")
       }
       return c, nil
   }
   ```

**Verify:**
```bash
cd services/scout-go && go build ./...
```
Expected: clean build.

---

### Task 1.2 — Implement pending-tx subscription

**Goal:** Subscribe to Sepolia pending tx hashes; resolve each to a full transaction; filter for contract creations.

**Files:** `services/scout-go/internal/scout/mempool.go`.

**Steps:**

1. Create `internal/scout/mempool.go`:
   ```go
   package scout

   import (
       "context"
       "log/slog"
       "time"

       "github.com/ethereum/go-ethereum/core/types"
       "github.com/ethereum/go-ethereum/ethclient"
       "github.com/ethereum/go-ethereum/rpc"
       "github.com/google/uuid"

       "github.com/<org>/swarm/scout/pkg/messages"
   )

   type MempoolWatcher struct {
       rpcURL string
       out    chan<- messages.Target
   }

   func NewMempoolWatcher(rpcURL string, out chan<- messages.Target) *MempoolWatcher {
       return &MempoolWatcher{rpcURL: rpcURL, out: out}
   }

   func (w *MempoolWatcher) Run(ctx context.Context) error {
       rpcClient, err := rpc.DialContext(ctx, w.rpcURL)
       if err != nil { return err }
       client := ethclient.NewClient(rpcClient)

       ch := make(chan common.Hash, 256)
       sub, err := rpcClient.EthSubscribe(ctx, ch, "newPendingTransactions")
       if err != nil { return err }
       defer sub.Unsubscribe()

       for {
           select {
           case <-ctx.Done(): return nil
           case err := <-sub.Err(): return err
           case h := <-ch:
               tx, _, err := client.TransactionByHash(ctx, h)
               if err != nil { continue }
               if tx.To() == nil { // contract creation
                   t := messages.Target{
                       ID:           uuid.NewString(),
                       Kind:         messages.TargetOnchain,
                       ChainID:      11155111, // Sepolia
                       DiscoveredAt: time.Now().UTC(),
                       Priority:     50, // default; refined later
                   }
                   slog.Info("contract creation observed", "tx", h.Hex())
                   w.out <- t
               }
           }
       }
   }
   ```

   Replace `<org>` with the value from Task 0.2.

2. Update `cmd/main.go` to wire it:
   ```go
   package main

   import (
       "context"
       "log/slog"
       "os/signal"
       "syscall"

       "github.com/<org>/swarm/scout/internal/config"
       "github.com/<org>/swarm/scout/internal/scout"
       "github.com/<org>/swarm/scout/pkg/messages"
   )

   func main() {
       ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
       defer stop()

       cfg, err := config.Load()
       if err != nil { slog.Error("config", "err", err); return }

       out := make(chan messages.Target, 64)
       go func() {
           for t := range out {
               slog.Info("target", "id", t.ID, "kind", t.Kind, "chain", t.ChainID)
           }
       }()

       w := scout.NewMempoolWatcher(cfg.SepoliaRPC, out)
       if err := w.Run(ctx); err != nil { slog.Error("watcher", "err", err) }
   }
   ```

**Verify:**
```bash
cd services/scout-go && go run ./cmd &
sleep 90
kill %1
```
Expected: at least one `target` log line appears within 90 seconds. Sepolia is busy enough that this should hit. If zero appear, check the RPC supports `newPendingTransactions` (some do not — Alchemy and Infura do; some public endpoints do not).

**Notes:** If your RPC provider doesn't support pending-tx subscriptions, fall back to polling `eth_getBlockByNumber("latest")` and inspecting receipts. Document the fallback in `docs/`.

---

### Task 1.3 — Add value-transfer threshold filter

**Goal:** Also emit Targets for large value transfers (potential rug pulls or whale moves) above `MIN_DRAIN_THRESHOLD_USD`.

**Files:** Modify `services/scout-go/internal/scout/mempool.go`.

**Steps:**

1. Inside the `case h := <-ch:` block, after the contract-creation check, add:
   ```go
   if tx.Value() != nil && tx.Value().Sign() > 0 {
       // crude USD estimate: assume ETH price hardcoded for hackathon; plug oracle later
       ethUsd := 3000.0
       valueEth := new(big.Float).Quo(
           new(big.Float).SetInt(tx.Value()),
           new(big.Float).SetInt(big.NewInt(1e18)),
       )
       valFloat, _ := valueEth.Float64()
       if valFloat * ethUsd >= 50000 { // tunable
           t := messages.Target{
               ID:           uuid.NewString(),
               Kind:         messages.TargetOnchain,
               ChainID:      11155111,
               Address:      tx.To().Hex(),
               DiscoveredAt: time.Now().UTC(),
               Priority:     60 + min(valFloat*ethUsd/1e6, 40),
               TVLUsd:       valFloat * ethUsd,
           }
           w.out <- t
       }
   }
   ```

**Verify:**
```bash
cd services/scout-go && go test ./internal/scout/...
```
Expected: any unit tests pass. Add a unit test that constructs a `*types.Transaction` with a known value and asserts a target is emitted.

**Notes:** Hardcoded ETH price is fine for the hackathon; mark as TODO. For mainnet, plug Chainlink or Coingecko.

---

### Task 1.4 — Add GitHub commit watcher

**Goal:** Watch a configurable list of DeFi repos for new commits and emit Targets.

**Files:** `services/scout-go/internal/scout/github.go`, `services/scout-go/configs/repos.yaml`.

**Steps:**

1. Create `configs/repos.yaml`:
   ```yaml
   repos:
     - aave/aave-v3-core
     - Uniswap/v4-core
     - compound-finance/compound-protocol
   poll_interval_seconds: 60
   ```

2. Create `internal/scout/github.go` with a polling loop using `github.com/google/go-github/v62/github`. On each new commit since last poll, emit a `Target{Kind: TargetGitHub, Repo: ..., CommitSha: ...}`. Persist last-seen sha to disk in `.scout-state.json`.

**Verify:**
```bash
cd services/scout-go && GITHUB_TOKEN=<token> go run ./cmd 2>&1 | grep -m1 "kind=github"
```
Expected: a `kind=github` target is logged within ~2 minutes (depends on whether any of the watched repos have recent commits).

---

### Task 1.5 — Add priority scoring

**Goal:** A pure function `Score(t Target) float64` that produces 0–100 based on TVL, novelty, kind.

**Files:** `services/scout-go/internal/scout/priority.go`, plus unit tests.

**Steps:**

1. Implement weighted formula: `score = 0.5*tvl_score + 0.3*novelty_score + 0.2*kind_score`.
2. `tvl_score` is `min(log10(tvl_usd) * 10, 100)`.
3. `novelty_score` is 100 if first seen <1h ago, 50 if <24h, 25 otherwise.
4. `kind_score` is 80 for `onchain`, 60 for `github`, 70 for `immunefi`.
5. Apply `Score` before emitting on the channel; overwrite the default `Priority`.

**Verify:**
```bash
cd services/scout-go && go test ./internal/scout/ -run TestScore -v
```
Expected: tests covering each kind and TVL boundary all pass.

---

**🚦 Phase 1 Gate.** Scout runs for 5 minutes against Sepolia and emits at least one onchain target and (if any commits landed) one github target, with priority scores in 0–100.

---

## Phase 2 — AXL Mesh Integration

**Phase goal:** Scout publishes targets to the AXL mesh; a debug subscriber sees them on a different process.

> ⚠️ **Agent: stop here and read the Gensyn AXL docs before writing code.** The function names, topic semantics, and key management below are illustrative. Match them to the real SDK. If you cannot find the SDK, ask the operator.

### Task 2.1 — Vendor the AXL SDK

**Goal:** AXL Go bindings importable from scout and orchestrator.

**Files:** `services/scout-go/go.mod`, `services/orchestrator-go/go.mod`.

**Steps:**

1. Per AXL docs, `go get` the official Go SDK package.
2. Create a thin wrapper in `services/scout-go/pkg/axl/axl.go` that exposes only the four methods we need: `NewNode(bootstrap []string)`, `Publish(topic string, payload []byte)`, `Subscribe(topic string) (<-chan []byte, error)`, `Close()`. The wrapper isolates us from SDK churn.

**Verify:** `go build ./...` is clean. The wrapper compiles even if its body is `return nil` stubs.

---

### Task 2.2 — Implement the wrapper for real

**Goal:** Replace stubs with real calls.

**Files:** `services/scout-go/pkg/axl/axl.go`.

**Steps:**

1. Wire `NewNode` to the SDK constructor, parsing `AXL_BOOTSTRAP_NODES` (comma-separated) from env.
2. Implement `Publish` and `Subscribe` per SDK.
3. Add a `Health() error` method that returns nil if the local node has at least one peer.

**Verify:** Write a unit test that starts two `Node` instances on different ports, publishes a message from one, and asserts the other receives it. Test passes.

---

### Task 2.3 — Wire scout to publish targets

**Goal:** The channel-receive goroutine in `cmd/main.go` publishes each target to AXL topic `targets/discovered` instead of (or in addition to) logging.

**Files:** `services/scout-go/cmd/main.go`.

**Steps:**

1. Construct an `axl.Node` from config.
2. Replace the log-only consumer with a publisher that JSON-marshals the target and publishes.
3. Keep the log line for observability.

**Verify:** Run scout in one terminal, run `axl-tail` (next task) in another, observe targets arriving.

---

### Task 2.4 — Build the `axl-tail` debug binary

**Goal:** A standalone subscriber that prints every message on every interesting topic. This is your most-used debugging tool for the rest of the project.

**Files:** `services/scout-go/cmd/axl-tail/main.go`.

**Steps:**

1. Subscribe to all four topics (`targets/discovered`, `analysis/findings`, `exploit/verified`, `disclosure/published`).
2. For each message, print `[topic] <pretty-json>` with a timestamp.
3. Add `--topic` CLI flag to filter.

**Verify:**
```bash
go run ./services/scout-go/cmd/axl-tail &
go run ./services/scout-go/cmd &
sleep 30
```
Expected: `axl-tail` prints at least one `[targets/discovered]` message.

---

### Task 2.5 — Deduplication on the mesh

**Goal:** Two scouts seeing the same contract creation should result in one logical target, not two.

**Files:** `services/scout-go/internal/scout/dedupe.go`.

**Steps:**

1. Hash a stable subset of fields (`kind + chainId + address + repo + commitSha`) into a content ID.
2. Maintain an LRU cache of recently published content IDs (size 10000, TTL 1 hour).
3. Before publishing, check the cache; if hit, drop.

**Verify:** Unit test asserts that publishing the same target twice results in one `Publish` call to a mock AXL node.

---

**🚦 Phase 2 Gate.** Run two scout instances simultaneously against the same Sepolia RPC. `axl-tail` shows each unique target exactly once.

---

## Phase 3 — Auditor Agent: Static Analysis

**Phase goal:** Rust agent listens on `targets/discovered`, fetches contract source, runs Aderyn + Slither, emits dual-validated findings.

### Task 3.1 — AXL subscriber in Rust

**Goal:** Auditor receives targets from the mesh.

**Files:** `services/auditor-rs/src/axl.rs`, update `src/main.rs`.

**Steps:**

1. If the Gensyn AXL SDK has Rust bindings, vendor them. If not, create a tiny gRPC or HTTP bridge in the Go scout that exposes a Server-Sent-Events stream of `targets/discovered` and have Rust consume that. Document the choice in `docs/axl-bridge.md`.
2. Implement an async `subscribe_targets() -> impl Stream<Item = Target>` in `src/axl.rs`.
3. In `main.rs`, replace the stub with a loop that consumes the stream and logs each target.

**Verify:**
```bash
cargo run --manifest-path services/auditor-rs/Cargo.toml &
go run ./services/scout-go/cmd &
sleep 60
```
Expected: auditor logs at least one received target.

---

### Task 3.2 — Source fetcher

**Goal:** Given a `Target` with an address + chainId, fetch verified Solidity source from Etherscan.

**Files:** `services/auditor-rs/src/fetcher.rs`.

**Steps:**

1. Add `reqwest = { version = "0.12", features = ["json"] }` to `Cargo.toml`.
2. Implement `fetch_source(chain_id: u64, address: &str) -> Result<SourceBundle>` where `SourceBundle` contains `{ files: HashMap<String, String>, abi: String, compiler_version: String }`.
3. Use the appropriate Etherscan API endpoint per chain. For unverified contracts, fetch bytecode only and mark `files` empty.
4. Write source to a temp dir under `/tmp/auditor/{target_id}/`.

**Verify:** Unit test against a known verified mainnet contract (e.g. USDC proxy `0xA0b86991...`). Assert files are populated.

**Notes:** Etherscan rate limits aggressively. Add a 5/sec semaphore.

---

### Task 3.3 — Aderyn runner

**Goal:** Invoke Aderyn on the fetched source, parse JSON output into `Finding`s.

**Files:** `services/auditor-rs/src/analyzers/aderyn.rs`.

**Steps:**

1. Implement `run_aderyn(source_dir: &Path) -> Result<Vec<Finding>>` that shells out to `aderyn <dir> --output json` and parses the result.
2. Map Aderyn's severity strings to our `Severity` enum.
3. Tag every finding with `tools: vec!["aderyn"]`.
4. Set `target_id` from the calling context.

**Verify:** Run against `test-fixtures/dvd/contracts/unstoppable/UnstoppableVault.sol` (after Task 3.6 clones DVD). Assert at least one finding is returned.

---

### Task 3.4 — Slither runner

**Goal:** Same as 3.3 but for Slither.

**Files:** `services/auditor-rs/src/analyzers/slither.rs`.

**Steps:**

1. Implement `run_slither(source_dir: &Path) -> Result<Vec<Finding>>`. Slither's JSON output is at `slither --json -`.
2. Map Slither's `impact` field to `Severity`.
3. Tag with `tools: vec!["slither"]`.

**Verify:** Same DVD fixture; at least one finding returned.

**Notes:** Slither needs `solc` matching the contract's pragma. Use `solc-select` and pick the version from the SourceBundle's `compiler_version`.

---

### Task 3.5 — Dual-source filter

**Goal:** Merge Aderyn + Slither outputs; emit a finding only if (a) both tools agree on the category for the same location, OR (b) a single tool reports `high` or `critical`.

**Files:** `services/auditor-rs/src/analyzers/merge.rs`.

**Steps:**

1. Define a category-equivalence map (e.g. Aderyn's `reentrancy-state-change` ≡ Slither's `reentrancy-eth`). Put it in `analyzers/equivalences.rs`.
2. Implement `merge(aderyn: Vec<Finding>, slither: Vec<Finding>) -> Vec<Finding>`:
   - For each Aderyn finding, look for a Slither finding with equivalent category and overlapping line range. If found, emit a single merged finding with `tools: ["aderyn","slither"]` and severity = max.
   - Emit single-tool findings only if `severity ∈ {High, Critical}`.
3. Drop everything else.

**Verify:** Unit test with hand-crafted Aderyn and Slither outputs covering: agreement case (emit), single-tool low (drop), single-tool high (emit), disagreement (drop the lows).

---

### Task 3.6 — DVD fixture and golden tests

**Goal:** A reproducible test corpus.

**Files:** `test-fixtures/dvd/`, `services/auditor-rs/tests/dvd_golden.rs`.

**Steps:**

1. `cd test-fixtures && git clone https://github.com/theredguild/damn-vulnerable-defi.git dvd && cd dvd && git checkout v4.0.0` (or whichever stable tag exists).
2. Write a golden test that runs the full analyzer pipeline against three DVD challenges (Unstoppable, Naive Receiver, Truster) and asserts the expected vulnerability category appears in the merged output. Commit the expected output as JSON files in `tests/golden/`.

**Verify:** `cargo test --test dvd_golden` passes.

**Notes:** When DVD updates, golden files need refresh — that's expected.

---

### Task 3.7 — Publish findings to AXL

**Goal:** Auditor publishes merged findings to `analysis/findings`.

**Files:** Update `services/auditor-rs/src/main.rs`.

**Steps:**

1. After analysis completes, JSON-serialize each finding and publish through the AXL bridge.
2. Log the number of findings published per target.

**Verify:** End-to-end run: scout publishes a target → auditor analyzes a known DVD contract (manually triggered with a mock target pointing at local source) → `axl-tail` shows findings on `analysis/findings`.

---

**🚦 Phase 3 Gate.** Pipeline runs end-to-end on a DVD fixture: target → source fetch → Aderyn + Slither → merged finding on AXL. Findings have `tools` populated correctly.

---

## Phase 4 — Orchestrator: LLM Exploit Generation

**Phase goal:** Orchestrator subscribes to findings, prompts an LLM to write a Foundry exploit test, and stores the test on disk.

### Task 4.1 — Orchestrator skeleton

**Goal:** Bare orchestrator that subscribes to `analysis/findings` and logs.

**Files:** `services/orchestrator-go/cmd/main.go`, `services/orchestrator-go/internal/orchestrator/orchestrator.go`.

**Steps:**

1. Mirror the scout's structure: `config`, `axl` wrapper (copy from scout for now), `cmd/main.go`.
2. Subscribe to `analysis/findings`. For each, log `finding id={id} category={category} severity={severity}`.

**Verify:** Run orchestrator while a finding flows through; log line appears.

---

### Task 4.2 — LLM client

**Goal:** A Go client that calls Anthropic (or OpenAI) and returns the response text.

**Files:** `services/orchestrator-go/internal/llm/client.go`.

**Steps:**

1. Implement `type Client interface { Complete(ctx, system, user string) (string, error) }`.
2. Provide an Anthropic implementation using the official Go SDK. Default model: `claude-sonnet-4-7-20250901` or whatever is current; read from env `LLM_MODEL`.
3. Add token-budget tracking: log estimated input/output tokens per call.
4. Wrap with a retry policy (3 attempts, exponential backoff) for transient errors.

**Verify:** Unit test with a mock HTTP server returning a canned response; assert `Complete` returns the expected text.

---

### Task 4.3 — Prompt template

**Goal:** A versioned prompt that consistently produces compilable Foundry tests.

**Files:** `prompts/exploit_v1.md`.

**Steps:**

1. Author a prompt with these sections:
   - **Role:** "You are a security researcher writing Foundry PoC tests."
   - **Context:** Insert finding JSON, source code, target address.
   - **Instructions:**
     1. Output only a single Solidity file, no prose.
     2. The file imports `forge-std/Test.sol`.
     3. The contract name is `ExploitTest`.
     4. Include a `setUp()` that forks at the current block.
     5. Include a `test_Exploit()` that demonstrates the flaw.
     6. Use `vm.assertGt(attackerBalanceAfter, attackerBalanceBefore + minDrain, ...)`.
   - **Output format:** strict — between `<solidity>` and `</solidity>` tags, nothing else.
2. Add 2–3 few-shot examples drawn from solved DVD challenges.

**Verify:** Pipe the prompt through the LLM client manually with one DVD finding. The output must compile under `forge build` after stripping the tags.

---

### Task 4.4 — Exploit generation pipeline

**Goal:** On each finding from AXL, build the prompt, call LLM, save the file.

**Files:** `services/orchestrator-go/internal/exploit/generate.go`.

**Steps:**

1. Implement `Generate(ctx, finding Finding, source SourceBundle) (path string, err error)`:
   - Render `prompts/exploit_v1.md` using `text/template` with finding + source.
   - Call `llm.Client.Complete`.
   - Extract content between `<solidity>` and `</solidity>`.
   - Write to `/tmp/orchestrator/exploits/{finding_id}.t.sol`.
   - Return the path.
2. Wire into the orchestrator's finding handler.

**Verify:** Run end-to-end on a DVD finding. The `.t.sol` file exists and compiles when copied into a Foundry project.

---

### Task 4.5 — Source-bundle re-fetch

**Goal:** Orchestrator needs the same source the auditor used. It can either re-fetch or accept the bundle path via the AXL message.

**Files:** Decide and document.

**Steps:**

1. **Recommendation:** auditor uploads its `SourceBundle` to a shared local path `/var/swarm/sources/{target_id}/` and includes the path in its finding message. Orchestrator reads from there.
2. Update `Finding` schema (in `proto/messages.json`) to add an optional `sourcePath` field. Regenerate Go and Rust types. Re-run `make roundtrip`.

**Verify:** End-to-end run shows orchestrator successfully loading source from `sourcePath` without re-hitting Etherscan.

---

### Task 4.6 — Retry loop with feedback

**Goal:** If the generated exploit fails to compile, feed the compiler error back to the LLM up to `LLM_MAX_RETRIES` times.

**Files:** `services/orchestrator-go/internal/exploit/generate.go`.

**Steps:**

1. After saving the `.t.sol`, run `forge build --root <tmp project>` against it. If it fails, capture stderr.
2. Re-prompt with: original prompt + previous output + "the following compiler error occurred: ...". Repeat up to N times.
3. If still failing after N retries, drop the finding and log `exploit-gen-failed`.

**Verify:** Inject a deliberately broken first response (mock LLM); assert the second attempt with corrected error context succeeds.

---

**🚦 Phase 4 Gate.** End-to-end: scout → auditor → orchestrator produces a `.t.sol` that compiles. The retry loop demonstrably recovers from at least one synthetic compile error.

---

## Phase 5 — Foundry Triple-Verification

**Phase goal:** Prove an exploit is real with three checks: it works on the live fork, it depends on the actual bug (differential), and it drains above threshold.

### Task 5.1 — Forge harness directory

**Goal:** A pre-built Foundry project where generated exploits drop in and run.

**Files:** `infra/forge-harness/foundry.toml`, `infra/forge-harness/lib/forge-std/`.

**Steps:**

1. `cd infra && forge init forge-harness --no-git`
2. Configure RPCs in `foundry.toml`.
3. Add `remappings.txt`: `forge-std/=lib/forge-std/src/`.

**Verify:** `cd infra/forge-harness && forge build` is clean.

---

### Task 5.2 — Live-fork test runner

**Goal:** Given a `.t.sol` and a fork URL, run `forge test --fork-url --fork-block-number latest`.

**Files:** `services/orchestrator-go/internal/verify/forge.go`.

**Steps:**

1. Implement `RunForkTest(exploitPath string, forkURL string, forkBlock int64) (TestResult, error)`.
2. Copy the exploit into `infra/forge-harness/test/`.
3. Shell out to `forge test --match-contract ExploitTest --json`. Parse JSON output.
4. Return `{ Passed bool, GasUsed uint64, Logs []string }`.

**Verify:** Unit test runs a known-good exploit (one of DVD's solved tests) and asserts `Passed = true`.

---

### Task 5.3 — Drain-amount extractor

**Goal:** Parse the test logs to extract the actual USD drain.

**Files:** `services/orchestrator-go/internal/verify/drain.go`.

**Steps:**

1. The prompt (Task 4.3) instructs the LLM to emit `console.log("DRAIN_USD", uint(...))` at the end of the test.
2. Implement `ExtractDrainUsd(logs []string) (float64, error)` that finds and parses that line.
3. If absent or zero, return error.

**Verify:** Unit test with synthetic logs.

---

### Task 5.4 — Differential check

**Goal:** Run the exploit against a patched version of the contract; assert it now fails.

**Files:** `services/orchestrator-go/internal/verify/differential.go`.

**Steps:**

1. Take the original source. Use the LLM (separate prompt, `prompts/patch_v1.md`) to produce a patched version where the suspected vulnerable function is hardened.
2. Deploy patched version to a fresh anvil fork using `forge script`.
3. Re-run the exploit pointed at the new address. Assert it now fails (revert, no drain, or `vm.expectRevert`).
4. Return `DifferentialPassed = true` only if the patched version successfully blocks the exploit.

**Verify:** On a DVD fixture, the original passes the live-fork test AND the patched version fails it.

**Notes:** Differential is the most expensive check (extra LLM call + extra anvil run). Make it optional via env flag `ENABLE_DIFFERENTIAL=true` for hackathon demo.

---

### Task 5.5 — Triple-verify orchestration

**Goal:** Wire 5.2 + 5.3 + 5.4 into a single function.

**Files:** `services/orchestrator-go/internal/verify/verify.go`.

**Steps:**

1. Implement `TripleVerify(exploitPath, source, finding) (VerifiedExploit, error)`:
   - Run live-fork test. If fail → error.
   - Extract drain. If `< MIN_DRAIN_THRESHOLD_USD` → error.
   - Run differential. If `ENABLE_DIFFERENTIAL=true` and fails → error.
   - Construct `VerifiedExploit{ ... DifferentialPassed: true|false }`.
   - Publish to `exploit/verified` topic.

**Verify:** End-to-end on DVD: at least one challenge produces a `VerifiedExploit` on the AXL mesh visible to `axl-tail`.

---

**🚦 Phase 5 Gate.** A full pipeline run on `damn-vulnerable-defi` produces at least one verified exploit with `drainAmountUsd > 0` and `differentialPassed = true`.

---

## Phase 6 — Bounty Reports & x402 Gating

**Phase goal:** Generate a public teaser + paywalled full report; the full report is gated behind KeeperHub x402.

> ⚠️ KeeperHub specifics may differ from below. Match the real SDK.

### Task 6.1 — Report generator

**Goal:** Produce two files per verified exploit: `teaser.md`, `full.md`.

**Files:** `services/orchestrator-go/internal/report/generate.go`, `prompts/report_v1.md`.

**Steps:**

1. LLM prompt builds a structured report with: Summary, Severity, Affected contract, Vulnerability class, PoC (full report only), Recommended fix, Disclosure timeline, Contact.
2. Teaser includes everything except PoC.
3. Save under `/tmp/orchestrator/reports/{exploit_id}/{teaser,full}.md`.

**Verify:** Files exist with required sections; teaser does NOT contain the words "function exploit" or any Solidity code.

---

### Task 6.2 — x402 paywall integration

**Goal:** Full report is served behind a 402 Payment Required gate via KeeperHub.

**Files:** `services/orchestrator-go/internal/x402/gate.go`.

**Steps:**

1. Per KeeperHub docs, register a payment endpoint with stablecoin amount (default: 100 USDC, configurable per finding via `severity * factor`).
2. Generate a public URL where the teaser is served at `/preview/{id}` (free) and the full report at `/report/{id}` (402 gated).
3. After successful payment callback, mark the report unlocked for that payer (signed token in URL).

**Verify:**
```bash
curl -i $X402_URL/report/<id>
```
Expected: HTTP 402 with payment instructions in body. After paying via test wallet, GET returns the full report.

**Notes:** If KeeperHub doesn't expose this exact pattern, document the actual flow and adapt. The contract (gated full report, free teaser) is what matters.

---

### Task 6.3 — Disclosure publication

**Goal:** Once the report is live, emit a `Disclosure` on AXL and (optionally) submit to Immunefi.

**Files:** `services/orchestrator-go/internal/disclosure/publish.go`.

**Steps:**

1. After x402 endpoint is live, construct a `Disclosure{ lane: "bounty", x402Url: ... }`.
2. Publish on `disclosure/published`.
3. If the target's protocol has an Immunefi program (lookup via Immunefi API), POST a submission. Mark `immunefiSubmitted: true`.

**Verify:** End-to-end: a verified exploit produces a disclosure visible on AXL with a working x402 URL.

---

### Task 6.4 — Disclosure window enforcement

**Goal:** Even after publishing the teaser, the full PoC is time-locked: it cannot be revealed publicly for `DISCLOSURE_WINDOW_DAYS` (default 90).

**Files:** `services/orchestrator-go/internal/disclosure/window.go`.

**Steps:**

1. Store `revealAt = publishedAt + window` in local state.
2. The protocol team's payment unlocks the full report immediately for them.
3. A separate scheduled job (cron-style goroutine) flips a public flag at `revealAt`, allowing anyone to read.

**Verify:** Set `DISCLOSURE_WINDOW_DAYS=0` in test env, publish, immediately fetch publicly — should succeed. Set to 365, publish, fetch publicly — should 402 even without payment.

---

**🚦 Phase 6 Gate.** A live, paywalled URL exists for at least one verified exploit. Teaser is publicly readable. Full report unlocks after a (test) payment.

---

## Phase 7 — 0G iNFT: Sovereignty & Memory

**Phase goal:** Deploy ERC-7857 iNFT on 0G; orchestrator pushes state updates on every disclosure.

> ⚠️ 0G ERC-7857 details: confirm the actual interface from 0G docs. The skeleton below is structurally correct but field names may differ.

### Task 7.1 — Write the iNFT contract

**Goal:** A minimal ERC-7857 implementation tracking the swarm's stats and authorized protocols.

**Files:** `contracts/src/SwarmINFT.sol`, `contracts/test/SwarmINFT.t.sol`.

**Steps:**

1. Implement based on the 0G ERC-7857 reference. Required state:
   - `mapping(uint256 => SwarmState) state` keyed by tokenId.
   - `SwarmState { uint256 disclosuresCount; uint256 cumulativeBountyUsd; bytes32 memoryPointer; bool paused; }`.
   - `mapping(address => bool) authorizedProtocols`.
2. Functions:
   - `recordDisclosure(uint256 tokenId, uint256 bountyUsd, bytes32 memoryDelta)` — only orchestrator key.
   - `setAuthorizedProtocol(address protocol, bool ok)` — only multisig.
   - `setPaused(bool)` — only multisig.
3. Add events for every state-changing function.

**Verify:** `forge test --match-contract SwarmINFTTest` passes a unit test covering each function and access control.

---

### Task 7.2 — Deploy script

**Goal:** Deterministic deployment to 0G testnet.

**Files:** `contracts/script/Deploy.s.sol`.

**Steps:**

1. Standard Foundry script reading `OG_PRIVATE_KEY` from env.
2. Deploys `SwarmINFT`, mints tokenId 1 to a configured operator address.
3. Writes deployment address to `deployments/og_testnet.json`.

**Verify:**
```bash
cd contracts && forge script script/Deploy.s.sol --rpc-url $OG_RPC_URL --broadcast
```
Expected: success; `deployments/og_testnet.json` populated; address verifiable via 0G explorer.

---

### Task 7.3 — Orchestrator iNFT client

**Goal:** Go client that calls `recordDisclosure` after each successful disclosure.

**Files:** `services/orchestrator-go/internal/inft/client.go`.

**Steps:**

1. Generate Go bindings: `abigen --abi contracts/out/SwarmINFT.sol/SwarmINFT.abi.json --pkg inft --out services/orchestrator-go/internal/inft/bindings.go`.
2. Implement `RecordDisclosure(ctx, exploitId, bountyUsd, memoryDelta)` that signs and sends via `bind.NewKeyedTransactor`.
3. Handle nonce management with a sync.Mutex around the transactor.
4. On success, log tx hash; on failure, retry once then alert.

**Verify:** Run a synthetic disclosure; check 0G explorer for the on-chain `DisclosureRecorded` event.

---

### Task 7.4 — Memory pointer

**Goal:** The iNFT's `memoryPointer` field is a content hash of the swarm's accumulated knowledge — past findings, false-positive blacklist.

**Files:** `services/orchestrator-go/internal/memory/store.go`.

**Steps:**

1. Maintain a local append-only log file `swarm-memory.jsonl` of every finding (verified or not), every false positive (manually flagged), and every disclosure.
2. On each disclosure, compute SHA-256 of the file → pass as `memoryDelta` to `recordDisclosure`.
3. Optionally upload the file to 0G storage (per their docs) and use the storage handle as the pointer; otherwise the hash alone is sufficient for hackathon.

**Verify:** Memory file grows on each event; iNFT state's `memoryPointer` matches the latest file hash.

---

### Task 7.5 — Read-side: dashboard pulls iNFT state

**Goal:** Anyone can read the swarm's stats from the iNFT.

**Files:** `services/dashboard/src/inft.ts` (if dashboard is TS) or wherever the dashboard lives.

**Steps:**

1. Use ethers.js or viem to read `state(1)`.
2. Display `disclosuresCount`, `cumulativeBountyUsd`, `paused` on the UI.

**Verify:** Dashboard shows the correct count after a few synthetic disclosures.

---

**🚦 Phase 7 Gate.** iNFT is live on 0G testnet. Disclosures from the orchestrator update its state. Dashboard reads state correctly.

---

## Phase 8 — Kill Switch, Rate Limits, Authorization

**Phase goal:** The swarm is safe to run unattended — a multisig can stop it, abuse can't spam it, and the rescue lane is hard-disabled without authorization.

### Task 8.1 — Pause-flag check

**Goal:** Every outbound action (LLM call, x402 publish, tx submit) reads `paused` from the iNFT first.

**Files:** `services/orchestrator-go/internal/safety/pause.go`.

**Steps:**

1. Implement a `PauseChecker` that polls the iNFT every 30 seconds and caches the flag.
2. Insert a check at the top of every action handler. If paused, log `paused, dropping action` and return.

**Verify:** Flip pause on chain via cast: `cast send <iNFT> "setPaused(bool)" true --private-key <multisig>`. Within 30s, observe the orchestrator drops a queued finding.

---

### Task 8.2 — Per-protocol rate limit

**Goal:** Don't submit more than N disclosures per protocol per day.

**Files:** `services/orchestrator-go/internal/safety/ratelimit.go`.

**Steps:**

1. Local store: `map[protocol]list[timestamp]`.
2. `Allow(protocol) bool` returns false if last 24h has ≥ N (configurable, default 3).

**Verify:** Unit test fires 5 events for the same protocol; only 3 pass.

---

### Task 8.3 — Authorization whitelist for rescue lane

**Goal:** The rescue path checks `authorizedProtocols[targetProtocol]` on the iNFT and refuses if false.

**Files:** `services/orchestrator-go/internal/disclosure/rescue.go`.

**Steps:**

1. For now, **stub the rescue function entirely**. Implement only the authorization check:
   ```go
   func (r *Rescue) Execute(ctx, exploit, protocol) error {
       if !r.inft.IsAuthorized(protocol) {
           return ErrUnauthorized
       }
       return ErrRescueDisabledForHackathon // intentional
   }
   ```
2. Document the rationale in `docs/rescue-lane.md`: "Disabled by default. Requires multisig vote to enable per protocol. See SECURITY.md."

**Verify:** Unit test asserts unauthorized → `ErrUnauthorized`, authorized → `ErrRescueDisabledForHackathon`. Both are explicit refusals.

---

### Task 8.4 — Audit log

**Goal:** Every action (analysis, exploit, disclosure, pause check) appends a structured line to `swarm-audit.jsonl`.

**Files:** `services/orchestrator-go/internal/safety/audit.go`.

**Steps:**

1. Implement a `Logger` with `Log(event string, fields map[string]any)`.
2. Use across all internal packages.
3. Include the iNFT memory hash from Task 7.4 every 100 lines so the log is tamper-evident in a coarse way.

**Verify:** Log file exists, is parseable as JSONL, contains entries from each phase.

---

**🚦 Phase 8 Gate.** Pause works end-to-end. Rate limit blocks bursts. Rescue lane refuses execution. Audit log is populated.

---

## Phase 9 — Demo Orchestration & Dashboard

**Phase goal:** A single command launches the whole swarm against a local target and a dashboard shows the AXL message flow live.

### Task 9.1 — Local target setup

**Goal:** A reproducible vulnerable contract on a local anvil that the swarm finds and verifies.

**Files:** `infra/scripts/demo-setup.sh`, `contracts/src/demo/VulnerableVault.sol`.

**Steps:**

1. Author a contrived vulnerable vault (e.g. classic reentrant withdraw) in `contracts/src/demo/VulnerableVault.sol`. Make it deliberately obvious — the demo prioritizes reliability, not realism.
2. `demo-setup.sh`:
   - Starts anvil on port 8545.
   - Deploys VulnerableVault.
   - Funds it with 100 ETH.
   - Funds an attacker EOA with 1 ETH for gas.
   - Prints the contract address.

**Verify:** Run the script; `cast code <addr>` returns non-empty bytecode.

---

### Task 9.2 — Demo target injection

**Goal:** A way to feed the deployed address into the scout as a synthetic onchain target without waiting for mempool detection.

**Files:** `services/scout-go/cmd/inject/main.go`.

**Steps:**

1. CLI: `inject --address 0x... --chain 31337 --priority 95 --tvl 300000`.
2. Constructs a `Target` and publishes directly to `targets/discovered`.

**Verify:** `inject ...` makes the target appear in `axl-tail` immediately.

---

### Task 9.3 — Live dashboard

**Goal:** Browser UI showing AXL traffic in real time, swarm stats from iNFT, and current pipeline stage.

**Files:** `services/dashboard/` (Vite + React or similar).

**Steps:**

1. Backend: a small Go HTTP server in `services/dashboard/server/` that subscribes to all four AXL topics and forwards to a WebSocket.
2. Frontend: simple React app showing four columns (Targets, Findings, Verified, Disclosed) with live cards. Plus an iNFT panel reading on-chain state.
3. Style minimally — even an unstyled table is fine for the hackathon. Function over form.

**Verify:** Open `http://localhost:5173`, run the demo, watch cards populate in order.

**Notes:** This is the most important deliverable for the demo. Judges need to *see* the swarm think.

---

### Task 9.4 — `make demo` target

**Goal:** One command runs everything.

**Files:** `Makefile`, `infra/scripts/run-all.sh`.

**Steps:**

1. `make demo` should:
   - Start anvil + deploy demo contract (Task 9.1).
   - Start AXL nodes.
   - Start scout, auditor, orchestrator, dashboard server (each in background, logs to `logs/`).
   - Inject the demo target.
   - Open the dashboard URL.
2. Use a process manager like `tmuxinator`, `overmind`, or a simple bash with trap-cleanup.

**Verify:** From a clean repo: `make build && make demo`. Within 60 seconds the dashboard shows the full pipeline complete for the demo target.

---

### Task 9.5 — Demo recording

**Goal:** A ≤2-minute screen recording showing the demo end-to-end. Required for submission.

**Files:** `docs/demo.mp4` (or hosted link in `README.md`).

**Steps:**

1. Run `make demo`.
2. Record screen + narration walking through each AXL topic firing.
3. Show the iNFT state increment on chain.
4. Show the x402-gated full report URL.

**Verify:** Video exists, plays end-to-end without edits hiding errors.

---

**🚦 Phase 9 Gate.** Cold-start `make demo` works on a teammate's machine without intervention.

---

## Phase 10 — Documentation & Hardening

**Phase goal:** The repo is comprehensible, the threat model is explicit, the legal posture is defensible.

### Task 10.1 — README

**Files:** `README.md`.

**Sections (required):**

1. One-paragraph elevator pitch.
2. Architecture diagram (ASCII or image).
3. Quick start: `make build && make demo`.
4. Each service's role in 2 sentences.
5. Hackathon stack callouts: Gensyn AXL, KeeperHub, 0G — with links.
6. Demo video link.

**Verify:** A reader unfamiliar with the project can run the demo from the README alone.

---

### Task 10.2 — SECURITY.md

**Files:** `SECURITY.md`.

**Sections (required):**

1. **Authorization model:** rescue lane requires explicit multisig-set authorization per protocol; default deny.
2. **Disclosure window:** PoC stays paywalled for `DISCLOSURE_WINDOW_DAYS`; no surprise public reveal.
3. **Kill switch:** any multisig signer can pause all onchain actions.
4. **Rate limits:** at most N disclosures per protocol per 24h.
5. **What this swarm will not do:** execute exploits on unauthorized contracts; submit findings without verification; reveal PoCs publicly inside the disclosure window.
6. **Reporting issues with the swarm itself:** contact info.

**Verify:** Doc exists, covers all six sections, is referenced from README.

---

### Task 10.3 — Threat model

**Files:** `docs/threat-model.md`.

**Sections:**

1. Trust assumptions (LLM correctness, AXL availability, RPC honesty).
2. Adversaries (front-runners, malicious target contracts, prompt injectors via source comments).
3. Mitigations (private mempool routing, content-ID dedup, prompt sandboxing).
4. Known limitations (false positives, LLM cost, no formal verification).

**Verify:** Doc exists. Have a teammate try to find a missing adversary class; iterate.

---

### Task 10.4 — Per-service READMEs

**Files:** `services/*/README.md`.

**Each contains:** purpose, env vars consumed, AXL topics produced/consumed, build command, test command, troubleshooting.

**Verify:** All three exist and follow the same template.

---

### Task 10.5 — Final smoke test on a clean machine

**Goal:** Eliminate hidden environment dependencies before the demo.

**Steps:**

1. Spin up a fresh VM or container.
2. Install only the prerequisites listed at the top of this doc.
3. Clone the repo, fill `.env` from secret manager.
4. `make build && make demo`.
5. Note any error; fix; iterate until clean.

**Verify:** Full demo passes on a never-before-used machine.

---

## Common Failure Modes & Recovery

| Symptom | Likely cause | Fix |
|---|---|---|
| `forge build` fails on generated exploit | LLM emitted wrong import path | Tighten prompt; add few-shot example |
| Auditor never sees targets | AXL topic mismatch | Confirm both sides use exact string `targets/discovered` |
| Aderyn JSON parse error | Wrong solc version | Use `solc-select` per-source pragma |
| `forge test` passes but drain is 0 | Test setup didn't fork the right block | Pass `--fork-block-number` explicitly |
| iNFT tx reverts | Operator key not authorized | Check `setAuthorizedProtocol` was called |
| LLM rate-limited | Burst from multiple findings | Add semaphore around `llm.Complete` |
| Schema round-trip fails | Hand-edited Go but not Rust (or vice versa) | Re-run Task 0.9 after every schema change |
| Anvil OOM during fork | Too many forks left running | Kill stale anvil PIDs in cleanup |
| x402 returns 200 instead of 402 | KeeperHub gate not configured | Re-check endpoint registration; consult their docs |

---

## Glossary

- **AXL** — Gensyn's peer-to-peer pub/sub network used as the swarm's nervous system.
- **iNFT (ERC-7857)** — Intelligent NFT standard on 0G; a smart-contract identity with mutable state.
- **x402** — HTTP 402 Payment Required protocol used by KeeperHub to gate paid content with stablecoin payments.
- **Differential check** — Verification step that re-runs the exploit against a patched contract to confirm the patch blocks it.
- **DVD** — Damn Vulnerable DeFi, the standard test corpus for EVM security tooling.
- **Triple-verify** — The pipeline's verification gate: live-fork + drain-threshold + differential.

---

## Final Checklist Before Submission

- [ ] `make build` clean from a fresh clone
- [ ] `make demo` succeeds end-to-end on a teammate's machine
- [ ] Demo video recorded and linked in README
- [ ] SECURITY.md and threat model committed
- [ ] iNFT deployed to 0G testnet, address in README
- [ ] One x402-gated report URL is live and reachable
- [ ] Pause switch tested in a recorded run
- [ ] All `<org>` placeholders in code replaced with real org name
- [ ] `.env` is gitignored; `.env.example` is committed
- [ ] No private keys in git history (`git log -p | grep -i 'private\|key'` returns nothing suspicious)

