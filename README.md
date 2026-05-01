# Autonomous Web3 Security Swarm

An agentic mesh of security bots that autonomously discovers, analyzes, verifies, and discloses smart-contract vulnerabilities.

## Quick Start

```bash
make build
make demo
```

## Architecture

```
┌──────────┐     ┌──────────────┐     ┌──────────┐
│  Scout   │────▶│  AXL Mesh    │────▶│ Auditor  │
│ (Go)     │     │  pub/sub     │     │ (Rust)   │
└──────────┘     └──────────────┘     └──────────┘
                                          │
                                          ▼
                                   ┌──────────────┐
                                   │ Orchestrator │
                                   │   (Go)       │
                                   └──────────────┘
```

- **Scout** — Listens to mempool, GitHub, Immunefi for new targets
- **Auditor** — Runs Aderyn + Slither static analysis
- **Orchestrator** — LLM exploit generation, Foundry verification, x402 reports

## Hackathon Stack

- [Gensyn AXL](https://gensyn.ai) — decentralized mesh
- [KeeperHub](https://keeperhub.io) — x402 payments
- [0G](https://0g.ai) — iNFT sovereignty

## Demo Video

See `docs/demo.mp4` (coming soon).
