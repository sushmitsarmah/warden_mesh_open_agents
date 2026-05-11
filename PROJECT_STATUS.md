# Project Status: Autonomous Web3 Security Swarm

## What's Done

### Builds & Tests
| Command | Result |
|---|---|
| `make build` | All binaries compile (scout, auditor, orchestrator, contracts, dashboard) |
| `make test` | All tests pass (Go, Rust, Solidity) |
| `make roundtrip` | Go ↔ Rust schema verified |
| `go vet` | Clean on all 3 Go services |
| `cargo check` | Clean (warnings only, zero errors) |

---

## Component Status

| Component | Status | Notes |
|---|---|---|
| Scout (mempool + GitHub + Address + Forta) | ✅ Complete | EVM + Solana repos + Firedancer; Forta alert feed; dual GitHub watcher goroutines |
| Auditor — EVM (Aderyn + Slither + 3 DeFi checkers) | ✅ Complete | Dual-source merge, Etherscan fetcher, timelock/oracle/bridge analyzers |
| Auditor — Solana (9 tools) | ✅ Complete | See full tool list below |
| Auditor — Firedancer (C) | ✅ Complete | cppcheck, clang-tidy, solfuzz harnesses, known-issues filter |
| AXL Mesh | ✅ Complete | Dual-node: Node A (Go agents, 9002) + Node B (Rust auditor, 9003) |
| LLM Client | ✅ Complete | OpenAI / Anthropic / Ollama / 0G Compute |
| EVM Verification (Foundry) | ✅ Complete | Fork test + drain extractor |
| Solana Verification | ✅ Complete | `solana-test-validator` + TypeScript/JS PoC runner |
| Firedancer Verification | ✅ Complete | Cluster crash / bank-hash-mismatch / sandbox-escape detection |
| x402 Payments | ✅ Complete | KeeperHub Workflow API + stub fallback |
| 0G Storage | ✅ Complete | Official SDK v1.3.0; SHA-256 fallback when offline |
| 0G iNFT Recording | ✅ Complete | On-chain disclosure + storage hash |
| Dashboard TUI | ✅ Complete | Process manager, log viewer, config editor, dual AXL launcher |
| Test Fixtures (Solana) | ✅ Complete | Deliberate-vuln Anchor program + TypeScript PoC; `make build-vulnerable-vault` |
| Differential Check | ⚠️ Stub | Always returns `true`; opt-in via `ENABLE_DIFFERENTIAL=true` |
| Rescue Lane | 🚫 Disabled | Requires multisig authorization — intentionally off |

---

## Code Written

### Scout (Go)
- Mempool watcher (`newPendingTransactions` subscription)
- Deduper (content-ID LRU cache)
- Priority scorer (TVL × novelty × kind)
- GitHub commit watcher — polls EVM repos with `bountyType: solidity-evm`
- **Solana GitHub watcher** — second watcher goroutine polls Solana program repos with `bountyType: solana-program`
- **Forta alert watcher** — polls Forta GraphQL API every 5 min; deduplicates by `alertId`; maps CRITICAL/HIGH/MEDIUM/LOW to priority; graceful no-op when `FORTA_API_KEY` unset
- Address watcher (monitors specific contracts & wallets, emits targets)
- Etherscan source fetcher
- AXL publisher
- Unified watch config (`configs/repos.yaml`: EVM repos, Solana repos, contracts, wallets)

### Auditor (Rust)

**EVM analyzers:**
- `aderyn.rs` — Aderyn static analysis, markdown parser
- `slither.rs` — Slither JSON parser
- `merge.rs` / `equivalences.rs` — dual-source merge and false-positive filter
- **`timelock.rs`** — detects privileged admin functions (`onlyOwner`, `onlyAdmin`, …) that affect funds or change critical params but are not protected by a `TimelockController`; flags missing timelock across all `.sol` files
- **`oracle_check.rs`** — 6 checks: `latestRoundData` staleness (missing `updatedAt`/`maxStaleness`), round validity, negative-price guard, deprecated `.latestAnswer()`, spot-price `slot0()`/`sqrtPriceX96` without TWAP, `getReserves()` division as price
- **`bridge_deps.rs`** — 7 bridge protocols (LayerZero, Wormhole, CCTP, Axelar, Stargate, Across, Hyperlane); detects import markers and flags missing required validation checks (trusted remote, emitter address, source domain, replay protection, etc.)

**Firedancer (C) analyzers:**
- `cppcheck.rs` — cppcheck XML parser, out-of-scope suppression
- `clang_tidy.rs` — clang-tidy JSON parser, GCC-only enforcement
- `fuzz_harness.rs` — solfuzz-agave harness runner (7 targets), bank-hash-mismatch detection
- `known_issues_checker.rs` — fetches all 20+ known-issue tracker URLs, fingerprint suppression

**Solana program analyzers (9 tools):**
- `cargo_audit.rs` — RustSec advisory database; parses `cargo audit --json`
- `cargo_deny.rs` — license, duplicate-dep, and untrusted-registry policy; writes default `deny.toml` if absent
- `cargo_geiger.rs` — counts unsafe blocks in root crate and deps; severity scales with count
- `clippy_solana.rs` — `cargo clippy --message-format=json` with Solana security lints (integer arithmetic, unwrap, indexing)
- `solana_patterns.rs` — regex file walk across all `.rs` files; 14 Solana-specific vulnerability patterns (missing signer check, arbitrary CPI, ownership check, bump canonicalization, SPL drain, account confusion, reentrancy, truncating casts, etc.)
- `semgrep_solana.rs` — `semgrep --config p/solana --json`; AST-based pattern matching with community ruleset
- `soteria.rs` — `sot check --json`; Solana-specific semantic static analysis
- `anchor_idl.rs` — builds Anchor IDL, flags mutable unconstrained accounts and signer-on-PDA mistakes
- `trident_fuzz.rs` — `trident fuzz run-hfuzz`; auto-generates fuzz harnesses from Anchor IDL; detects crashes, panics, and unexpected errors

### Orchestrator (Go)
- LLM client (OpenAI-compatible, multi-provider)
- Exploit generation with prompt dispatch by `bountyType`
- **EVM verifier** (`verify/forge.go`) — Foundry fork test, drain extractor, differential check stub
- **Solana verifier** (`verify/solana.go`) — clones repo, `cargo build-sbf`, starts `solana-test-validator`, runs TypeScript/JS PoC, interprets impact
- **Firedancer verifier** (`verify/cluster.go`) — build with GCC, inject PoC, check crash/mismatch/escape
- Scope check (`verify/scope_check.go`) — out-of-scope path, TODO/FIXME, bounty-config validation
- Known issues filter (`verify/known_issues_filter.go`) — duplicate detection, public-fix check
- Report generator (teaser + full report)
- AXL subscriber
- x402 payment gate (KeeperHub)
- Safety: pause checker, rate limiter, audit logger
- iNFT client (Go bindings, wired into disclosure pipeline)
- Memory store (tamper-evident log)

### Prompts
- `exploit_v1.md` — EVM Solidity Foundry PoC
- `exploit_c_v1.md` — Firedancer C crash PoC (GCC, remote attacker, solfuzz formats)
- `exploit_solana_v1.md` — Solana program PoC (7 vuln classes, TypeScript/Anchor client, `console.log("Exploit successful/failed")` contract)

### Bounty Manifests
- `bounties/firedancer.yaml` — complete Firedancer rules (compiler, attacker model, known issues, severity map)
- `bounties/solana_programs.yaml` — Solana program rules (10 repos, 9 analyzers, severity map, verifier config)

### Test Fixtures
- `test-fixtures/solana/vulnerable-vault/` — Fully runnable Anchor workspace
  - `programs/vulnerable-vault/src/lib.rs` — 8 deliberate vulnerabilities: integer overflow, missing signer, no access control, arbitrary CPI, missing ownership check, bump canonicalization, insecure close, account confusion
  - `poc/exploit.ts` — TypeScript PoC demonstrating all 8; outputs `"Exploit successful: all 8 vulnerabilities demonstrated"` matching `interpretSolanaResult()` in `verify/solana.go`
  - `package.json`, `tsconfig.json`, `Makefile`, `tests/` — complete scaffold; `make exploit` spins up a local validator and runs the PoC

### Contracts (Solidity)
- `SwarmINFT.sol` — iNFT with pause, authorization, disclosure tracking
- `VulnerableVault.sol` — reentrant demo contract
- `Deploy.s.sol` — Foundry deployment script

### Dashboard (TUI)
- Bubble Tea terminal UI — 5 tabs: Overview, Logs, Services, Charts, Config
- Live process manager: start/stop/restart all 4 agents + both AXL nodes
- 500-line ring buffer log viewer per service
- Config editor: add/remove repos, contracts, wallets with confirmation and auto-save

---

## Solana Analyzer Pipeline (in order)

```
bountyType: "solana-program"
        │
        ▼
1. cargo-audit      ← RustSec: known CVEs in Rust dependencies
2. cargo-deny       ← license / duplicate-dep / untrusted registry policy
3. cargo-geiger     ← unsafe block count (root crate + deps)
4. clippy-solana    ← integer arithmetic, unwrap, indexing lints
5. solana-patterns  ← 14 regex patterns (no build needed)
6. semgrep          ← p/solana AST ruleset
7. soteria          ← Solana-specific semantic analysis
8. anchor-idl       ← IDL constraint checker (Anchor programs only)
9. trident          ← honggfuzz harness (Anchor programs only)
        │
        ▼
    findings → AXL → Orchestrator
        │
        ▼
LLM (exploit_solana_v1.md prompt)
        │
        ▼
solana-test-validator verification
        │
        ▼
x402 report → 0G iNFT
```

All tools are optional at runtime — a missing binary logs a warning and returns empty findings rather than crashing the pipeline.

---

## EVM DeFi Checklist Coverage

Based on the Adevar Labs DeFi Pre-Launch Security Checklist (2026 Edition). See `docs/adevar-labs-checklist.md` for full mapping.

| Category | Detector | Status |
|---|---|---|
| Oracle safety (Chainlink staleness) | `oracle_check.rs` | ✅ |
| Oracle safety (negative price) | `oracle_check.rs` | ✅ |
| Oracle safety (deprecated API) | `oracle_check.rs` | ✅ |
| Spot price manipulation (`slot0`) | `oracle_check.rs` | ✅ |
| AMM price used as oracle | `oracle_check.rs` | ✅ |
| Admin function timelocks | `timelock.rs` | ✅ |
| Bridge message validation | `bridge_deps.rs` | ✅ (7 protocols) |
| Live on-chain alert monitoring | `forta.go` | ✅ |
| Reentrancy (EVM) | Slither + Aderyn | ✅ |
| Integer overflow (EVM) | Slither + Aderyn | ✅ |
| Access control | Slither + Aderyn + `timelock.rs` | ✅ |

---

## Known Limitations

1. **Differential Check** — Always returns `true`. Real implementation patches contract via LLM, redeploys, re-runs. Opt-in via `ENABLE_DIFFERENTIAL=true`.
2. **Rescue Lane** — Disabled by design. Would execute white-hat rescues on live protocols. Requires multisig authorization.
3. **x402 Stub Mode** — If `KEEPERHUB_API_KEY` unset, returns placeholder URLs for development.
4. **ETH Price** — Hardcodes $3000 ETH for TVL estimation. Use oracle for production.
5. **Solana tool availability** — All 9 Solana analyzers are optional. Install with the commands in `.env`.
6. **Trident fuzz time** — Default 30 seconds per target (`TRIDENT_FUZZ_SECS`). Increase for deeper coverage.
7. **Dual-Node Bootstrap** — `AXL_PEERS_FOR_NODE_A` and `AXL_PEERS_FOR_NODE_B` require one-time peer key exchange after first run.

---

## Recently Completed

### Solana Vulnerable Vault Test Fixture (2026-05-11)
- Added `test-fixtures/solana/vulnerable-vault/` — fully runnable Anchor workspace for pipeline testing
- **Program** (`lib.rs`): 8 deliberate vulnerabilities covering every class the Solana analyzer detects
- **PoC** (`poc/exploit.ts`): TypeScript client demonstrating all 8 exploits against `solana-test-validator`
- **Scaffold**: `package.json`, `tsconfig.json`, `Makefile`, `tests/vulnerable-vault.ts` — `make exploit` builds and runs end-to-end
- Analogous role to `VulnerableVault.sol` in the EVM pipeline; wired into `make build-vulnerable-vault`

### EVM DeFi Checklist Gaps Closed (2026-05-11)
- **`timelock.rs`** — detects admin functions (`setFee`, `withdraw`, `upgrade`, etc.) unprotected by `TimelockController`
- **`oracle_check.rs`** — 6 Chainlink/Uniswap oracle safety checks (staleness, negative price, spot-price manipulation)
- **`bridge_deps.rs`** — 7 bridge protocols; flags missing message-authentication checks
- **`forta.go`** (Scout) — live Forta Network alert feed; polls every 5 min; graceful no-op without `FORTA_API_KEY`
- Full checklist mapping: `docs/adevar-labs-checklist.md`

### Solana Program Scanning (2026-05-11)
- Added `bountyType: "solana-program"` as a first-class target type alongside EVM and Firedancer
- **Scout**: extended `WatchConfig` with `solana_repos` / `solana_bounty_type` fields; second GitHub watcher goroutine watches 8 high-value Solana program repos
- **Auditor**: 9 new Rust analyzers — cargo-audit, cargo-deny, cargo-geiger, clippy-solana, solana-patterns, semgrep, soteria, anchor-idl, trident
- **Orchestrator**: `verify/solana.go` — `cargo build-sbf` build, `solana-test-validator` deployment, TypeScript/JS PoC execution, impact classification
- **Prompts**: `exploit_solana_v1.md` — covers 7 Solana vuln classes with metadata contract
- **Bounty manifest**: `bounties/solana_programs.yaml` — 10 repos, full severity map, tool list

### Firedancer Bounty Support (prior)
- Bounty manifest (`bounties/firedancer.yaml`) encoding all rules from assets/scope/info docs
- C analyzers: cppcheck, clang-tidy, solfuzz harness runner (7 targets)
- Cluster verifier with GCC enforcement, attacker model validation, modified-validator check
- Known issues checker fetching all 20+ tracker URLs
- Scope monitor for mid-contest changelog polling

### Dual-Node AXL Mesh
- Node A (port 9002): Scout + Orchestrator
- Node B (port 9003): Auditor
- Separate ed25519 keypairs, peer key exchange documented

### 0G Storage & iNFT
- Official 0G Go SDK v1.3.0 with SHA-256 fallback
- `SwarmINFT.sol` deployed on 0G Galileo Testnet: `0x9756eD45Fe95d53b0d72F5efe2977Df1c876089c`
- Go bindings generated from ABI, wired into disclosure pipeline

---

## Build Commands

```bash
make build                   # Build all components
make test                    # Run all tests (Go + Rust + Solidity)
make roundtrip               # Test Go ↔ Rust schema
make run-dashboard           # Launch terminal dashboard
make clean                   # Clean all build artifacts
make build-vulnerable-vault  # Build the deliberate-vuln Anchor test fixture
```

## Environment Setup

```bash
# Option A: Nix (fully reproducible)
nix develop

# Option B: Manual
make setup   # Checks tools, auto-installs aderyn if missing
make build
```

Copy `.env` → `.env.local` and fill in API keys before running agents.
