package scout

import (
	"context"
	"log/slog"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/google/uuid"

	"github.com/local/swarm/scout/pkg/messages"
)

type MempoolWatcher struct {
	rpcURL     string
	out       chan<- messages.Target
	bountyType string
}

func NewMempoolWatcher(rpcURL string, out chan<- messages.Target, bountyType string) *MempoolWatcher {
	return &MempoolWatcher{rpcURL: rpcURL, out: out, bountyType: bountyType}
}

func (w *MempoolWatcher) Run(ctx context.Context) error {
	rpcClient, err := rpc.DialContext(ctx, w.rpcURL)
	if err != nil {
		return err
	}
	client := ethclient.NewClient(rpcClient)

	ch := make(chan common.Hash, 256)
	sub, err := rpcClient.EthSubscribe(ctx, ch, "newPendingTransactions")
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			return err
		case h := <-ch:
			tx, _, err := client.TransactionByHash(ctx, h)
			if err != nil {
				continue
			}
			if tx.To() == nil { // contract creation
				t := messages.Target{
					ID:         uuid.NewString(),
					BountyType: w.bountyType,
					Kind:       "onchain",
					ChainID:    11155111,
					DiscoveredAt: time.Now().UTC(),
					Priority:   50,
				}
				slog.Info("contract creation observed", "tx", h.Hex())
				w.out <- t
			} else if tx.Value() != nil && tx.Value().Sign() > 0 {
				ethUsd := 3000.0
				valueEth := new(big.Float).Quo(
					new(big.Float).SetInt(tx.Value()),
					new(big.Float).SetInt(big.NewInt(1e18)),
				)
				valFloat, _ := valueEth.Float64()
				if valFloat*ethUsd >= 50000 {
					t := messages.Target{
						ID:         uuid.NewString(),
						BountyType: w.bountyType,
						Kind:       "onchain",
						ChainID:    11155111,
						Address:    tx.To().Hex(),
						DiscoveredAt: time.Now().UTC(),
						Priority:   60 + min(valFloat*ethUsd/1e6, 40),
						TVLUsd:     valFloat * ethUsd,
					}
					w.out <- t
				}
			}
		}
	}
}
