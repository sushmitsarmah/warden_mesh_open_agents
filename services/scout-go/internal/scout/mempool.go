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
	rpcURL string
	out    chan<- messages.Target
}

func NewMempoolWatcher(rpcURL string, out chan<- messages.Target) *MempoolWatcher {
	return &MempoolWatcher{rpcURL: rpcURL, out: out}
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
					ID:           uuid.NewString(),
					Kind:         messages.TargetOnchain,
					ChainID:      11155111, // Sepolia
					DiscoveredAt: time.Now().UTC(),
					Priority:     50, // default; refined later
				}
				slog.Info("contract creation observed", "tx", h.Hex())
				w.out <- t
			} else if tx.Value() != nil && tx.Value().Sign() > 0 {
				// crude USD estimate: assume ETH price hardcoded for hackathon; plug oracle later
				ethUsd := 3000.0
				valueEth := new(big.Float).Quo(
					new(big.Float).SetInt(tx.Value()),
					new(big.Float).SetInt(big.NewInt(1e18)),
				)
				valFloat, _ := valueEth.Float64()
				if valFloat*ethUsd >= 50000 { // tunable
					t := messages.Target{
						ID:           uuid.NewString(),
						Kind:         messages.TargetOnchain,
						ChainID:      11155111,
						Address:      tx.To().Hex(),
						DiscoveredAt: time.Now().UTC(),
						Priority:     60 + min(valFloat*ethUsd/1e6, 40),
						TVLUsd:       valFloat * ethUsd,
					}
					w.out <- t
				}
			}
		}
	}
}
