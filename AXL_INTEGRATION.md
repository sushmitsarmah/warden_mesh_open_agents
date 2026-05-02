# AXL Mesh Integration

The project now uses **Gensyn AXL** as the decentralized communication layer between agents.

## Architecture

Each agent (Scout, Auditor, Orchestrator) runs as an AXL client that connects to a local AXL node via HTTP API. Messages are routed through the AXL mesh using a broadcast pub/sub pattern.

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Scout     │      │  Auditor    │      │Orchestrator │
│   (Go)      │      │  (Rust)     │      │    (Go)     │
└──────┬──────┘      └──────┬──────┘      └──────┬──────┘
       │                    │                     │
       │  HTTP API          │  HTTP API           │  HTTP API
       │                    │                     │
       └────────────────────┼─────────────────────┘
                           │
                    ┌──────┴──────┐
                    │  AXL Node   │
                    │   (P2P)     │
                    └─────────────┘
```

## Topics

- **`targets/discovered`** — Scout publishes new contract targets
- **`analysis/findings`** — Auditor publishes vulnerability findings
- **`exploit/verified`** — Orchestrator publishes verified exploits

## Configuration

### 1. Start the AXL Node

```bash
cd /path/to/axl
./node -config node-config.json
```

Your AXL node will print its public key:
```
Your Public Key: 4181e7e5adfab772ca50b946007ab34ae4f60b9295012c402ce5f3c5306b4db4
```

### 2. Configure Peer Keys

Each agent needs to know the public keys of other agents to communicate. Add them to `.env`:

```bash
# Single local AXL node (all agents connect to same node)
AXL_API_URL=http://127.0.0.1:9002
AXL_PEER_KEYS=

# Multiple agents on different machines
AXL_PEER_KEYS=4181e7e5adfab772ca50b946007ab34ae4f60b9295012c402ce5f3c5306b4db4,another_peer_key_here
```

**Note:** If all agents run on the same machine and connect to the same AXL node, leave `AXL_PEER_KEYS` empty. The agents will communicate via the local node's internal routing.

### 3. Run the Agents

```bash
# Terminal 1: Scout
cd services/scout-go
./bin/scout

# Terminal 2: Auditor
cd services/auditor-rs
cargo run --bin auditor

# Terminal 3: Orchestrator
cd services/orchestrator-go
./bin/orchestrator
```

## How It Works

### Message Flow

1. **Scout** discovers a new contract and publishes to `targets/discovered`
2. **Auditor** subscribes to `targets/discovered`, analyzes the contract, and publishes findings to `analysis/findings`
3. **Orchestrator** subscribes to `analysis/findings`, generates exploits, and publishes to `exploit/verified`

### Broadcast Pattern

Each agent maintains a list of peer public keys. When publishing:
1. Wrap the payload in a `Message` struct with topic + payload
2. JSON serialize the message
3. Send to each peer via `POST /send` with `X-Destination-Peer-Id` header

When receiving:
1. Poll `GET /recv` every 100ms
2. Deserialize the message and extract topic
3. Route to topic-specific subscribers

### Local Development

For local testing with a single AXL node:
- All agents connect to `http://127.0.0.1:9002`
- Set `AXL_PEER_KEYS=""` (empty)
- The AXL node handles local message routing internally

### Multi-Node Setup

For distributed deployment:
1. Each agent runs its own AXL node
2. Exchange public keys between agents
3. Configure `AXL_PEER_KEYS` with comma-separated peer keys
4. Messages broadcast to all peers

## API Reference

### AXL HTTP API (provided by the node)

- **POST /send** — Send message to a peer
  - Header: `X-Destination-Peer-Id: <64-char-hex-key>`
  - Body: raw bytes

- **GET /recv** — Receive pending messages
  - Returns: raw message bytes (or 404 if no messages)

- **GET /topology** — Query network topology
  - Returns: JSON with node info and peers

## Troubleshooting

### No messages received

1. Check AXL node is running: `curl http://127.0.0.1:9002/topology`
2. Verify peer keys are correct (64-character hex)
3. Check agent logs for "AXL client initialized"

### Send failures

- Ensure the destination peer is connected to the mesh
- Check network connectivity between AXL nodes
- Verify peer key is correct

### Messages not routed to subscribers

- Check topic names match exactly (case-sensitive)
- Ensure subscriber is registered before publisher sends
- Check subscriber channel isn't full (buffer size: 100)

## Implementation Details

### Go (Scout, Orchestrator)

```go
// services/scout-go/pkg/axl/axl.go
node, _ := axl.NewNode(peerKeys)
node.Publish("targets/discovered", payload)
ch, _ := node.Subscribe("targets/discovered")
msg := <-ch // Receive message
```

### Rust (Auditor)

```rust
// services/auditor-rs/src/axl.rs
let client = AxlClient::new(peer_keys);
client.publish("analysis/findings", payload).await?;
let mut stream = subscribe_targets().await;
let target = stream.next().await;
```

## Next Steps

Once AXL integration is complete:
1. ✅ Scout publishes real targets from mempool
2. ✅ Auditor consumes targets and publishes findings
3. ✅ Orchestrator consumes findings and generates exploits
4. Test end-to-end with VulnerableVault.sol
5. Deploy to distributed nodes
