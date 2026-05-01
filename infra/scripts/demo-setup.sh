#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "=== Starting demo setup ==="

# Start anvil
anvil --fork-url "${SEPOLIA_RPC_URL:-http://localhost:8545}" --port 8545 &
ANVIL_PID=$!
sleep 2

# Deploy VulnerableVault
DEPLOY_OUTPUT=$(forge script ../contracts/script/DeployDemo.s.sol --rpc-url http://localhost:8545 --broadcast --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 -vvvv 2>&1 || true)
echo "$DEPLOY_OUTPUT"

# Fund vault
VAULT_ADDR=$(cat /tmp/vault-address.txt 2>/dev/null || echo "0x5FbDB2315678afecb367f032d93F642f64180aa3")
cast send "$VAULT_ADDR" --value 100ether --rpc-url http://localhost:8545 --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

# Fund attacker
cast send 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 --value 1ether --rpc-url http://localhost:8545 --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

echo "VulnerableVault deployed at: $VAULT_ADDR"
echo "Anvil PID: $ANVIL_PID"
