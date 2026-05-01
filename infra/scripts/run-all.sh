#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."

echo "=== Launching swarm ==="

# Ensure logs dir
mkdir -p logs

# Start AXL nodes (placeholder)
echo "AXL: using in-memory mock" > logs/axl.log

# Start scout
nohup go run ./services/scout-go/cmd > logs/scout.log 2>&1 &
SCOUT_PID=$!

# Start auditor
nohup cargo run --manifest-path services/auditor-rs/Cargo.toml --bin auditor > logs/auditor.log 2>&1 &
AUDITOR_PID=$!

# Start orchestrator
nohup go run ./services/orchestrator-go/cmd > logs/orchestrator.log 2>&1 &
ORCH_PID=$!

# Start dashboard
nohup go run ./services/dashboard/server > logs/dashboard.log 2>&1 &
DASH_PID=$!

echo "PIDs: scout=$SCOUT_PID auditor=$AUDITOR_PID orchestrator=$ORCH_PID dashboard=$DASH_PID"

# Inject demo target
go run ./services/scout-go/cmd/inject --address=0x5FbDB2315678afecb367f032d93F642f64180aa3 --chain=31337 --priority=95 --tvl=300000

echo "Dashboard: http://localhost:8080"

# Cleanup trap
trap "kill $SCOUT_PID $AUDITOR_PID $ORCH_PID $DASH_PID 2>/dev/null; pkill -f anvil" EXIT
wait
