# Security Policy

## Authorization Model

- **Rescue lane requires explicit multisig authorization per protocol.** Without authorization, the rescue lane is hard-disabled.
- The swarm is a bounty hunter by default, not a white-hat rescuer.

## Disclosure Window

- Full PoC stays paywalled for `DISCLOSURE_WINDOW_DAYS` (default 90).
- Protocol teams can pay to unlock immediately.
- Public release only after the window expires.

## Kill Switch

- Any multisig signer can call `setPaused(true)` on the iNFT.
- The orchestrator polls this flag every 30 seconds.
- If paused, all outbound actions are dropped.

## Rate Limits

- Maximum 3 disclosures per protocol per 24h window.

## What This Swarm Will NOT Do

1. Execute exploits on unauthorized contracts.
2. Submit findings without triple verification.
3. Reveal PoCs publicly inside the disclosure window.
4. Operate on mainnet without safety checks.

## Reporting Issues

Contact: security@example.com
