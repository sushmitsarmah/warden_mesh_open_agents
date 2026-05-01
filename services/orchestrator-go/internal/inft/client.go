package inft

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// SwarmINFT is a placeholder — generate bindings from SwarmINFT.sol ABI
type SwarmINFT struct{}

type Client struct {
	mu         sync.Mutex
	transactor *bind.TransactOpts
	contract   *SwarmINFT
	client     *ethclient.Client
}

func NewClient(rpcURL, contractAddr, privateKeyHex string) (*Client, error) {
	cl, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	pk, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}
	chainID, err := cl.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	transactor, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		return nil, err
	}
	_ = contractAddr
	return &Client{
		transactor: transactor,
		client:     cl,
	}, nil
}

func (c *Client) RecordDisclosure(ctx context.Context, tokenID, bountyUsd int64, memoryDelta [32]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = tokenID
	_ = bountyUsd
	_ = memoryDelta
	return nil
}

func (c *Client) IsPaused() (bool, error) {
	return false, nil
}

func (c *Client) IsAuthorized(protocol string) bool {
	_ = protocol
	return false
}