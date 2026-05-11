.PHONY: build install exploit clean

# Build the Solana program (produces target/deploy/vulnerable_vault.so)
build:
	anchor build

# Install TypeScript dependencies
install:
	yarn install --frozen-lockfile

# Run the PoC exploit against a running solana-test-validator
# Starts the validator, waits for it to be ready, then runs the exploit.
exploit: build install
	@echo "Starting solana-test-validator..."
	solana-test-validator \
		--bpf-program VuLnVauLt111111111111111111111111111111111 \
		target/deploy/vulnerable_vault.so \
		--reset \
		--quiet &
	@sleep 5
	ANCHOR_PROVIDER_URL=http://127.0.0.1:8899 yarn ts-node poc/exploit.ts
	@pkill -f solana-test-validator || true

# Run Anchor integration tests
test: build install
	anchor test

clean:
	cargo clean
	rm -rf node_modules dist .anchor test-ledger
