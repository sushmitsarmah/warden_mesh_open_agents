.PHONY: build test clean scout auditor orchestrator contracts \
  roundtrip test-analyzers test-exploit-gen setup check-tools dashboard

# ── Self-healing setup ──────────────────────────────────────────────
setup: check-tools
	@echo "Setup complete. Run 'make build'."

# Check that every external tool we need is present; auto-install if missing.
check-tools:
	@bash infra/scripts/check-tools.sh

# ── Main build ──────────────────────────────────────────────────────
build: check-tools scout auditor orchestrator contracts dashboard
	@echo "All components built successfully."

scout:
	cd services/scout-go && mkdir -p bin && go build -o bin/ ./...

orchestrator:
	cd services/orchestrator-go && mkdir -p bin && go build -o bin/ ./...

auditor:
	cd services/auditor-rs && cargo build --release

contracts:
	cd contracts && forge build

dashboard:
	cd services/dashboard/server && go build -o bin/dashboard .

# ── Run ─────────────────────────────────────────────────────────────
run-dashboard:
	cd services/dashboard/server && go run .

# ── Tests ───────────────────────────────────────────────────────────
test: check-tools
	cd services/scout-go && go test ./...
	cd services/orchestrator-go && go test ./...
	cd services/auditor-rs && cargo test
	cd contracts && forge test

roundtrip:
	bash infra/scripts/schema-roundtrip.sh

# ── Component tests ─────────────────────────────────────────────────
test-analyzers:
	cd services/auditor-rs && cargo run --bin test-analyzers

test-exploit-gen:
	cd services/orchestrator-go && go build -o bin/test-exploit-gen ./cmd/test-exploit-gen && ./bin/test-exploit-gen

# ── Clean ───────────────────────────────────────────────────────────
clean:
	cd services/auditor-rs && cargo clean
	cd contracts && forge clean
	rm -rf services/*/bin
