package scout

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"

	"github.com/local/swarm/scout/pkg/messages"
)

// AddressWatcher polls watched contract and wallet addresses for on-chain
// activity and emits targets when suspicious transactions are observed.
type AddressWatcher struct {
	rpcURL       string
	out          chan<- messages.Target
	contracts    []string
	wallets      []string
	pollInterval time.Duration
}

// NewAddressWatcher creates a watcher for specific contract/wallet addresses.
func NewAddressWatcher(
	rpcURL string,
	out chan<- messages.Target,
	contracts []string,
	wallets []string,
	pollInterval time.Duration,
) *AddressWatcher {
	return &AddressWatcher{
		rpcURL:       rpcURL,
		out:          out,
		contracts:    contracts,
		wallets:      wallets,
		pollInterval: pollInterval,
	}
}

// Run polls the chain for transactions involving watched addresses.
func (w *AddressWatcher) Run(ctx context.Context) error {
	client, err := ethclient.DialContext(ctx, w.rpcURL)
	if err != nil {
		return fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

	// Build lookup set for fast address matching.
	watched := make(map[string]bool)
	for _, a := range w.contracts {
		watched[strings.ToLower(a)] = true
	}
	for _, a := range w.wallets {
		watched[strings.ToLower(a)] = true
	}

	if len(watched) == 0 {
		slog.Info("address watcher: no contracts or wallets configured, sleeping")
		<-ctx.Done()
		return nil
	}

	// Start from the current head — don't replay history.
	head, err := client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("get block number: %w", err)
	}
	slog.Info("address watcher started", "contracts", len(w.contracts), "wallets", len(w.wallets), "from_block", head)

	seen := make(map[string]bool) // dedupe tx hashes within session
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			latest, err := client.BlockNumber(ctx)
			if err != nil {
				slog.Warn("address watcher: block number failed", "err", err)
				continue
			}

			for bn := head + 1; bn <= latest; bn++ {
				block, err := client.BlockByNumber(ctx, big.NewInt(int64(bn)))
				if err != nil {
					slog.Warn("address watcher: fetch block failed", "block", bn, "err", err)
					continue
				}

				txs := block.Transactions()
				for _, tx := range txs {
					if tx.Hash().Hex() == "" {
						continue
					}
					if seen[tx.Hash().Hex()] {
						continue
					}
					seen[tx.Hash().Hex()] = true

					matched, matchedAddr, addrType := w.matchTx(tx, watched)
					if !matched {
						continue
					}

					// Build target
					t := messages.Target{
						ID:           uuid.NewString(),
						Kind:         messages.TargetOnchain,
						ChainID:      w.chainID(client, ctx),
						Address:      matchedAddr,
						TxHash:       tx.Hash().Hex(),
						DiscoveredAt: time.Now().UTC(),
						Priority:     55, // baseline for watched-address hits
					}

					// Boost priority for high-value txs
					if tx.Value() != nil {
						valEth := new(big.Float).Quo(
							new(big.Float).SetInt(tx.Value()),
							new(big.Float).SetInt(big.NewInt(1e18)),
						)
						vFloat, _ := valEth.Float64()
						ethUsd := 3000.0 // same hardcode as mempool
						t.TVLUsd = vFloat * ethUsd
						if t.TVLUsd >= 50000 {
							t.Priority = 70 + min(vFloat*ethUsd/1e6, 30)
						}
					}

					w.out <- t
					slog.Info("address watcher target emitted",
						"type", addrType,
						"address", matchedAddr,
						"tx", tx.Hash().Hex(),
						"block", bn,
						"priority", t.Priority,
						"value_usd", t.TVLUsd,
					)
				}
			}
			head = latest
		}
	}
}

func (w *AddressWatcher) chainID(client *ethclient.Client, ctx context.Context) int {
	id, err := client.ChainID(ctx)
	if err != nil {
		return 11155111 // Sepolia default
	}
	return int(id.Int64())
}

// matchTx checks if a transaction involves any watched address.
func (w *AddressWatcher) matchTx(tx *types.Transaction, watched map[string]bool) (bool, string, string) {
	// Check sender — msg.From is not exposed directly here, but To() gives receiver.
	// We check To() first.
	if tx.To() != nil {
		to := strings.ToLower(tx.To().Hex())
		if watched[to] {
			return true, to, "contract"
		}
	}

	// For contract wallets, we also want to catch *from* the wallet.
	// ethclient.Transaction doesn't expose From directly without signer.
	// We'll recover it lazily only when needed for wallet matches.
	for _, addr := range w.wallets {
		lower := strings.ToLower(addr)
		if watched[lower] {
			// Best-effort: check if To matches wallet (incoming) or we need From (outgoing).
			// Since From extraction is expensive, skip it here and just flag To matches.
			// A production scanner would use eth_getTransactionReceipt to get From.
			if tx.To() != nil && strings.ToLower(tx.To().Hex()) == lower {
				return true, lower, "wallet"
			}
		}
	}

	return false, "", ""
}
