# Autonomous Web3 Security Swarm

An agentic mesh of security bots that autonomously discovers, analyzes, verifies, and discloses smart-contract vulnerabilities.

## Architecture

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────────┐
│  Scout (Go)     │────▶│  AXL Node A  │     │  Auditor (Rust)  │
│  - mempool      │     │  (port 9002) │◄────│  - aderyn        │
│  - github       │     │     mesh     │     │  - slither       │
│  - immunefi     │     │   network    │     │  - etherscan     │
└─────────────────┘     └──────┬───────┘     └──────────────────┘
                               │                         │
┌─────────────────┐            │                         │
│ Orchestrator(Go)│◄───────────┘                         │
│ - LLM exploit   │                                      │
│ - Foundry verify│            ┌──────────────┐         │
│ - x402 publish  │            │  AXL Node B  │◄────────┘
│ - 0G iNFT record│            │  (port 9003) │
└─────────────────┘            └──────────────┘

Topics: targets/discovered  |  analysis/findings  |  exploit/verified
```

**Agents:**
- **Scout** — Watches Ethereum mempool (Sepolia), GitHub commits, Immunefi, and **specific contract & wallet addresses** for new targets. Emits `Target` messages.
- **Auditor** — Consumes targets, fetches verified source from Etherscan, runs Aderyn + Slither dual-source static analysis. Emits `Finding` messages.
- **Orchestrator** — Consumes findings, prompts LLM (OpenAI/Anthropic/Ollama) to generate exploit PoC, runs Foundry live-fork verification, generates paywalled reports (x402), and records disclosures on 0G iNFT.
- **Dashboard** — Terminal UI (Bubble Tea) with live process management, log viewer, and a built-in **Config Editor** for managing watch lists.

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

**Minimal setup for a demo** (only need a few keys):

```bash
# Required — pick at least one LLM provider
OPENAI_API_KEY=sk-...
# or ANTHROPIC_API_KEY=sk-ant-...
# or LLM_BASE_URL=http://localhost:11434/v1

# Required — RPC for blockchain access
MAINNET_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY

# Optional — for GitHub commit watcher
GITHUB_TOKEN=ghp_...

# Optional — for 0G iNFT on-chain recording
# OG_RPC_URL=https://rpc-testnet.0g.ai
# OG_PRIVATE_KEY=0x...
# OG_INFT_ADDRESS=0x...

# Optional — KeeperHub x402 production payments
# KEEPERHUB_API_KEY=...

# AXL (Gensyn) mesh — updated for dual-node setup
# Node A port 9002 (Scout + Orchestrator), Node B port 9003 (Auditor)
# AXL_API_URL_NODE_A=http://127.0.0.1:9002
# AXL_API_URL_NODE_B=http://127.0.0.1:9003
# AXL_PEERS_FOR_NODE_A=<node_b_public_key>
# AXL_PEERS_FOR_NODE_B=<node_a_public_key>
```

### 3. Run Everything from the Dashboard (Recommended)

The TUI dashboard is the single entry point. Start it and launch agents from there:

```bash
make run-dashboard
```

This opens the terminal UI. From there:

1. Press `3` to switch to **Services** tab
2. Use `↓` to select **AXL Node A**
3. Press `Enter` to start it
4. Repeat for **AXL Node B**, **Scout**, **Auditor**, **Orchestrator**
5. Or press `a` to start **ALL** at once — nodes boot first, then agents start after a 2-second delay
6. Press `5` to open the **Config** tab and manage watched repos, contracts, and wallets

Watch the **Overview** tab (`1`) for live pipeline stats, the **Logs** tab (`2`) for real-time output, and the **Charts** tab (`4`) for visualizations.

**Keybindings:**

| Key | Action |
|---|---|
| `1-5` | Switch tabs (Overview / Logs / Services / Charts / Config) |
| `↑↓` / `j/k` | Select service / item |
| `Tab` / `←→` | Switch sub-section (in Config tab: Repos / Contracts / Wallets) |
| `Enter` / `s` | Toggle start/stop selected service |
| `a` | Start ALL services (or Add item in Config editor) |
| `x` | Stop ALL services (or Remove item in Config editor — confirms first) |
| `r` | Restart selected service (or Reload config in Config editor) |
| `Ctrl+L` | Clear logs |
| `q` / `Ctrl+C` | Quit |

### 4. Run Services Manually (Alternative)

If you prefer running each agent in its own terminal, first start the AXL nodes, then the agents:

**Terminal 1 — AXL Node A:**
```bash
cd axl
./node -config node-config-a.json
```

**Terminal 2 — AXL Node B:**
```bash
cd axl
./node -config node-config-b.json
```

**Terminal 3 — Scout:**
```bash
cd services/scout-go
AXL_API_URL_NODE_A=http://127.0.0.1:9002 AXL_PEERS_FOR_NODE_A=... SWARM_PRIVATE_KEY=0x... go run ./cmd
```

**Terminal 4 — Auditor:**
```bash
cd services/auditor-rs
AXL_API_URL_NODE_B=http://127.0.0.1:9003 AXL_PEERS_FOR_NODE_B=... cargo run --release
```

**Terminal 5 — Orchestrator:**
```bash
cd services/orchestrator-go
AXL_API_URL_NODE_A=http://127.0.0.1:9002 AXL_PEERS_FOR_NODE_A=... OPENAI_API_KEY=sk-... go run ./cmd
```

### 5. Verify the Pipeline Works

1. **Scout** observes a target (mempool event or GitHub commit)
2. **Auditor** receives it via AXL, runs Aderyn + Slither
3. **Orchestrator** receives the finding, prompts LLM for exploit
4. Exploit is generated to `/tmp/orchestrator/exploits/`
5. Foundry fork test runs — if it passes, a report is generated
6. Disclosure is published to AXL mesh and recorded on 0G iNFT

Check logs in the dashboard or use `axl-tail`:
```bash
cd services/scout-go && go run ./cmd/axl-tail
```

### 6. Deploy the iNFT Contract (0G Testnet)

```bash
cd contracts
export OG_PRIVATE_KEY=0x...
forge script script/Deploy.s.sol --rpc-url https://rpc-testnet.0g.ai --broadcast
```

Copy the deployed address into `.env` as `OG_INFT_ADDRESS`.

---

## Build & Test

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

| Component | Status | Notes |
|---|---|---|
| Scout (mempool + GitHub + Address) | ✅ Complete | Publishes targets to AXL; watches specific contracts & wallets |
| AXL Mesh (pub/sub) | ✅ Complete | Dual-node: Node A (port 9002, Go agents) + Node B (port 9003, Rust auditor) |
| Auditor (Aderyn + Slither) | ✅ Complete | Fetches source, runs analyzers, publishes findings |
| LLM Client (exploit gen) | ✅ Complete | Multi-provider: OpenAI, Anthropic, Ollama, 0G Compute |
| Foundry Verification | ✅ Complete | Fork tests via `infra/forge-harness/` |
| x402 Payment Gating | ✅ Complete | KeeperHub Workflow API; stub mode when `KEEPERHUB_API_KEY` unset |
| 0G Storage | ✅ Complete | Full reports uploaded; root hash used as iNFT memory pointer |
| 0G iNFT Recording | ✅ Complete | Records disclosure + storage hash on-chain |
| Dashboard TUI | ✅ Complete | Process manager, live log streaming, watch list config editor |
| Differential Check | ⚠️ Stub | Always passes; set `ENABLE_DIFFERENTIAL=true` to activate |
| Rescue Lane | 🚫 Disabled | Intentionally off for hackathon (requires multisig) |

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

5. **0G Storage Fallback** — When `STORAGE_GATEWAY_URL` is blank or the gateway is unreachable, reports are stored locally at `/tmp/0g-storage-local/`. The iNFT memory pointer is still written; it just points to a local SHA-256 hash rather than a live 0G root hash.

6. **Audit Checkpoint** — `internal/safety/audit.go` writes a placeholder "checkpoint" string instead of a real memory hash. The `internal/memory/store.go` package exists for this; wiring is a TODO.
7. **Dual-Node Peer Keys** — `AXL_PEERS_FOR_NODE_A` and `AXL_PEERS_FOR_NODE_B` must be populated with each other's public keys after the nodes start once. Run `curl http://127.0.0.1:{port}/topology | jq -r '.our_public_key'` to get each node's key, then add them to `.env`.

---

## License

MIT — See `SECURITY.md` for responsible disclosure guidelines.
