#!/usr/bin/env bash
set -euo pipefail
REPO="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO/services/scout-go"
go run ./cmd/dump-target > /tmp/target-roundtrip.json
cd "$REPO"
cargo run --manifest-path services/auditor-rs/Cargo.toml --bin roundtrip-check < /tmp/target-roundtrip.json
