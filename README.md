# Autonomous Web3 Security Swarm

An agentic mesh of security bots that autonomously discovers, analyzes, verifies, and discloses smart-contract and on-chain program vulnerabilities across **EVM**, **Solana programs**, and **Solana validator (Firedancer)** targets.

## Architecture

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────────┐
│  Scout (Go)     │────▶│  AXL Node A  │     │  Auditor (Rust)  │
│  - mempool      │     │  (port 9002) │◄────│  - EVM: aderyn   │
│  - github       │     │     mesh     │     │    + slither      │
│  - forta alerts │     │   network    │     │    + timelock     │
│  - addresses    │     └──────┬───────┘     │    + oracle check │
│  - solana repos │            │             │    + bridge scan  │
└─────────────────┘            │             │  - Solana: 9 tools│
                               │             │  - C: cppcheck   │
┌─────────────────┐            │             │    + clang-tidy   │
│ Orchestrator(Go)│◄───────────┘             └──────────────────┘
│ - LLM exploit   │                                      │
│ - EVM: Foundry  │            ┌──────────────┐         │
│ - Solana: test- │            │  AXL Node B  │◄────────┘
│   validator     │            │  (port 9003) │
│ - C: cluster    │            └──────────────┘
│ - x402 publish  │
│ - 0G iNFT record│
└─────────────────┘

Topics: targets/discovered  |  analysis/findings  |  exploit/verified
```

**Agents:**
- **Scout** — Watches Ethereum mempool (Sepolia), GitHub commits across EVM + Solana repos, Forta Network alerts, and specific contract/wallet addresses. Emits `Target` messages tagged with `bountyType`.
- **Auditor** — Consumes targets, runs the appropriate analyzer stack per `bountyType`, emits `Finding` messages.
- **Orchestrator** — Consumes findings, prompts LLM to generate exploit PoC, runs the appropriate verifier (Foundry fork / Solana test-validator / Firedancer cluster), generates paywalled reports (x402), and records disclosures on 0G iNFT.
- **Dashboard** — Terminal UI (Bubble Tea) with live process management, log viewer, and a built-in Config Editor for managing watch lists.

**Mesh:** Gensyn AXL provides topic-based pub/sub across all agents.

---

## Supported Target Types

| `bountyType` | Language | Analyzers | Verifier |
|---|---|---|---|
| `solidity-evm` | Solidity | Aderyn, Slither, timelock detector, oracle checker, bridge scanner | Foundry fork test |
| `solana-program` | Rust / Anchor | cargo-audit, cargo-deny, cargo-geiger, clippy, solana-patterns, semgrep, soteria, anchor-idl, trident | `solana-test-validator` |
| `firedancer` | C | cppcheck, clang-tidy, solfuzz harnesses, known-issues filter | Firedancer cluster |

---

## Hackathon Stack

- [Gensyn AXL](https://gensyn.ai) — decentralized mesh messaging
- [KeeperHub](https://keeperhub.io) — x402 payment gating
- [0G](https://0g.ai) — iNFT sovereignty & on-chain disclosure tracking

---

## Prerequisites

You have two options for setting up the environment:

### Option A: Nix (Recommended — fully reproducible)

```bash
nix develop
```

This drops you into a shell with Go, Rust, Foundry, Node.js, Python, Slither, and all other dependencies pre-installed. Aderyn is automatically installed on first entry.

If you don't have flakes enabled:
```bash
nix-shell
```

### Option B: Manual Install

**Core tools:**

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

**Additional tools for Solana program scanning:**

| Tool | Install | Purpose |
|---|---|---|
| Solana CLI | `sh -c "$(curl -sSfL https://release.solana.com/stable/install)"` | `solana-test-validator` for PoC verification |
| Anchor CLI | `npm i -g @coral-xyz/anchor-cli` | IDL checker + Trident fuzzer |
| cargo-audit | `cargo install cargo-audit` | RustSec advisory database |
| cargo-deny | `cargo install cargo-deny` | License/policy enforcement |
| cargo-geiger | `cargo install cargo-geiger` | Unsafe block detection |
| Trident | `cargo install trident-cli` | Anchor fuzzing framework |
| Semgrep | `pip install semgrep` | `p/solana` ruleset scanner |
| Soteria | `npm i -g @soteria-dev/sot` | Solana-specific static analyzer |

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

**Minimal setup for a demo:**

```bash
# Required — pick at least one LLM provider
OPENAI_API_KEY=sk-...
# or ANTHROPIC_API_KEY=sk-ant-...

# Required — RPC for EVM blockchain access
MAINNET_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY

# Optional — for GitHub commit watcher (higher rate limits)
GITHUB_TOKEN=ghp_...

# Optional — Forta Network alert feed
# FORTA_API_KEY=...

# Optional — Solana program scanning
# SOLANA_RPC_URL=https://api.mainnet-beta.solana.com
SOLANA_TEST_VALIDATOR_URL=http://127.0.0.1:8899
TRIDENT_FUZZ_SECS=30

# Optional — 0G iNFT on-chain recording
# OG_RPC_URL=https://evmrpc-testnet.0g.ai
# OG_PRIVATE_KEY=0x...
# OG_INFT_ADDRESS=0x9756eD45Fe95d53b0d72F5efe2977Df1c876089c

# Optional — KeeperHub x402 production payments
# KEEPERHUB_API_KEY=...

# AXL (Gensyn) mesh — dual-node setup
# AXL_API_URL_NODE_A=http://127.0.0.1:9002
# AXL_API_URL_NODE_B=http://127.0.0.1:9003
```

### 3. Run Everything from the Dashboard (Recommended)

```bash
make run-dashboard
```

This opens the terminal UI. From there:

1. Press `3` to switch to **Services** tab
2. Use `↓` to select **AXL Node A**
3. Press `Enter` to start it
4. Repeat for **AXL Node B**, **Scout**, **Auditor**, **Orchestrator**
5. Or press `a` to start **ALL** at once — nodes boot first, agents start after a 2-second delay
6. Press `5` to open the **Config** tab to manage watched repos, contracts, and wallets

Watch the **Overview** tab (`1`) for live pipeline stats, **Logs** (`2`) for real-time output, **Charts** (`4`) for visualizations.

**Keybindings:**

| Key | Action |
|---|---|
| `1-5` | Switch tabs (Overview / Logs / Services / Charts / Config) |
| `↑↓` / `j/k` | Select service / item |
| `Tab` / `←→` | Switch sub-section (in Config tab: Repos / Contracts / Wallets) |
| `Enter` / `s` | Toggle start/stop selected service |
| `a` | Start ALL services (or Add item in Config editor) |
| `x` | Stop ALL services (or Remove item in Config editor — confirms first) |
| `r` | Restart selected service |
| `Ctrl+L` | Clear logs |
| `q` / `Ctrl+C` | Quit |

### 4. Run Services Manually (Alternative)

**Terminal 1 — AXL Node A:**
```bash
cd axl && ./node -config node-config-a.json
```

**Terminal 2 — AXL Node B:**
```bash
cd axl && ./node -config node-config-b.json
```

**Terminal 3 — Scout:**
```bash
cd services/scout-go
AXL_API_URL_NODE_A=http://127.0.0.1:9002 go run ./cmd
```

**Terminal 4 — Auditor:**
```bash
cd services/auditor-rs
AXL_API_URL_NODE_B=http://127.0.0.1:9003 cargo run --release
```

**Terminal 5 — Orchestrator:**
```bash
cd services/orchestrator-go
AXL_API_URL_NODE_A=http://127.0.0.1:9002 OPENAI_API_KEY=sk-... go run ./cmd
```

### 5. Verify the Pipeline Works

**EVM path:**
1. Scout observes a Solidity contract deployment or GitHub commit (or Forta CRITICAL/HIGH alert)
2. Auditor fetches source from Etherscan, runs Aderyn + Slither + timelock/oracle/bridge checkers
3. Orchestrator prompts LLM → Foundry fork test → x402 report → 0G iNFT

**Solana path:**
1. Scout detects a new commit on a watched Solana program repo (tagged `bountyType: solana-program`)
2. Auditor clones the repo and runs the 9-tool Solana stack in sequence:
   `cargo-audit` → `cargo-deny` → `cargo-geiger` → `clippy` → `solana-patterns` → `semgrep` → `soteria` → `anchor-idl` → `trident`
3. Orchestrator prompts LLM (using `prompts/exploit_solana_v1.md`) → runs PoC against `solana-test-validator` → publishes report

**Firedancer path:**
1. Scout detects a new commit on `firedancer-io/firedancer` (tagged `bountyType: firedancer`)
2. Auditor clones and runs cppcheck + clang-tidy + solfuzz harnesses
3. Orchestrator runs cluster verification

Check logs or use `axl-tail`:
```bash
cd services/scout-go && go run ./cmd/axl-tail
```

### 6. iNFT Contract (0G Testnet)

| Field | Value |
|---|---|
| Contract Address | `0x9756eD45Fe95d53b0d72F5efe2977Df1c876089c` |
| Explorer | [View on ChainScan](https://chainscan-galileo.0g.ai/address/0x9756eD45Fe95d53b0d72F5efe2977Df1c876089c) |

To re-deploy:
```bash
cd contracts
export OG_PRIVATE_KEY=0x...
forge script script/Deploy.s.sol --rpc-url https://evmrpc-testnet.0g.ai --broadcast
```

---

## Build & Test

```bash
make build                   # Build all services + contracts
make test                    # Run all tests
make roundtrip               # Test Go ↔ Rust schema roundtrip
make clean                   # Clean build artifacts

# Component-specific
make test-analyzers          # Run Aderyn+Slither test harness
make test-exploit-gen        # Run LLM exploit generation test
make build-vulnerable-vault  # Build the Solana deliberate-vuln test fixture
```

---

## Test Fixtures

### Solana: VulnerableVault

`test-fixtures/solana/vulnerable-vault/` is a fully runnable Anchor workspace designed to exercise every check in the 9-tool Solana analyzer pipeline.

**8 deliberate vulnerabilities in `programs/vulnerable-vault/src/lib.rs`:**

| # | Instruction | Vulnerability | Severity |
|---|---|---|---|
| 1 | `deposit` | Integer overflow (unchecked `+`) | High |
| 2 | `withdraw` | Missing signer check (`AccountInfo` instead of `Signer`) | Critical |
| 3 | `set_fee` | No access control (any caller can change fee) | High |
| 4 | `cpi_proxy` | Arbitrary CPI (attacker-controlled target program) | Critical |
| 5 | `load_user` | Missing ownership check on data account | High |
| 6 | `create_escrow` | Missing bump canonicalization (`create_program_address` with caller-supplied bump) | High |
| 7 | `close_and_send` | Insecure account close (lamports drained, data not zeroed) | Medium |
| 8 | `admin_withdraw` | Account confusion via `try_deserialize_unchecked` | Critical |

**`poc/exploit.ts`** demonstrates all 8 against a local `solana-test-validator` and exits 0 when all pass.

```bash
cd test-fixtures/solana/vulnerable-vault
make exploit   # builds program, starts validator, runs PoC
```

### EVM: VulnerableVault.sol

`contracts/src/VulnerableVault.sol` — reentrancy demo contract used to validate the Foundry fork-test verifier.

---

## Watched Repositories

Configured in `services/scout-go/configs/repos.yaml`.

**EVM / Solidity:**
- aave/aave-v3-core, Uniswap/v4-core, compound-finance/compound-protocol
- makerdao/dss, curvefi/curve-contract, dydxprotocol/v4-chain, gmx-io/gmx-synthetics

**Solana programs (Rust / Anchor) — `bountyType: solana-program`:**
- coral-xyz/anchor, solana-labs/solana-program-library
- marinade-finance/liquid-staking-program, raydium-io/raydium-amm
- orca-so/whirlpools, jito-foundation/jito-programs
- drift-labs/protocol-v2, metaplex-foundation/mpl-token-metadata

**Solana validator (C) — `bountyType: firedancer`:**
- firedancer-io/firedancer

---

## Project Status

| Component | Status | Notes |
|---|---|---|
| Scout (mempool + GitHub + Forta + Address) | ✅ Complete | EVM + Solana program repos + Firedancer; Forta alert feed |
| AXL Mesh (pub/sub) | ✅ Complete | Dual-node: Node A (port 9002) + Node B (port 9003) |
| Auditor — EVM | ✅ Complete | Aderyn, Slither, timelock, oracle, bridge detectors |
| Auditor — Solana (9 tools) | ✅ Complete | cargo-audit, cargo-deny, cargo-geiger, clippy, solana-patterns, semgrep, soteria, anchor-idl, trident |
| Auditor — Firedancer (C) | ✅ Complete | cppcheck, clang-tidy, solfuzz harnesses |
| LLM Client (exploit gen) | ✅ Complete | Multi-provider: OpenAI, Anthropic, Ollama |
| Foundry Verification (EVM) | ✅ Complete | Fork tests via `infra/forge-harness/` |
| Solana Verification | ✅ Complete | `solana-test-validator` + PoC runner |
| Firedancer Verification | ✅ Complete | Cluster-level crash/mismatch/escape detection |
| x402 Payment Gating | ✅ Complete | KeeperHub Workflow API; stub mode when key unset |
| 0G Storage | ✅ Complete | Official SDK v1.3.0; SHA-256 fallback when offline |
| 0G iNFT Recording | ✅ Complete | Records disclosure + storage hash on-chain |
| Dashboard TUI | ✅ Complete | Process manager, log viewer, config editor |
| Test Fixtures | ✅ Complete | Solana: 8-vuln Anchor program + PoC; EVM: VulnerableVault.sol |
| Differential Check | ⚠️ Stub | Always passes; set `ENABLE_DIFFERENTIAL=true` to activate |
| Rescue Lane | 🚫 Disabled | Intentionally off (requires multisig) |

---

## Documentation

- `PROJECT_STATUS.md` — What's done & what's left
- `AGENT_IMPLEMENTATION_PLAN.md` — Full phase-by-phase build guide
- `AXL_INTEGRATION.md` — Mesh setup & topic reference
- `X402_INTEGRATION.md` — Payment gating setup
- `docs/threat-model.md` — Security assumptions
- `docs/adevar-labs-checklist.md` — DeFi pre-launch security checklist coverage mapping
- `bounties/firedancer.yaml` — Firedancer bounty manifest
- `bounties/solana_programs.yaml` — Solana program bounty manifest
- `prompts/exploit_v1.md` — EVM exploit LLM prompt
- `prompts/exploit_c_v1.md` — Firedancer exploit LLM prompt
- `prompts/exploit_solana_v1.md` — Solana program exploit LLM prompt

---

## Known Limitations & Design Decisions

1. **Differential Check** — Returns `true` by default. Set `ENABLE_DIFFERENTIAL=true` to enable real differential verification (expensive; opt-in).

2. **Rescue Lane** — Disabled by design for the hackathon. Requires multisig authorization per protocol.

3. **x402 Stub Mode** — If `KEEPERHUB_API_KEY` is unset, the payment gate returns placeholder URLs for development.

4. **ETH Price** — Mempool watcher uses a hardcoded $3000 ETH price for TVL estimation. Integrate Chainlink or Coingecko for production.

5. **0G Storage** — Falls back to local SHA-256 hashing at `/tmp/0g-storage-local/` if the network is unreachable.

6. **Solana tool availability** — All nine Solana analyzers are optional at runtime. If a tool is not installed, the auditor logs a warning and continues — the pipeline never crashes due to a missing tool.

7. **Trident fuzzing time** — Defaults to 30 seconds per target (`TRIDENT_FUZZ_SECS`). Increase for deeper fuzzing outside of CI.

8. **Dual-Node Peer Keys** — `AXL_PEERS_FOR_NODE_A` and `AXL_PEERS_FOR_NODE_B` must be populated after the nodes start once. Run `curl http://127.0.0.1:{port}/topology | jq -r '.our_public_key'` to get each key.

---

## License

MIT — See `SECURITY.md` for responsible disclosure guidelines.
