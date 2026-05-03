# Project Status: Autonomous Web3 Security Swarm

## What's Done

### Builds & Tests
| Command | Result |
|---|---|
| `make build` | All 6 binaries compile (scout, auditor, orchestrator, contracts, dashboard) |
| `make test` | All tests pass (Go, Rust, Solidity) |
| `make roundtrip` | Go ↔ Rust schema verified |
| `go vet` | Clean on all 3 Go services |

### Code Written

**Scout (Go)**
- Mempool watcher (`newPendingTransactions` subscription)
- Deduper (content-ID LRU cache)
- Priority scorer (TVL × novelty × kind)
- GitHub commit watcher (**go-github/v62**, polls repos, deduped, rate-limited)
- Address watcher (monitors specific contracts & wallets for on-chain activity, emits targets)
- Etherscan source fetcher (verified contract source + multi-file support)
- AXL publisher (HTTP API wrapper, broadcasts to peers)
- Unified watch config (`configs/repos.yaml`: repos, contracts, wallets)

**Auditor (Rust)**
- Aderyn runner (shells out to binary, parses markdown)
- Slither runner (shells out to binary, parses JSON)
- Dual-source merge logic (filters by equivalence)
- Etherscan source fetcher (async reqwest, multi-file support)
- AXL subscriber/publisher (HTTP API wrapper)

**Orchestrator (Go)**
- LLM client (OpenAI-compatible, supports Anthropic/Ollama/vllm)
- Exploit generation with prompt template
- Foundry verification pipeline (fork test, drain extractor, differential check — placeholder)
- Report generation (teaser + full)
- AXL subscriber (HTTP API wrapper)
- x402 payment gate (KeeperHub integration with stub fallback)
- Safety: pause checker, rate limiter, audit logger
- iNFT client (Go bindings generated from ABI, wired into disclosure pipeline)
- Memory store (tamper-evident log)

**Dashboard (TUI)**
- Bubble Tea terminal UI with 5 tabs: Overview, Logs, Services, Charts, Config
- Live process manager: start/stop/restart all 4 agents from terminal
- 500-line ring buffer log viewer per service
- AXL node launcher from submodule (no manual `cd axl` needed)
- **Config editor**: add/remove repos, contracts, and wallet addresses from the TUI (with confirmation on delete, validation, and auto-save)

**Contracts (Solidity)**
- `SwarmINFT.sol` — iNFT with pause, authorization, disclosure tracking
- `VulnerableVault.sol` — Reentrant demo contract
- `Deploy.s.sol` — Foundry deployment script

**Config & Docs**
- `.env`, `.gitignore`, `flake.nix`, `shell.nix`
- `Makefile` with self-healing `check-tools` target
- `proto/messages.json` — shared schema
- `prompts/exploit_v1.md`, `prompts/report_v1.md`
- `README.md`, `SECURITY.md`, `docs/threat-model.md`

---

## Component Status

| Component | Status | Notes |
|---|---|---|
| Scout (mempool + GitHub + Address) | Complete | go-github/v62, Etherscan, address watcher for contracts & wallets |
| Auditor (Aderyn + Slither) | Complete | Dual-source merge, Etherscan fetcher |
| AXL Mesh | Complete | Dual-node setup: Node A (Go agents, port 9002) + Node B (Rust auditor, port 9003), peer key exchange |
| LLM Client | Complete | Auto-detects provider (OpenAI/Anthropic) |
| Foundry Verification | Complete | Fork test + drain extractor live |
| Differential Check | Placeholder | Returns `true`; opt-in via `ENABLE_DIFFERENTIAL=true` |
| x402 Payments | Complete | KeeperHub + stub fallback for dev |
| Dashboard TUI | Complete | Terminal UI, process manager, log viewer, config editor, dual AXL node launcher |
| 0G iNFT | Complete | Bindings generated, wired into disclosure |
| Rescue Lane | Disabled | Intentionally disabled for safety |

---

## Known Limitations

1. **Differential Check** — Always returns `true`. Real implementation patches contract via LLM, redeploys, re-runs. Expensive; intentionally opt-in.
2. **Rescue Lane** — Disabled by design. Would execute white-hat rescues on live protocols. Requires multisig authorization per protocol.
3. **x402 Stub Mode** — If `KEEPERHUB_API_KEY` unset, returns placeholder URLs for development.
4. **ETH Price** — Mempool watcher hardcodes $3000 ETH for TVL estimation. Use oracle for production.
5. **Audit Checkpoint** — `internal/safety/audit.go` writes literal `"checkpoint"` every 100 lines instead of computing a real SHA-256 hash.
6. **Dual-Node Bootstrap** — `AXL_PEERS_FOR_NODE_A` and `AXL_PEERS_FOR_NODE_B` must be populated with each other's public keys after the first run. This is a one-time setup step (documented in `.env`).

---

## Recently Completed

### Dual-Node AXL Mesh Refactor
- Split single AXL node into **Node A** (port 9002) and **Node B** (port 9003)
- Node A hosts Go-based **Scout** and **Orchestrator**
- Node B hosts Rust-based **Auditor**
- Updated environment variables: `AXL_API_URL_NODE_A`, `AXL_API_URL_NODE_B`, `AXL_PEERS_FOR_NODE_A`, `AXL_PEERS_FOR_NODE_B`
- Created separate node configs (`node-config-a.json`, `node-config-b.json`) with independent ed25519 keys
- Dashboard TUI now launches both nodes; `Start ALL` bootstraps nodes first, then agents after a 2-second delay

### 0G iNFT Integration
- Generated Go bindings from `SwarmINFT.abi.json` using `abigen`
- Wired `inft.Client` into orchestrator disclosure pipeline
- Added 0G config to `.env`: `OG_RPC_URL`, `OG_PRIVATE_KEY`, `OG_INFT_ADDRESS`
- Fixed `safety/pause.go` to use new `IsPaused(tokenID)` signature

### Config-Driven Watch Lists & Address Watcher
- Extended `configs/repos.yaml` schema to include `contracts` and `wallets` fields
- Built `AddressWatcher` in Scout that polls the chain for transactions involving watched contracts/wallets
- Emits targets with priority scoring when activity is detected
- Dashboard TUI Config tab provides full CRUD for repos, contracts, and wallets

### GitHub Commit Watcher
- Added `go-github/v62` dependency to scout-go
- Implemented `GitHubWatcher` with repo polling, commit deduplication, rate-limit awareness
- Reads `configs/repos.yaml` for watched repos
- Persists last-seen SHA to `.scout-state.json`

### Terminal Dashboard (TUI)
- Replaced placeholder HTTP server with Bubble Tea terminal UI
- 5 tabs: Overview (stats), Logs (live stream), Services (process manager), Charts (visualization), Config (watch list editor)
- Start/stop/restart all agents including AXL node from `axl/` submodule
- 500-line ring buffer per service for log history
- Live sparklines, funnel charts, activity timeline, and service health bars

### Nix Flake
- Added `flake.nix` + `shell.nix` for reproducible dev environment
- Includes Go, Rust, Foundry, Node, Python+Slither, utilities
- Auto-installs Aderyn on first `nix develop` entry
- `make build` runs `check-tools` which verifies and auto-installs missing tools

---

## Build Commands

```bash
make build          # Build all components (runs tool check first)
make test           # Run all tests (Go + Rust + Solidity)
make roundtrip      # Test Go ↔ Rust schema
make run-dashboard  # Launch terminal dashboard
make clean          # Clean all build artifacts
```

## Environment Setup

```bash
# Option A: Nix (fully reproducible)
nix develop

# Option B: Manual
make setup          # Checks tools, auto-installs aderyn if missing
make build
```

Copy `.env.example` → `.env` and fill in API keys before running agents.
