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

type AddressWatcher struct {
	rpcURL     string
	out       chan<- messages.Target
	contracts []string
	wallets   []string
	bountyType string
	pollInterval time.Duration
}

func NewAddressWatcher(
	rpcURL string,
	out chan<- messages.Target,
	contracts []string,
	wallets []string,
	bountyType string,
	pollInterval time.Duration,
) *AddressWatcher {
	return &AddressWatcher{
		rpcURL:      rpcURL,
		out:         out,
		contracts:   contracts,
		wallets:     wallets,
		bountyType:  bountyType,
		pollInterval: pollInterval,
	}
}

func (w *AddressWatcher) Run(ctx context.Context) error {
	client, err := ethclient.DialContext(ctx, w.rpcURL)
	if err != nil {
		return fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

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

	head, err := client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("get block number: %w", err)
	}
	slog.Info("address watcher started", "contracts", len(w.contracts), "wallets", len(w.wallets), "from_block", head)

	seen := make(map[string]bool)
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

					t := messages.Target{
						ID:           uuid.NewString(),
						BountyType:   w.bountyType,
						Kind:         "onchain",
						ChainID:      w.chainID(client, ctx),
						Address:      matchedAddr,
						TxHash:       tx.Hash().Hex(),
						DiscoveredAt: time.Now().UTC(),
						Priority:     55,
					}

					if tx.Value() != nil {
						valEth := new(big.Float).Quo(
							new(big.Float).SetInt(tx.Value()),
							new(big.Float).SetInt(big.NewInt(1e18)),
						)
						vFloat, _ := valEth.Float64()
						ethUsd := 3000.0
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
		return 11155111
	}
	return int(id.Int64())
}

func (w *AddressWatcher) matchTx(tx *types.Transaction, watched map[string]bool) (bool, string, string) {
	if tx.To() != nil {
		to := strings.ToLower(tx.To().Hex())
		if watched[to] {
			return true, to, "contract"
		}
	}

	for _, addr := range w.wallets {
		lower := strings.ToLower(addr)
		if watched[lower] {
			if tx.To() != nil && strings.ToLower(tx.To().Hex()) == lower {
				return true, lower, "wallet"
			}
		}
	}

	return false, "", ""
}
