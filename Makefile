.PHONY: build test clean scout auditor orchestrator contracts roundtrip

build: scout auditor orchestrator contracts

scout:
	cd services/scout-go && mkdir -p bin && go build -o bin/ ./...

orchestrator:
	cd services/orchestrator-go && mkdir -p bin && go build -o bin/ ./...

auditor:
	cd services/auditor-rs && cargo build --release

contracts:
	cd contracts && forge build

roundtrip:
	bash infra/scripts/schema-roundtrip.sh

test:
	cd services/scout-go && go test ./...
	cd services/orchestrator-go && go test ./...
	cd services/auditor-rs && cargo test
	cd contracts && forge test

clean:
	cd services/auditor-rs && cargo clean
	cd contracts && forge clean
	rm -rf services/*/bin
