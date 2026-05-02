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
- Etherscan source fetcher (verified contract source + multi-file support)
- AXL publisher (HTTP API wrapper, broadcasts to peers)

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
- Bubble Tea terminal UI with 3 tabs: Overview, Logs, Services
- Live process manager: start/stop/restart all 4 agents from terminal
- 500-line ring buffer log viewer per service
- AXL node launcher from submodule (no manual `cd axl` needed)

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
| Scout (mempool + GitHub) | Complete | go-github/v62, Etherscan, deduping, scoring |
| Auditor (Aderyn + Slither) | Complete | Dual-source merge, Etherscan fetcher |
| AXL Mesh | Complete | HTTP API pub/sub, topic routing |
| LLM Client | Complete | Auto-detects provider (OpenAI/Anthropic) |
| Foundry Verification | Complete | Fork test + drain extractor live |
| Differential Check | Placeholder | Returns `true`; opt-in via `ENABLE_DIFFERENTIAL=true` |
| x402 Payments | Complete | KeeperHub + stub fallback for dev |
| Dashboard TUI | Complete | Terminal UI, process manager, log viewer |
| 0G iNFT | Complete | Bindings generated, wired into disclosure |
| Rescue Lane | Disabled | Intentionally disabled for safety |

---

## Known Limitations

1. **Differential Check** — Always returns `true`. Real implementation patches contract via LLM, redeploys, re-runs. Expensive; intentionally opt-in.
2. **Rescue Lane** — Disabled by design. Would execute white-hat rescues on live protocols. Requires multisig authorization per protocol.
3. **x402 Stub Mode** — If `KEEPERHUB_API_KEY` unset, returns placeholder URLs for development.
4. **ETH Price** — Mempool watcher hardcodes $3000 ETH for TVL estimation. Use oracle for production.
5. **Audit Checkpoint** — `internal/safety/audit.go` writes literal `"checkpoint"` every 100 lines instead of computing a real SHA-256 hash.

---

## Recently Completed

### 0G iNFT Integration
- Generated Go bindings from `SwarmINFT.abi.json` using `abigen`
- Wired `inft.Client` into orchestrator disclosure pipeline
- Added 0G config to `.env`: `OG_RPC_URL`, `OG_PRIVATE_KEY`, `OG_INFT_ADDRESS`
- Fixed `safety/pause.go` to use new `IsPaused(tokenID)` signature

### GitHub Commit Watcher
- Added `go-github/v62` dependency to scout-go
- Implemented `GitHubWatcher` with repo polling, commit deduplication, rate-limit awareness
- Reads `configs/repos.yaml` for watched repos
- Persists last-seen SHA to `.scout-state.json`

### Terminal Dashboard (TUI)
- Replaced placeholder HTTP server with Bubble Tea terminal UI
- 3 tabs: Overview (stats), Logs (live stream), Services (process manager)
- Start/stop/restart all agents including AXL node from `axl/` submodule
- 500-line ring buffer per service for log history

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
