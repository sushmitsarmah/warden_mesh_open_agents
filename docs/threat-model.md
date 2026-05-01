# Threat Model

## Trust Assumptions

1. LLM correctness: the LLM may hallucinate exploit code. Mitigated by Foundry verification.
2. AXL availability: the mesh may partition. Mitigated by local buffering.
3. RPC honesty: Ethereum RPCs may lie about state. Mitigated by multiple RPC checks.

## Adversaries

1. **Front-runners** — May intercept disclosed exploits before patch. Mitigated by private mempool routing.
2. **Malicious target contracts** — May trick the analyzer with obfuscation. Mitigated by dual-source analysis.
3. **Prompt injectors** — May embed malicious instructions in source comments. Mitigated by sandboxed prompt context.

## Mitigations

- Content-ID deduplication prevents duplicate work.
- Kill switch halts the swarm if misbehavior is detected.
- Audit log provides tamper-evident history.

## Known Limitations

- False positives: dual-source filter reduces but does not eliminate them.
- LLM cost: each exploit generation costs API credits.
- No formal verification: the swarm is not a replacement for audits.
