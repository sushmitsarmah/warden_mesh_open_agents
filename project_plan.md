## Improved Project Plan: Autonomous Web3 Security Swarm

### Reframed Value Proposition

The original plan tries to do two things at once — emergency fund rescue and bounty disclosure — and the rescue path drags legal and MEV risk into what should be an information product. The cleaner framing is: **the swarm is an autonomous bug bounty hunter that produces verified, exploit-proven vulnerability reports and sells them to protocol teams via x402, with an opt-in rescue lane only for protocols that have pre-signed authorization.** The bounty path is the main loop; rescue is a premium feature for whitelisted partners. This makes the system defensible, monetizable, and demo-able.

### Architectural Simplification

The Go-orchestrator + Rust-auditor split is justified only if the Rust analysis is genuinely CPU-bound and parallel enough to need it. For a hackathon, running Aderyn (which is already a Rust binary) as a subprocess from Go gives you the same performance benefit without maintaining two codebases and an inter-language IPC layer over AXL. I'd keep the Go/Rust separation only as a logical agent boundary on the AXL mesh — both written in Go, with Aderyn shelled out — unless you have a Rust specialist on the team. The key insight is that AXL doesn't care what language the nodes speak; the mesh is the integration point.

### Five Layers, Reorganized

**Layer 1 — Discovery (Go).** A Scout Agent watches three feeds in parallel: pending contract deployments via `go-ethereum`'s `ethclient` subscriptions, GitHub commit webhooks for a curated list of DeFi repos, and Immunefi/Sherlock scope additions via their public APIs. Each surface produces a normalized `Target` struct (chain, address-or-repo, source-or-bytecode, severity prior, discovery timestamp). A small priority queue scores targets by TVL × novelty × time-since-last-audit, so the swarm doesn't waste analysis budget on low-value contracts.

**Layer 2 — Coordination (Gensyn AXL).** Targets are gossiped over the AXL mesh as signed JSON envelopes. Critically, the mesh isn't just transport — it's where deduplication and consensus happen. Multiple Scout nodes finding the same target should converge on one investigation, not five. Use AXL's pub/sub topics: `targets/discovered`, `analysis/findings`, `exploit/verified`, `disclosure/published`. Every agent subscribes to the topics relevant to its role.

**Layer 3 — Analysis (Rust agent + Aderyn).** The Auditor Agent fetches live state with `alloy-rs`, runs Aderyn for static patterns, and runs a second pass with Slither (via subprocess) for cross-validation. Findings only graduate to Layer 4 if **both tools flag the same issue class** OR if a single tool flags a high-severity pattern (reentrancy on a function with `call.value`, unprotected `selfdestruct`, missing initializer guards). This dual-source check is the false-positive filter the original plan was missing.

**Layer 4 — Verification (LLM + Foundry, with a critical addition).** When a finding clears Layer 3, an LLM writes a Foundry PoC. But the verification isn't just "does the test pass" — it's a three-way check:
- The exploit must pass against a fork at the **current block** (proves it works on real state).
- The exploit must fail against the same contract with the suspected vulnerable function reverted to a safe pattern (proves the exploit depends on the actual bug, not test-setup artifacts).
- The drain amount must exceed a configurable threshold (filters trivial findings).

Only triple-verified exploits move to Layer 5. The LLM gets up to N retries with the failure trace fed back as context.

**Layer 5 — Action.** Two lanes, gated on the target's authorization status:
- **Bounty Lane (default):** Generate a redacted public summary and a full PoC. The full report is gated behind an x402 paywall via KeeperHub. Protocol teams pay in stablecoins to unlock. Simultaneously, file via Immunefi's API if the protocol has a program. This is the legally clean monetization path.
- **Rescue Lane (opt-in only):** For protocols on a pre-signed authorization whitelist, the verified exploit transaction is rewritten as a rescue (drain to a protocol-controlled safe address) and submitted privately via KeeperHub/Flashbots. **Without a signed authorization on file, this lane is hard-disabled.**

**Layer 6 — Sovereignty (0G iNFT).** The ERC-7857 iNFT on 0G stores the swarm's persistent state: cumulative bounties earned, reports published, authorized protocols, and a pointer to encrypted operational memory (past findings, blacklisted false-positive patterns). Treasury revenue from x402 flows to the iNFT's controlled address; outflows pay for LLM credits and node hosting. The iNFT is also the swarm's reputation primitive — anyone can verify its track record onchain before paying for a report.

### Missing Pieces I'd Add

A **kill switch** controlled by a multisig that can pause all onchain actions if the swarm misbehaves. A **rate limiter** per protocol so the swarm doesn't spam bounty submissions. An **explicit disclosure window** (e.g., 90 days) before any vulnerability is made public, matching industry norms. **Logging to a tamper-evident store** (the 0G memory pointer is a natural fit) so every action is auditable.

### Hackathon Scope Cut

Realistically, for a 24–48 hour hackathon, you should demo Layers 1, 2, 3, 4, and the bounty half of Layer 5, plus a stub iNFT. The rescue lane is a "future work" slide. Trying to demo a live private-mempool rescue with real funds is asking for a runtime failure on stage.

---

## Step-by-Step Implementation Plan for an Agent

This is sequenced so each step produces a runnable artifact, with verification before moving on. An agent (or a developer) should not advance until the checkpoint passes.

**Step 1 — Repository scaffolding and config.** Create a monorepo with `services/scout-go`, `services/auditor-rs`, `services/orchestrator-go`, `contracts/` (for the iNFT), `prompts/`, and `infra/`. Define a shared `Target` and `Finding` schema as Protobuf or JSON Schema in a `proto/` directory so Go and Rust agree on wire format. Set up a `.env.example` with placeholders for RPC URLs, LLM API keys, AXL bootstrap nodes, KeeperHub credentials, and 0G RPC. *Checkpoint:* `make build` produces binaries for both services.

**Step 2 — Scout Agent: mempool listener.** In `scout-go`, implement an `ethclient` subscription to pending transactions on one EVM chain (start with Sepolia for safety). Filter for contract creations and high-value transfers. Emit a `Target` to stdout as JSON. *Checkpoint:* run for 60 seconds against Sepolia and observe at least one `Target` printed.

**Step 3 — Scout Agent: GitHub and Immunefi feeds.** Add two more goroutines: one polling GitHub's GraphQL API for new commits on a configurable repo list, one polling Immunefi's scope API. Normalize all three feed outputs to the same `Target` schema. Add the priority-scoring function. *Checkpoint:* a unit test that feeds synthetic events through all three sources and asserts the priority queue orders them correctly.

**Step 4 — AXL mesh integration.** Spin up a local AXL node in `scout-go`. Publish discovered `Target`s to the `targets/discovered` topic. Write a tiny `axl-tail` debug binary that subscribes to all topics and prints messages — this is your observability tool for the rest of the build. *Checkpoint:* run two scout nodes locally; both should see each other's published targets via `axl-tail`.

**Step 5 — Auditor Agent skeleton (Rust).** In `auditor-rs`, implement an AXL subscriber on `targets/discovered`. On message receipt, use `alloy-rs` to fetch bytecode and source (via Etherscan API for verified contracts) and write to a temp directory. *Checkpoint:* publish a synthetic target from `axl-tail`; observe the Rust agent fetch and persist the contract source.

**Step 6 — Static analysis pipeline.** Wire Aderyn and Slither as subprocess calls from the Auditor. Parse their JSON outputs into the shared `Finding` schema. Implement the dual-source filter (only emit findings flagged by both tools, or single-tool findings of severity ≥ HIGH). Publish to `analysis/findings`. *Checkpoint:* run against a known-vulnerable contract from the Damn Vulnerable DeFi suite; verify a known reentrancy is reported and a known false-positive pattern is suppressed.

**Step 7 — Orchestrator and LLM exploit generation.** In `orchestrator-go`, subscribe to `analysis/findings`. For each finding, build a structured prompt containing the source, the finding, and a Foundry test template. Call the LLM. Write the response to `exploits/{finding_id}.t.sol`. *Checkpoint:* for the Damn Vulnerable contract, the LLM produces a `.t.sol` that compiles under `forge build`.

**Step 8 — Foundry triple-verification.** Implement a `verify.sh` (or a Go wrapper) that: (a) forks the chain at current block via `anvil --fork-url`, (b) runs `forge test` against the LLM exploit, (c) patches the suspected vulnerable function with a safe stub and re-runs, (d) checks drain amount exceeds threshold. Only emit `exploit/verified` if all three pass. Add a retry loop: on (a)-failure, feed the trace back to the LLM up to 3 times. *Checkpoint:* end-to-end run from synthetic target to verified exploit on Damn Vulnerable DeFi without human intervention.

**Step 9 — Bounty report generation and x402 gating.** Build a report generator that produces two artifacts per verified exploit: a public teaser (vulnerability class, affected contract, severity, no PoC) and a full report (PoC, mitigation, references). Integrate KeeperHub's x402 SDK to gate the full report behind a stablecoin payment URL. *Checkpoint:* the teaser is publicly fetchable; the full URL returns 402 until paid; after a test payment, it returns the full report.

**Step 10 — 0G iNFT deployment.** Write the ERC-7857 contract: stores cumulative stats, an authorized-protocols list (settable by multisig), and an encrypted-memory pointer. Deploy to 0G testnet. Add a Go client in the orchestrator that pushes a state update on every `disclosure/published` event. *Checkpoint:* deploy, push three synthetic disclosure updates, read the iNFT state and confirm all three are recorded.

**Step 11 — Kill switch and rate limiting.** Add a multisig-controlled pause flag on the iNFT. The orchestrator reads this flag before any outbound action (LLM call, x402 publication, transaction submission). Implement per-protocol rate limits in the orchestrator's local store. *Checkpoint:* trigger the pause from the multisig; verify the orchestrator halts within one polling interval.

**Step 12 — Rescue lane (optional, only if scope allows).** Behind a feature flag and authorization-whitelist check, implement transaction submission via KeeperHub's Flashbots integration. **Test exclusively on a local Anvil fork.** Do not enable on mainnet for the demo. *Checkpoint:* on a forked vulnerable contract, the rescue tx executes privately and funds land in the configured safe address.

**Step 13 — Demo orchestration.** Build a single `make demo` target that: starts AXL nodes, scout, auditor, orchestrator, deploys the vulnerable contract to a local Anvil, and lets the swarm discover-analyze-verify-publish end-to-end while a TUI or web dashboard shows AXL message flow in real time. The dashboard is critical for the hackathon presentation — judges need to *see* the swarm thinking.

**Step 14 — Documentation and threat model.** Write a `SECURITY.md` covering the legal authorization model, the kill switch, the disclosure window, and the explicit list of what the swarm will not do. This document is what separates this project from "an autonomous exploiter" in the eyes of judges and any future users.

The critical-path demo is steps 1–9 and 13. Steps 10–11 give the project its sovereignty narrative. Step 12 is genuinely optional and arguably should be cut. Step 14 is what makes the project shippable beyond the hackathon.

If you'd like, I can turn this into a downloadable Markdown or Word doc, or expand any specific layer (the Foundry verification logic and the AXL topic schema are the two places where most teams underestimate the work).