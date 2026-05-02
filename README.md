# Autonomous Web3 Security Swarm

An agentic mesh of security bots that autonomously discovers, analyzes, verifies, and discloses smart-contract vulnerabilities.

## Architecture

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────────┐
│  Scout (Go)     │────▶│  AXL Mesh    │────▶│  Auditor (Rust)  │
│  - mempool      │     │  pub/sub     │     │  - aderyn        │
│  - github       │     │  topics:     │     │  - slither       │
│  - immunefi     │     │  - targets   │     │  - etherscan     │
└─────────────────┘     │  - findings  │     └──────────────────┘
                        │  - verified  │              │
                        │  - published │              ▼
                        └──────────────┘     ┌──────────────────┐
                              ▲              │ Orchestrator (Go)│
                              │              │ - LLM exploit gen│
                              │              │ - Foundry verify │
                              └──────────────│ - x402 publish   │
                                             │ - 0G iNFT record │
                                             └──────────────────┘
```

**Agents:**
- **Scout** — Watches Ethereum mempool (Sepolia), GitHub commits (Aave, Uniswap, Compound), and Immunefi for new targets. Emits `Target` messages.
- **Auditor** — Consumes targets, fetches verified source from Etherscan, runs Aderyn + Slither dual-source static analysis. Emits `Finding` messages.
- **Orchestrator** — Consumes findings, prompts LLM (OpenAI/Anthropic/Ollama) to generate exploit PoC, runs Foundry live-fork verification, generates paywalled reports (x402), and records disclosures on 0G iNFT.
- **Dashboard** — HTTP server subscribing to AXL topics for real-time observability.

**Mesh:** Gensyn AXL provides topic-based pub/sub across all agents.

**Hackathon Stack:**
- [Gensyn AXL](https://gensyn.ai) — decentralized mesh messaging
- [KeeperHub](https://keeperhub.io) — x402 payment gating
- [0G](https://0g.ai) — iNFT sovereignty & on-chain disclosure tracking

---

## Prerequisites

You have two options for setting up the environment:

### Option A: Nix (Recommended — fully reproducible)

If you have [Nix](https://nixos.org/) with flakes enabled:

```bash
nix develop
```

This drops you into a shell with Go, Rust, Foundry, Node.js, Python, Slither, and all other dependencies pre-installed. **Aderyn is automatically installed on first entry** if not already present. No manual steps needed.

If you don't have flakes enabled, use the legacy shell:
```bash
nix-shell
```

### Option B: Manual Install

| Tool | Version | Verify |
|---|---|---|
| Go | ≥ 1.22 | `go version` |
| Rust + Cargo | ≥ 1.75 | `cargo --version` |
| Foundry (forge, anvil, cast) | latest | `forge --version` |
| Node.js | ≥ 20 | `node --version` |
| Slither | latest | `slither --version` |
| Aderyn | latest | `aderyn --version` |
| `make` | any | `make --version` |
| `jq` | any | `jq --version` |

Install Foundry:
```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

Install Slither:
```bash
pip install slither-analyzer
```

Install Aderyn:
```bash
cargo install aderyn
```

---

## Quick Start

### 1. Clone & Build

```bash
git clone <repo>
cd <repo>
make build
```

### 2. Configure Environment

Copy `.env` to `.env.local` and fill in your keys:

```bash
cp .env .env.local
```

Required variables (see `.env` for full list):

```bash
# RPC endpoints (Alchemy/Infura)
SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY
MAINNET_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY

# LLM (pick one)
OPENAI_API_KEY=sk-...
# or
ANTHROPIC_API_KEY=sk-ant-...
# or local: LLM_BASE_URL=http://localhost:11434/v1

# Etherscan
ETHERSCAN_API_KEY=YOUR_KEY

# GitHub (for commit watcher)
GITHUB_TOKEN=ghp_...

# 0G Testnet (optional, for iNFT)
OG_RPC_URL=https://rpc-testnet.0g.ai
OG_PRIVATE_KEY=0x...
OG_INFT_ADDRESS=0x... # after deploying SwarmINFT.sol

# AXL Mesh (optional, local mode works without)
AXL_API_URL=http://127.0.0.1:9002
AXL_PEER_KEYS=

# KeeperHub x402 (optional, stub mode works without)
KEEPERHUB_API_KEY=
REPORT_PRICE_USD=1000
```

### 3. Run Tests

```bash
make test
```

Runs:
- `go test ./...` in scout-go & orchestrator-go
- `cargo test` in auditor-rs
- `forge test` in contracts

### 4. Run the Scout

```bash
cd services/scout-go
SWARM_PRIVATE_KEY=0x... go run ./cmd
```

The scout will:
- Subscribe to Sepolia mempool for contract deployments & large transfers
- Poll GitHub repos (`configs/repos.yaml`) for new commits
- Publish deduplicated, scored targets to AXL mesh topic `targets/discovered`

### 5. Run the Auditor

```bash
cd services/auditor-rs
cargo run --release
```

The auditor will:
- Subscribe to `targets/discovered`
- Fetch verified source from Etherscan
- Run Aderyn + Slither, merge results
- Publish findings to `analysis/findings`

### 6. Run the Orchestrator

```bash
cd services/orchestrator-go
OPENAI_API_KEY=sk-... go run ./cmd
```

The orchestrator will:
- Subscribe to `analysis/findings`
- Generate exploit PoC via LLM (`prompts/exploit_v1.md`)
- Run Foundry fork test, extract drain amount
- Generate teaser + full report (`prompts/report_v1.md`)
- Create x402-gated payment URL (stub mode if no KeeperHub key)
- Record disclosure on 0G iNFT (if configured)
- Publish to `disclosure/published`

### 7. Run the TUI Dashboard

```bash
cd services/dashboard/server
go run .
# or simply: make run-dashboard
```

A beautiful terminal UI opens with **3 tabs**:
- **Overview** — Live pipeline stats, agent statuses
- **Logs** — Real-time log stream of selected service (500-line ring buffer)
- **Services** — Start/stop/restart any agent from the terminal

**Keybindings:**
| Key | Action |
|---|---|
| `1-3` / `Tab` / `←→` | Switch tabs |
| `↑↓` / `j/k` | Select service (Services tab) |
| `Enter` / `s` | Toggle start/stop selected service |
| `a` | Start ALL services |
| `x` | Stop ALL services |
| `r` | Restart selected service |
| `Ctrl+L` | Clear logs |
| `q` / `Ctrl+C` | Quit (gracefully stops all child processes) |

The dashboard **automatically launches the AXL node** from the `axl/` submodule when you start it — no need to `cd axl && ./node` manually.

---

### 8. Deploy the iNFT Contract (0G Testnet)

```bash
cd contracts
forge script script/Deploy.s.sol --rpc-url $OG_RPC_URL --broadcast
```

Update `.env` with the deployed `OG_INFT_ADDRESS`.

---

## Build Commands

```bash
make build       # Build all services + contracts
make test        # Run all tests
make roundtrip   # Test Go ↔ Rust schema roundtrip
make clean       # Clean build artifacts
make demo        # Launch full demo stack

# Component-specific tests
make test-analyzers    # Run Aderyn+Slither test harness
make test-exploit-gen  # Run LLM exploit generation test
```

---

## Project Status

All major components implemented. See `PROJECT_STATUS.md` for detailed status.

| Component | Status |
|---|---|
| Scout (mempool + GitHub) | ✅ Complete |
| AXL Mesh (pub/sub) | ✅ Complete |
| Auditor (Aderyn + Slither) | ✅ Complete |
| LLM Client (exploit gen) | ✅ Complete |
| Foundry Verification | ✅ Complete |
| x402 Payment Gating | ✅ Complete (with stub fallback) |
| 0G iNFT | ✅ Complete |
| Dashboard | ✅ Complete |

---

## Documentation

- `PROJECT_STATUS.md` — What's done & what's left
- `AGENT_IMPLEMENTATION_PLAN.md` — Full phase-by-phase build guide
- `AXL_INTEGRATION.md` — Mesh setup & topic reference
- `X402_INTEGRATION.md` — Payment gating setup
- `docs/threat-model.md` — Security assumptions

---

## Known Limitations & Design Decisions

1. **Differential Check** (`internal/verify/differential.go`) — Returns `true` by default. To enable real differential verification (patch contract via LLM, redeploy, re-run exploit), set `ENABLE_DIFFERENTIAL=true`. This is expensive and intentionally opt-in.

2. **Rescue Lane** (`internal/disclosure/rescue.go`) — **Disabled by design** for the hackathon. The rescue lane would execute white-hat rescues on live protocols. Requires multisig authorization per protocol. Documented in `AGENT_IMPLEMENTATION_PLAN.md` Phase 8.3.

3. **x402 Stub Mode** — If `KEEPERHUB_API_KEY` is unset, the payment gate returns placeholder URLs for development. Set a real API key for production.

4. **ETH Price** — Mempool watcher uses a hardcoded $3000 ETH price for TVL estimation. For production, integrate a Chainlink or Coingecko oracle.

5. **Dashboard WebSocket** — The `/ws` endpoint returns 200. Full WebSocket streaming of AXL events to browser is a TODO for the frontend.

6. **Audit Checkpoint** — `internal/safety/audit.go` writes a placeholder "checkpoint" string every 100 lines instead of computing a real memory hash. The `internal/memory/store.go` package exists for this; integration is a TODO.

---

## License

MIT — See `SECURITY.md` for responsible disclosure guidelines.
